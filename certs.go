package hatchcert

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/go-acme/lego/v3/certcrypto"
	"github.com/go-acme/lego/v3/certificate"
)

const ValidityDays = 30

type Cert struct {
	Name       string
	Domains    []string
	AuthMethod string
}

func exp(fname string) (int, error) {
	pemcerts, err := ioutil.ReadFile(fname)
	if err != nil {
		return 0, err
	}
	cs, err := certcrypto.ParsePEMBundle(pemcerts)
	if err != nil {
		return 0, err
	}
	for _, c := range cs {
		if !c.IsCA {
			days := time.Until(c.NotAfter) / (time.Hour * 24)
			if days < 0 {
				days = 0
			}
			return int(days), nil
		}
	}
	return 0, io.EOF
}

func Active(path string, certs []Cert) {
	for _, cert := range certs {
		f := filepath.Join(path, "live", cert.Name, "fullchain")
		days, err := exp(f)
		if err != nil {
			fmt.Fprint(os.Stderr, cert.Name, ": ", f, ": ", err, "\n")
		} else {
			fmt.Print(cert.Name, ": expires in ", days, " day(s)\n")
		}
	}
}

func ScanCerts(path string, certs []Cert) ([]Cert, error) {
	var errors MultiError
	var issue []Cert
	for _, cert := range certs {
		f := filepath.Join(path, "live", cert.Name, "fullchain")
		days, err := exp(f)
		if err != nil {
			if os.IsNotExist(err) {
				issue = append(issue, cert)
			} else {
				errors = append(errors, err)
			}
			continue
		}
		if days < ValidityDays {
			issue = append(issue, cert)
		}
	}
	return issue, errors.Nil()
}

func Issue(a *AccountMeta, cert Cert) error {
	request := certificate.ObtainRequest{
		Domains:    cert.Domains,
		Bundle:     false,
		PrivateKey: nil,
		MustStaple: false,
	}

	o := InterceptOutput()
	defer o.Restore()
	c, err := a.Client.Certificate.Obtain(request)
	if err != nil {
		o.Emit()
		return err
	}
	store, err := storeCert(a.Path, cert.Name, c)
	if err != nil {
		return err
	}
	return updateLinks(a.Path, store, cert.Domains)
}

func storeCert(base, name string, cert *certificate.Resource) (string, error) {
	certs := filepath.Join(base, "certs")
	os.MkdirAll(certs, 0755)

	storerel, err := ioutil.TempDir(certs, name+".")
	if err != nil {
		return "", err
	}
	store, err := filepath.Abs(storerel)
	if err != nil {
		os.Remove(storerel)
		return "", err
	}
	os.Chmod(store, 0755)

	var errors MultiError
	if cert.PrivateKey != nil {
		err := ioutil.WriteFile(filepath.Join(store, "privkey"), cert.PrivateKey, 0644)
		if err != nil {
			errors = append(errors, err)
		}
	}

	var chain []byte
	chain = append(chain, cert.Certificate...)
	chain = append(chain, cert.IssuerCertificate...)
	// In ACMEv2, the issuer is always included even if you don't request a
	// bundle; filter duplicates manually
	chain = dedupCerts(trailingNewline(chain))
	err = ioutil.WriteFile(filepath.Join(store, "fullchain"), chain, 0644)
	if err != nil {
		errors = append(errors, err)
	}

	if cert.CertURL != "" || cert.CertStableURL != "" {
		url := cert.CertStableURL
		if url == "" {
			url = cert.CertURL
		}
		urlb := []byte(url + "\n")
		err := ioutil.WriteFile(filepath.Join(store, "url"), urlb, 0644)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return store, errors.Nil()
}

func updateLinks(base, store string, domains []string) error {
	var errors MultiError
	live := filepath.Join(base, "live")
	os.MkdirAll(live, 0755)
	for _, domain := range domains {
		err := replaceLink(live, store, domain)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors.Nil()
}

func trailingNewline(b []byte) []byte {
	if len(b) > 0 && b[len(b)-1] != '\n' {
		return append(b, '\n')
	}
	return b
}

func dedupCerts(b []byte) (ret []byte) {
	srch := []byte("\n-----END CERTIFICATE-----\n")
	seen := map[string]bool{}
	ptr := b
	for {
		c := bytes.Index(ptr, srch)
		if c < 0 {
			break
		}
		c += len(srch)
		cert := string(ptr[:c])
		ptr = ptr[c:]
		if !seen[cert] {
			seen[cert] = true
			ret = append(ret, cert...)
		}
	}
	ret = append(ret, ptr...)
	return
}
