package hatchcert

import (
	"context"
	"crypto"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mholt/acmez/v3"
	"github.com/mholt/acmez/v3/acme"
)

var (
	// LEDirectoryProduction URL to the Let's Encrypt production
	LEDirectoryProduction = "https://acme-v02.api.letsencrypt.org/directory"

	// LEDirectoryStaging URL to the Let's Encrypt staging
	LEDirectoryStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

var (
	ErrNothingIssued = errors.New("no certificate was issued")
)

// Active shows the user which certificates are available.
func Active(path string, certs []Cert) {
	for _, cert := range certs {
		f := filepath.Join(path, "live", cert.Name, "fullchain")
		certs, err := loadCerts(f)
		if err != nil {
			fmt.Fprint(os.Stderr, cert.Name, ": ", f, ": ", err, "\n")
		} else {
			t := ValidityTime(certs)
			fmt.Print(cert.Name, ": expires in ", formatDuration(t), "\n")
		}
	}
}

type Hatcher struct {
	path   string
	conf   Configuration
	client *acmez.Client
	acct   acme.Account
	// Strictly used to capture details when a failure occurs, unless in
	// verbose mode.
	acmeLogger *Logger
}

func mkclient(conf Configuration, l *slog.Logger) *acmez.Client {
	var hc *http.Client
	server := conf.ACME
	if server == "" || server == "prod" {
		server = LEDirectoryProduction
	} else if server == "staging" {
		server = LEDirectoryStaging
	} else if server == "pebble" {
		server = "https://127.0.0.1:14000/dir"
		hc = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}
	return &acmez.Client{
		Client: &acme.Client{
			Directory:  server,
			HTTPClient: hc,
			Logger:     l,
			UserAgent:  "hatchcert",
		},
		ChallengeSolvers: conf.Solvers,
	}
}

func New(path string, conf Configuration, acmeLogs *slog.Logger) *Hatcher {
	return &Hatcher{
		path:   path,
		conf:   conf,
		client: mkclient(conf, acmeLogs),
	}
}

func (h *Hatcher) AccountInfo() {
	acfile := filepath.Join(h.path, "account")
	if !exists(acfile) {
		fmt.Fprintln(os.Stderr, "Account file does not exist:", acfile)
		return
	}
	var saved SavedAccount
	if err := unmarshal(acfile, &saved); err != nil {
		fmt.Fprintln(os.Stderr, "Account file invalid:", acfile, err)
		return
	}
	// TODO: we SHOULD register the account here
	if saved.AccountKey == "" {
		fmt.Fprintln(os.Stderr, "Account key has not been generated yet")
		return
	}
	key, err := parseKey(saved.AccountKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Account key invalid:", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	if acct, err := h.client.GetAccount(ctx, acme.Account{PrivateKey: key.(crypto.Signer)}); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to obtain account information:", err)
		return
	} else {
		fmt.Println("Status:", acct.Status)
		fmt.Println("Contact:", acct.Contact)
		fmt.Println("Account URL:", acct.Location)
	}
}

// NeedsRenewal returns whether the certificate should be issued NOW based on
// renewal information provided by the ACME server. Fallback expiry detection
// based on NotAfter should still take place, especially in the case of
// unlikely errors.
func (h *Hatcher) NeedsRenewal(cert Cert) (bool, error) {
	if len(cert.Certs) == 0 {
		// We do not have a certificate yet
		return true, nil
	}

	// Live ARI check
	// NOTE: We only run once a day and typical Retry-After is 6 to 24 hours,
	// so there's no need to cache this at this point.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	r, err := h.client.GetRenewalInfo(ctx, cert.Certs[0])
	if err != nil {
		return false, err
	}

	slog.Debug("Fetched renewal information",
		"cert", cert.Name,
		"at", r.SelectedTime)

	now := time.Now()
	renew := !r.SelectedTime.IsZero() && now.Add(time.Hour*24).After(r.SelectedTime)
	return renew, err
}

const HasEnabledReplacing = false

func (h *Hatcher) Issue(cert Cert) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	certPrivateKey, err := generateKey(cert.KeyType)
	if err != nil {
		return err
	}
	csr, err := acmez.NewCSR(certPrivateKey, cert.Domains)
	if err != nil {
		return err
	}

	template := acmez.OrderParameters{
		Account: h.acct,
		Profile: h.conf.Profile,
		CSR:     acmez.StaticCSR(csr),
	}
	if len(cert.Certs) > 0 {
		// Beware: this can make the request fail if the ACME server doesn't
		// like various aspects of the cert ID / AKI.
		if HasEnabledReplacing {
			template.Replaces = cert.Certs[0]
		}
	}

	for _, d := range cert.Domains {
		template.Identifiers = append(template.Identifiers, acme.Identifier{
			Type: "dns", Value: d})
	}

	certs, err := h.client.ObtainCertificate(ctx, template)
	if err != nil && template.Replaces != nil && HasEnabledReplacing {
		// TODO: we should only do this on e.g. a 400 error related to the
		// Replaces field, but this isn't trivial as of yet.
		template.Replaces = nil
		certs, err = h.client.ObtainCertificate(ctx, template)
	}
	if err != nil {
		return err
	} else if len(certs) == 0 {
		return ErrNothingIssued
	}

	encodedPrivKey := encodePrivateKey(certPrivateKey)

	name := cert.Domains[0]
	use := certs[0]

	if cert.PreferredChain != "" && len(certs) > 1 {
		// If the user prefers a certain chain, pick it if it exists. We allow
		// a bit of flexibility to find either the root or intermediary; their
		// names should be enough.
	findpref:
		for _, issued := range certs {
			cs, err := parsePEMBundle(issued.ChainPEM)
			if err == nil {
				for _, c := range cs {
					if c.IsCA && c.Subject.CommonName == cert.PreferredChain {
						use = issued
						break findpref
					}
					if c.IsCA && c.Issuer.CommonName == cert.PreferredChain {
						use = issued
						break findpref
					}
				}
			}
		}
	}

	id, err := storeCert(h.path, name, use.ChainPEM, encodedPrivKey)
	if err != nil {
		return err
	}

	// TODO: take note of renewal info, as rough guideline?
	return updateLinks(h.path, id, cert.Domains)
}

type SavedAccount struct {
	URL        string `json:"account_url,omitempty"`
	Email      string `json:"email"`
	AccountKey string `json:"account_key"`
}

func (h *Hatcher) EnsureAccount() error {
	acfile := filepath.Join(h.path, "account")
	var saved SavedAccount

	if exists(acfile) {
		if err := unmarshal(acfile, &saved); err != nil {
			return nil
		}
	}

	store := false
	account := acme.Account{
		Contact:              []string{"mailto:" + h.conf.Email},
		TermsOfServiceAgreed: true,
	}
	if saved.AccountKey == "" {
		pk, err := generateECKey()
		if err != nil {
			return err
		}
		account.PrivateKey = pk
		saved.Email = h.conf.Email
		saved.AccountKey = string(encodeECKey(pk))
		store = true
	} else {
		key, err := parseKey(saved.AccountKey)
		if err != nil {
			return err
		}
		account.PrivateKey = key.(crypto.Signer)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if acct, err := h.client.NewAccount(ctx, account); err != nil {
		return err
	} else {
		h.acct = acct
	}
	if h.acct.Location != "" && saved.URL != h.acct.Location {
		// This'll only happen on first run
		slog.Info("ACME account registered",
			"url", h.acct.Location)
		saved.URL = h.acct.Location
		store = true
	}
	if store {
		if err := marshal(acfile, saved); err != nil {
			return err
		}
	}
	return nil
}
