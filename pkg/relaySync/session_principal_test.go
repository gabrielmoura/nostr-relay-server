package relaysync

import "testing"

func TestSamePrincipalNormalizesMappedIPv4(t *testing.T) {
	if !samePrincipal(Principal{ID: "127.0.0.1"}, Principal{ID: "::ffff:127.0.0.1"}) {
		t.Fatal("IPv4 and mapped IPv6 identities should compare equal")
	}
	if samePrincipal(Principal{ID: "127.0.0.1"}, Principal{ID: "127.0.0.2"}) {
		t.Fatal("different IP identities should not compare equal")
	}
}
