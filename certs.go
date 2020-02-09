package hatchcert

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-acme/lego/v3/certificate"
)

type Cert struct {
	Name       string
	Domains    []string
	AuthMethod string
}

func issue(a *AccountMeta, cert Cert) error {
	request := certificate.ObtainRequest{
		Domains:    cert.Domains,
		Bundle:     false,
		PrivateKey: nil,
		MustStaple: false,
	}

	c, err := a.Client.Certificate.Obtain(request)
	if err != nil {
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

	if cert.Certificate != nil {
		err := ioutil.WriteFile(filepath.Join(store, "cert"), cert.Certificate, 0644)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if cert.Certificate != nil || cert.IssuerCertificate != nil {
		var chain []byte
		chain = append(chain, cert.Certificate...)
		chain = append(chain, cert.IssuerCertificate...)
		err := ioutil.WriteFile(filepath.Join(store, "fullchain"), chain, 0644)
		if err != nil {
			errors = append(errors, err)
		}
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

func Issue(a *AccountMeta, certs []Cert) error {
	var errors MultiError
	for _, cert := range certs {
		if err := issue(a, cert); err != nil {
			errors = append(errors, err)
		}
	}
	return errors.Nil()
}
