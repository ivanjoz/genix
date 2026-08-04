package webpage

import (
	"app/core"
	"testing"
)

// TestNormalizeStorefrontDomain documents the accepted direct-subdomain contract.
func TestNormalizeStorefrontDomain(t *testing.T) {
	previousEnv := core.Env
	core.Env = &core.EnvStruct{ZONE_NAME: "un.pe"}
	defer func() {
		core.Env = previousEnv
	}()

	testCases := []struct {
		name       string
		input      string
		expected   string
		shouldFail bool
	}{
		{name: "normalizes URL", input: "HTTPS://Tienda-X.UN.PE/path", expected: "tienda-x.un.pe"},
		{name: "accepts numeric label", input: "123.un.pe", expected: "123.un.pe"},
		{name: "rejects zone apex", input: "un.pe", shouldFail: true},
		{name: "rejects nested subdomain", input: "www.tienda.un.pe", shouldFail: true},
		{name: "rejects invalid label", input: "-tienda.un.pe", shouldFail: true},
		{name: "rejects another zone", input: "tienda.example.com", shouldFail: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			domain, domainError := normalizeStorefrontDomain(testCase.input)
			if testCase.shouldFail {
				if domainError == nil {
					t.Fatalf("expected %q to fail, got %q", testCase.input, domain)
				}
				return
			}
			if domainError != nil {
				t.Fatalf("expected %q to be valid: %v", testCase.input, domainError)
			}
			if domain != testCase.expected {
				t.Fatalf("expected %q, got %q", testCase.expected, domain)
			}
		})
	}
}

func TestCanBypassDomainCooldown(t *testing.T) {
	cases := []struct {
		name     string
		user     *core.UsuarioToken
		expected bool
	}{
		{"cuenta de desarrollo", &core.UsuarioToken{CompanyID: 1, ID: 1}, true},
		{"otro usuario de la misma company", &core.UsuarioToken{CompanyID: 1, ID: 2}, false},
		{"mismo id en otra company", &core.UsuarioToken{CompanyID: 2, ID: 1}, false},
		{"otra company y otro usuario", &core.UsuarioToken{CompanyID: 7, ID: 3}, false},
		// El token es lo único que decide, así que uno ausente no puede colarse por aquí.
		{"sin token", nil, false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := canBypassDomainCooldown(testCase.user); got != testCase.expected {
				t.Errorf("canBypassDomainCooldown() = %v, se esperaba %v", got, testCase.expected)
			}
		})
	}
}
