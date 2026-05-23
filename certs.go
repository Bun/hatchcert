package hatchcert

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	ValidityUnit             = time.Hour
	TargetValidityPercentage = 30
	ThreeDays                = time.Hour * 24 * 3
)

type Cert struct {
	Name           string
	Domains        []string
	KeyType        string
	PreferredChain string

	Certs   []*x509.Certificate
	Expired bool
}

// ValidityPercentage returns the remaining validity of the first leaf
// certificate in a bundle as a rough percentage. The case where a certificate
// is not valid yet is not considered relevant.
func ValidityPercentage(certs []*x509.Certificate) (time.Duration, int) {
	now := time.Now()
	for _, cert := range certs {
		if !cert.IsCA {
			d := cert.NotAfter.Sub(now)
			if d < 0 {
				d = 0
			}
			ds := int(d / time.Second)
			pct := 0
			if period := int(cert.NotAfter.Sub(cert.NotBefore) / time.Second); period > 0 {
				if pct = 100 * ds / period; pct > 100 {
					pct = 100
				}
			}
			return d, pct
		}
	}
	return 0, 0
}

func loadCerts(fname string) ([]*x509.Certificate, error) {
	pemcerts, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	return parsePEMBundle(pemcerts)
}

func parsePEMBundle(bundle []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	for {
		var block *pem.Block
		block, bundle = pem.Decode(bundle)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("hatchcert: no certificates found in PEM bundle")
	}
	return certs, nil
}

func LoadCerts(path string, certs []Cert) []Cert {
	for i, cert := range certs {
		f := filepath.Join(path, "live", cert.Name, "fullchain")
		cert.Certs, _ = loadCerts(f)
		certs[i] = cert
	}
	return certs
}

// ScanCerts returns a list of desired certificates that either don't exist
// yet, or are going to expire soon.
func ScanCerts(path string, certs []Cert) ([]Cert, error) {
	var errors MultiError
	var issue []Cert
	for _, cert := range certs {
		f := filepath.Join(path, "live", cert.Name, "fullchain")
		certs, err := loadCerts(f)
		if err != nil {
			// Something went wrong, we will consider this cert
			if os.IsNotExist(err) {
				issue = append(issue, cert)
			} else {
				errors = append(errors, err)
			}
			continue
		}
		cert.Certs = certs
		delta, pct := ValidityPercentage(certs)
		if delta <= ThreeDays {
			cert.Expired = true // Helper to skip ARI check
		}
		if pct < TargetValidityPercentage {
			issue = append(issue, cert)
		}
	}
	return issue, errors.Nil()
}

func storeCert(base, name string, certPEM, privPEM []byte) (string, error) {
	certs := filepath.Join(base, "certs")
	os.MkdirAll(certs, 0755)

	storerel, err := os.MkdirTemp(certs, name+".")
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

	if len(privPEM) > 0 {
		if err := os.WriteFile(filepath.Join(store, "privkey"), privPEM, 0644); err != nil {
			errors = append(errors, err)
		}
	}

	chain := trailingNewline(certPEM)
	if err := os.WriteFile(filepath.Join(store, "fullchain"), chain, 0644); err != nil {
		errors = append(errors, err)
	}

	var combined []byte
	combined = append(combined, chain...)
	combined = append(combined, privPEM...)
	if err := os.WriteFile(filepath.Join(store, "combined"), combined, 0644); err != nil {
		errors = append(errors, err)
	}

	return store, errors.Nil()
}

func updateLinks(base, store string, domains []string) error {
	var errors MultiError
	live := filepath.Join(base, "live")
	os.MkdirAll(live, 0755)
	for _, domain := range domains {
		if err := replaceLink(live, store, domain); err != nil {
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
