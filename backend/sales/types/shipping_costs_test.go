package types

import "testing"

func TestResolveShippingAmountFallsBackUpTheUbigeoTree(t *testing.T) {
	// Miraflores (150122) sits under Lima province (1501) under Lima department (15).
	cases := []struct {
		name          string
		rates         map[int32]int32
		wantAmount    int32
		wantDelivered bool
	}{
		{"district rate wins", map[int32]int32{150122: 600, 1501: 900, 15: 1500}, 600, true},
		{"province when the district has none", map[int32]int32{1501: 900, 15: 1500}, 900, true},
		{"department last", map[int32]int32{15: 1500}, 1500, true},
		{"zero counts as not set", map[int32]int32{150122: 0, 1501: 900}, 900, true},
		{"no level has a rate", map[int32]int32{150101: 500}, 0, false},
	}
	for _, testCase := range cases {
		amount, found := ResolveShippingAmount(150122, testCase.rates)
		if amount != testCase.wantAmount || found != testCase.wantDelivered {
			t.Errorf("%s: got (%v, %v), want (%v, %v)", testCase.name, amount, found,
				testCase.wantAmount, testCase.wantDelivered)
		}
	}
}

func TestPublicWebCustomerHidesThePasswordHash(t *testing.T) {
	customer := WebCustomer{Email: "a@b.pe", PasswordHash: "secret", CreatedIP: 99}
	public := customer.PublicWebCustomer()
	if public.PasswordHash != "" || public.CreatedIP != 0 {
		t.Fatalf("public customer leaks private fields: %+v", public)
	}
	if !customer.HasPassword() {
		t.Fatal("the original keeps its hash")
	}
}
