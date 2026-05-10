package hatchcert

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type dnsPropagationHelper struct {
	// PropagationTimeout is the optional amount of time to wait for the DNS
	// record to propagate.
	PropagationTimeout time.Duration
}

// waitForPropagation polls the authoritative nameserver until the TXT record
// appears, or until PropagationTimeout elapses.
func (s dnsPropagationHelper) waitForPropagation(ctx context.Context, ns, fqdn, value string) error {
	deadline := time.Now().Add(s.PropagationTimeout)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	for {
		found, err := s.verifyTXT(ctx, ns, fqdn, value)
		if err == nil && found {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for TXT %q to propagate: %w", fqdn, ctx.Err())
		case <-time.After(time.Second * 2):
		}
	}
}

// verifyTXT queries the configured nameserver for the TXT record and checks
// whether the expected value is present.
func (s dnsPropagationHelper) verifyTXT(ctx context.Context, ns, fqdn, value string) (bool, error) {
	msg := &dns.Msg{}
	msg.SetQuestion(fqdn, dns.TypeTXT)
	msg.RecursionDesired = false

	client := &dns.Client{}

	reply, _, err := client.ExchangeContext(ctx, msg, ns)
	if err != nil {
		return false, err
	}

	for _, rr := range reply.Answer {
		txt, ok := rr.(*dns.TXT)
		if !ok {
			continue
		}
		for _, s := range txt.Txt {
			if s == value {
				return true, nil
			}
		}
	}
	return false, nil
}

func trailingDot(n string) string {
	if !strings.HasSuffix(n, ".") {
		return n + "."
	}
	return n
}

func nameserverAddr(ns string) string {
	if _, _, err := net.SplitHostPort(ns); err != nil {
		return net.JoinHostPort(ns, "53")
	}
	return ns
}

// findZone walks up the DNS hierarchy to find the authoritative zone for fqdn
// by querying the local resolver for SOA records.
func findZone(resolver, fqdn string) (string, error) {
	labels := dns.SplitDomainName(fqdn)
	if labels == nil {
		return "", fmt.Errorf("invalid fqdn: %q", fqdn)
	}

	c := &dns.Client{}

	// The expected case is that the first query will give us the result we
	// need
	for i := range labels {
		candidate := dns.Fqdn(strings.Join(labels[i:], "."))

		msg := &dns.Msg{}
		msg.SetQuestion(candidate, dns.TypeSOA)

		reply, _, err := c.Exchange(msg, resolver)
		if err != nil || reply.Rcode != dns.RcodeSuccess {
			continue
		}
		for _, rr := range reply.Answer {
			// The FQDN has a SOA record
			if soa, ok := rr.(*dns.SOA); ok {
				return soa.Hdr.Name, nil
			}
		}
		for _, rr := range reply.Ns {
			// The FQDN doesn't have a SOA record, but the authority section
			// points us to it
			if soa, ok := rr.(*dns.SOA); ok {
				return soa.Hdr.Name, nil
			}
		}
	}
	return "", fmt.Errorf("could not determine authoritative zone for %q", fqdn)
}
