package hatchcert

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/providers/http/webroot"
)

type Configuration struct {
	ACME        string
	AcceptedTOS bool
	Email       string
	Certs       []Cert

	Challenge struct {
		HTTP []challenge.Provider
		DNS  []challenge.Provider
	}
}

func Conf(fname string) (c Configuration) {
	buf, _ := ioutil.ReadFile(fname)
	lines := strings.Split(string(buf), "\n")

	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, " ")
		switch parts[0] {
		case "acme_url":
			c.ACME = parts[1]
		case "accept_tos":
			c.AcceptedTOS = true
		case "email":
			c.Email = parts[1]
		case "domain":
			c.Certs = append(c.Certs, Cert{Name: parts[1], Domains: parts[1:]})

		case "webroot":
			// TODO: this provider is retarded and writes to
			// "./.well-known/acme-challenge/XXX" in the given path
			// TODO: make relative to main dir
			// TODO: the directory must exist
			os.MkdirAll(parts[1], 0755)
			p, err := webroot.NewHTTPProvider(parts[1])
			if err != nil {
				panic(err)
			}
			c.Challenge.HTTP = append(c.Challenge.HTTP, p)

		default:
			panic(line)
		}
	}

	return
}
