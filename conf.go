package hatchcert

import (
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/providers/http/webroot"
)

type Configuration struct {
	ACME        string
	AcceptedTOS bool
	Email       string
	Certs       []Cert
	UpdateHooks [][]string

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
		case "update-hook":
			// TODO: could do some more parsing
			c.UpdateHooks = append(c.UpdateHooks, parts[1:])

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

		case "http":
			listen := ":80"
			if len(parts) > 1 {
				listen = parts[1]
			}
			host, port, err := net.SplitHostPort(listen)
			if err != nil {
				panic(err)
			}
			p := http01.NewProviderServer(host, port)
			c.Challenge.HTTP = append(c.Challenge.HTTP, p)

		case "env":
			kv := strings.SplitN(parts[1], "=", 2)
			if err := os.Setenv(kv[0], kv[1]); err != nil {
				panic(err)
			}

		case "dns":
			// TODO: postpone since it relies on env
			provider, err := dns.NewDNSChallengeProviderByName(parts[1])
			if err != nil {
				panic(err)
			}
			c.Challenge.DNS = append(c.Challenge.DNS, provider)

		default:
			panic(line)
		}
	}

	return
}
