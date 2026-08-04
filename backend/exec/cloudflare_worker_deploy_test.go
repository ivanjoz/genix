package exec

import "testing"

func TestParseCompanyIDArgument(t *testing.T) {
	companyID, parseError := parseCompanyIDArgument("123")
	if parseError != nil || companyID != 123 {
		t.Fatalf("expected CompanyID 123, got %d, error=%v", companyID, parseError)
	}

	invalidArguments := []string{"", "0", "-1", "abc", "1 2"}
	for _, invalidArgument := range invalidArguments {
		if _, parseError := parseCompanyIDArgument(invalidArgument); parseError == nil {
			t.Fatalf("expected %q to fail", invalidArgument)
		}
	}
}

func TestNormalizeAndValidateHostname(t *testing.T) {
	hostname, normalizeError := normalizeAndValidateHostname(" HTTPS://Store.UN.PE/path ")
	if normalizeError != nil || hostname != "store.un.pe" {
		t.Fatalf("unexpected hostname=%q error=%v", hostname, normalizeError)
	}

	invalidHostnames := []string{"localhost", "-store.un.pe", "store_.un.pe", "store..un.pe"}
	for _, invalidHostname := range invalidHostnames {
		if _, normalizeError := normalizeAndValidateHostname(invalidHostname); normalizeError == nil {
			t.Fatalf("expected %q to fail", invalidHostname)
		}
	}
}
