package server_utils

import (
	"testing"
)

func TestAPIGroupsUseDocumentedBoundaries(t *testing.T) {
	tests := []struct {
		method string
		bytes  int
		want   uint8
	}{
		{"GET", 32*1024 - 1, 0}, {"GET", 32 * 1024, 1}, {"GET", 256 * 1024, 1},
		{"GET", 256*1024 + 1, 2}, {"POST", 0, 3}, {"POST", 32 * 1024, 4},
		{"POST", 256*1024 + 1, 5},
	}
	for _, test := range tests {
		got, err := APIGroup(test.method, test.bytes)
		if err != nil || got != test.want {
			t.Fatalf("APIGroup(%q, %d) = %d, %v; want %d", test.method, test.bytes, got, err, test.want)
		}
	}
}

func TestCreditFormulasRoundPartialBlocksUp(t *testing.T) {
	checks := []struct {
		method string
		bytes  int
		want   uint16
	}{
		{"GET", 0, 2}, {"GET", 8 * 1024, 2}, {"GET", 8*1024 + 1, 3},
		{"GET", 24 * 1024, 3}, {"GET", 24*1024 + 1, 4},
		{"POST", 0, 5}, {"POST", 8 * 1024, 5}, {"POST", 8*1024 + 1, 6},
		{"POST", 16 * 1024, 6}, {"POST", 16*1024 + 1, 7},
	}
	for _, check := range checks {
		got, err := APICPUCredits(check.method, check.bytes)
		if err != nil || got != check.want {
			t.Fatalf("APICPUCredits(%q, %d) = %d, %v; want %d", check.method, check.bytes, got, err, check.want)
		}
	}
	inference, err := InferenceCredits(8*1024+1, 8*1024+1)
	if err != nil || inference != 6 {
		t.Fatalf("InferenceCredits() = %d, %v; want 6", inference, err)
	}
}
