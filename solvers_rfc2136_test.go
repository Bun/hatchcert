package hatchcert

import "testing"

func TestAutoZone(t *testing.T) {
	testDnsAutoZone := func(expect, fqdn string) {
		if r, err := findZone("127.0.0.1:5353", fqdn); err != nil && expect != "" {
			t.Errorf("findZone(%q) err %v", fqdn, err)
		} else if r != expect {
			t.Errorf("findZone(%q) != %q (%q)", fqdn, expect, r)
		}
	}
	testDnsAutoZone("", "invalid-domain.")
	//testDnsAutoZone("example.com", "www.example.com.")
	testDnsAutoZone("localhost.test.", "localhost.test.")
	testDnsAutoZone("localhost.test.", "www.localhost.test.")
}
