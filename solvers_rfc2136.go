package hatchcert

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mholt/acmez/v3/acme"
	"github.com/miekg/dns"
)

type dnsDomainConfig struct {
	// Nameserver is the authoritative nameserver to send DNS UPDATE messages
	// to in "host:port" form.
	Nameserver string

	Zone    string
	KeyName string
	Secret  string

	// Algo defaults to
	// hmac-sha256 if empty.
	Algo string
}

// dnsUpdate2136 implements a DNS-01 challenge solver using RFC 2136 dynamic DNS
// updates.
type dnsUpdate2136 struct {
	dnsPropagationHelper
	cfg dnsDomainConfig
}

// Present creates the DNS TXT record required to satisfy the DNS-01 challenge.
func (s dnsUpdate2136) Present(ctx context.Context, challenge acme.Challenge) error {
	fqdn := trailingDot(challenge.DNS01TXTRecordName())
	value := challenge.DNS01KeyAuthorization()

	if err := s.upsertTXT(ctx, s.cfg, fqdn, value, true); err != nil {
		return fmt.Errorf("rfc2136: present %q: %w", challenge.Identifier.Value, err)
	}

	return nil
}

// Wait until the DNS record has propagated.
func (s dnsUpdate2136) Wait(ctx context.Context, challenge acme.Challenge) error {
	if s.PropagationTimeout > 0 {
		ns := s.cfg.Nameserver
		fqdn := trailingDot(challenge.DNS01TXTRecordName())
		value := challenge.DNS01KeyAuthorization()
		if err := s.waitForPropagation(ctx, ns, fqdn, value); err != nil {
			return fmt.Errorf("rfc2136: propagation check %s: %w", fqdn, err)
		}
	}

	return nil
}

// CleanUp removes the DNS TXT record after the challenge has been completed.
func (s dnsUpdate2136) CleanUp(ctx context.Context, challenge acme.Challenge) error {
	fqdn := trailingDot(challenge.DNS01TXTRecordName())
	value := challenge.DNS01KeyAuthorization()

	if err := s.upsertTXT(ctx, s.cfg, fqdn, value, false); err != nil {
		return fmt.Errorf("rfc2136: cleanup %s: %w", fqdn, err)
	}
	return nil
}

// upsertTXT adds (insert=true) or removes (insert=false) a TXT record via
// RFC 2136 DNS UPDATE.
func (s dnsUpdate2136) upsertTXT(ctx context.Context, cfg dnsDomainConfig, fqdn, value string, insert bool) error {
	// TODO: probably not as part of parsing configuration
	zone := cfg.Zone
	if zone == "" {
		var err error
		if zone, err = findZone(cfg.Nameserver, fqdn); err != nil {
			return err
		}
	}

	slog.Info("upsertTXT",
		"fqdn", fqdn,
		"zone", cfg.Zone)

	rr := &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   fqdn,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    30,
		},
		Txt: []string{value},
	}

	msg := &dns.Msg{}
	msg.SetUpdate(zone)

	if insert {
		msg.Insert([]dns.RR{rr})
	} else {
		msg.Remove([]dns.RR{rr})
	}

	return s.sendUpdate(ctx, cfg, msg)
}

// sendUpdate transmits a DNS UPDATE message to the configured nameserver,
// optionally authenticating it with TSIG.
func (s dnsUpdate2136) sendUpdate(ctx context.Context, cfg dnsDomainConfig, msg *dns.Msg) error {
	client := &dns.Client{Net: "tcp"}

	if cfg.KeyName != "" {
		alg := cfg.Algo
		if alg == "" {
			alg = dns.HmacSHA256
		}
		msg.SetTsig(cfg.KeyName, alg, 300, time.Now().Unix())
		client.TsigSecret = map[string]string{cfg.KeyName: cfg.Secret}
	}

	reply, _, err := client.ExchangeContext(ctx, msg, cfg.Nameserver)
	if err != nil {
		return fmt.Errorf("DNS update exchange: %w", err)
	}
	if reply.Rcode != dns.RcodeSuccess {
		return fmt.Errorf("DNS update failed with rcode %s (%d)",
			dns.RcodeToString[reply.Rcode], reply.Rcode)
	}
	return nil
}
