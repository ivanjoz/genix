package core

import (
	"testing"
)

func TestCreditLimitResponseUsesHTTP429AndRawCodeHeader(t *testing.T) {
	request := HandlerArgs{Route: "products"}
	response := request.MakeCreditRateLimitResponse(&CreditLimitExceeded{Code: 0b1_1011})
	if response.StatusCode != 429 || response.Headers["X-Rate-Limit-Code"] != "27" {
		t.Fatalf("unexpected handler response: %+v", response)
	}
	lambdaResponse := MakeErrRespFinal(int32(response.StatusCode), response.Error)
	if lambdaResponse.StatusCode != 429 {
		t.Fatalf("Lambda status = %d; want 429", lambdaResponse.StatusCode)
	}
}

func TestClientIPKeyIsStableAndPrefixedForIPv6(t *testing.T) {
	checks := []struct {
		ip   string
		want int64
		ok   bool
	}{
		{"1.2.3.4", 0x01020304, true},
		{"255.255.255.255", 0xFFFFFFFF, true},
		// IPv4-mapped IPv6 must key the same as the plain IPv4 address, or the same client would
		// get two budgets depending on how the proxy spelled it.
		{"::ffff:1.2.3.4", 0x01020304, true},
		{"", 0, false},
		{"not-an-ip", 0, false},
	}
	for _, check := range checks {
		request := HandlerArgs{ClientIP: check.ip}
		got, ok := request.ClientIPKey()
		if ok != check.ok || (ok && got != check.want) {
			t.Fatalf("ClientIPKey(%q) = %d, %v; want %d, %v", check.ip, got, ok, check.want, check.ok)
		}
	}

	// Two addresses inside one customer's /64 must share a key: limiting per address would be
	// free to bypass for anyone holding a normal IPv6 allocation.
	first := HandlerArgs{ClientIP: "2001:db8:abcd:1234::1"}
	second := HandlerArgs{ClientIP: "2001:db8:abcd:1234:ffff:ffff:ffff:ffff"}
	firstKey, _ := first.ClientIPKey()
	secondKey, _ := second.ClientIPKey()
	if firstKey != secondKey {
		t.Fatalf("addresses in one /64 gave %d and %d; want one shared key", firstKey, secondKey)
	}
	if firstKey <= 0xFFFFFFFF {
		t.Fatalf("IPv6 key %d landed in the IPv4 range, where it could collide", firstKey)
	}

	// A different /64 must not share it, or unrelated customers would throttle each other.
	other := HandlerArgs{ClientIP: "2001:db8:abcd:9999::1"}
	otherKey, _ := other.ClientIPKey()
	if otherKey == firstKey {
		t.Fatal("distinct /64 prefixes collapsed onto the same key")
	}
}

func TestFarewardAddressDerivesTheHostFromPublic(t *testing.T) {
	checks := []struct {
		name             string
		host             string
		port             int
		public           bool
		isDevArg         bool
		useRemoteDevHost bool
		want             string
	}{
		// A private daemon only listens on loopback, so a leftover public host must not be dialed:
		// every lock call would go to a machine that cannot answer.
		{"private ignores a stale host", "150.136.42.240", 14013, false, false, false, "127.0.0.1:14013"},
		{"private without a host", "", 14013, false, false, false, "127.0.0.1:14013"},
		{"public dials the host", "150.136.42.240", 14013, true, false, false, "150.136.42.240:14013"},
		// Empty means unconfigured, which ConfigureFareward refuses at startup instead of
		// letting the first lock fail at request time.
		{"public without a host", "  ", 14013, true, false, false, ""},
		{"an omitted port falls back", "10.0.1.65", 0, true, false, false, "10.0.1.65:14013"},
		// A dev machine runs the daemon that matches its checkout, so a public host meant for a
		// deployment is overridden rather than dialed across a wire-protocol boundary.
		{"dev overrides a public host", "150.136.42.240", 14013, true, true, false, "127.0.0.1:14013"},
		{"dev opts into the remote daemon", "150.136.42.240", 14013, true, true, true, "150.136.42.240:14013"},
		// The opt-in only chooses between hosts. It cannot reach a daemon bound to loopback on
		// another machine, so a private daemon stays loopback.
		{"dev opt-in cannot reach a private daemon", "150.136.42.240", 14013, false, true, true, "127.0.0.1:14013"},
	}
	for _, check := range checks {
		got := makeFarewardAddress(
			check.host, check.port, check.public, check.isDevArg, check.useRemoteDevHost)
		if got != check.want {
			t.Fatalf("%s: got %q; want %q", check.name, got, check.want)
		}
	}
}
