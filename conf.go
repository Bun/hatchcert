package hatchcert

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mholt/acmez/v3"
	"github.com/mholt/acmez/v3/acme"
)

type Configuration struct {
	ACME           string
	AcceptedTOS    bool
	Email          string
	PreferredChain string
	Profile        string
	Certs          []Cert
	UpdateHooks    []string

	WebServer string
	dnsLookup map[string]dnsDomainConfig
	Solvers   map[string]acmez.Solver
}

func Conf(fname string) (c Configuration, err error) {
	buf, err := os.ReadFile(fname)
	if err != nil {
		return c, err
	}
	lines := strings.Split(string(buf), "\n")
	c.Solvers = make(map[string]acmez.Solver)

	c.dnsLookup = make(map[string]dnsDomainConfig)
	var dnsConfig dnsDomainConfig

	dnsSolvers := newMultisolver()
	var activeSolver func() acmez.Solver

	for lnum, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cmd, args, _ := strings.Cut(line, " ")
		args = strings.Trim(args, " \t")
		switch strings.ReplaceAll(cmd, "_", "-") {
		case "acme-url":
			c.ACME = args
		case "accept-tos":
			c.AcceptedTOS = true
		case "email":
			c.Email = args
		case "domain":
			if args == "" {
				return c, fmt.Errorf("line %v: domain keyword requires one or more domains", lnum+1)
			}
			parts := strings.Split(args, " ")
			for i, part := range parts {
				// We don't expect FQDN
				parts[i] = strings.TrimRight(part, ".")
			}
			c.Certs = append(c.Certs, Cert{Name: parts[0], Domains: parts})
			if activeSolver != nil {
				s := activeSolver()
				for _, n := range parts {
					dnsSolvers.lookup[strings.TrimPrefix(n, "*.")] = s
				}
			}
		case "preferred-chain":
			c.PreferredChain = args
		case "profile":
			c.Profile = args
		case "update-hook":
			c.UpdateHooks = append(c.UpdateHooks, args)

		// Challenge solvers
		case "httpdir":
			os.MkdirAll(args, 0755)
			c.Solvers[acme.ChallengeTypeHTTP01] = &webrootSolver{
				root: args,
			}
		case "webroot":
			// Legacy
			path := filepath.Join(args, ".well-known/acme-challenge")
			os.MkdirAll(path, 0755)
			c.Solvers[acme.ChallengeTypeHTTP01] = &webrootSolver{
				root: path,
			}

		case "http":
			listen := ":80"
			if args != "" {
				listen = args
			}
			c.WebServer = listen
			c.Solvers[acme.ChallengeTypeHTTP01] = &httpSolver{
				listen: listen,
			}

		case "env":
			slog.Warn("Deprecated option `env` has no effect",
				"line", lnum+1)

		case "dns":
			parts := strings.Split(args, " ")
			if _arg(parts, 0) == "persist" {
				activeSolver = nil
				c.Solvers["dns-persist-01"] = dnsPersist01{}
			} else if _arg(parts, 0) == "rfc2136" {
				activeSolver = func() acmez.Solver {
					return dnsUpdate2136{
						dnsPropagationHelper: dnsPropagationHelper{
							PropagationTimeout: time.Minute,
						},
						cfg: dnsConfig,
					}
				}

				c.Solvers[acme.ChallengeTypeDNS01] = dnsSolvers
				dnsConfig.Nameserver = nameserverAddr(_arg(parts, 1))
				dnsConfig.Zone = _arg(parts, 2)
			} else if _arg(parts, 0) == "exec" {
				if len(parts) == 1 {
					return c, fmt.Errorf("line %v: `exec` requires a script name",
						lnum+1)
				}
				cmd := strings.TrimPrefix(args, "exec ")
				activeSolver = func() acmez.Solver {
					return execSolver{
						command: cmd,
					}
				}
				c.Solvers[acme.ChallengeTypeDNS01] = dnsSolvers
			} else {
				return c, fmt.Errorf("line %v: unsupported DNS provider %q",
					lnum+1, _arg(parts, 0))
			}

		case "tsig":
			parts := strings.Split(args, " ")
			if l := len(parts); l != 2 && l != 3 {
				return c, fmt.Errorf("line %v: expected: tsig <key> <secret> (algo)",
					lnum+1)
			}
			dnsConfig.KeyName = trailingDot(_arg(parts, 0))
			dnsConfig.Secret = _arg(parts, 1)
			if alg := _arg(parts, 2); alg != "" {
				dnsConfig.Algo = trailingDot(alg)
			}

		default:
			return c, fmt.Errorf("line %v: unsupported keyword %q",
				lnum+1, cmd)
		}
	}

	return
}

func _arg(args []string, i int) string {
	if i >= len(args) {
		return ""
	}
	return args[i]
}
