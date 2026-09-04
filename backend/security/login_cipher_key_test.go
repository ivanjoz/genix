package security

import (
	"os"
	"testing"

	"app/core"
	coreTypes "app/core/types"
)

// ID 2, not 1: user 1 takes the buildBootstrapAdminAccesos branch, which needs the embedded access
// catalog. What is under test here is the CipherKey branch, so the grants stay out of the way.
func makeTestUser() coreTypes.User {
	return coreTypes.User{CompanyID: 1, ID: 2, User: "tester", FirstName: "Test"}
}

// The plaintext UserInfo branch is a deliberate hole in the login response, so what it is keyed on
// is worth pinning: the "dev" launch argument, and nothing a config file can reach.
func TestReadDevArgumentOnlyMatchesTheLaunchArgument(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	cases := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"start.js", []string{"/tmp/go-build/exe/backend", "dev"}, true},
		{"deployed unit, no arguments", []string{"/usr/local/bin/genix/genix_app"}, false},
		{"exec function", []string{"backend", "fn-recalc"}, false},
		// os.Args[0] is a path the deployer chooses, so a binary that happens to be named "dev"
		// must not grant anything.
		{"binary named dev", []string{"/usr/local/bin/dev"}, false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			os.Args = testCase.args
			if got := core.ReadDevArgument(); got != testCase.expected {
				t.Fatalf("ReadDevArgument() = %v, expected %v for args %v", got, testCase.expected, testCase.args)
			}
		})
	}
}

// An empty CipherKey is the client saying "I have no crypto.subtle". Outside a dev launch that must
// be an error and never a plaintext response — there is deliberately no config key that can grant it.
func TestMakeUsuarioResponseRefusesAnEmptyCipherKeyOutsideDev(t *testing.T) {
	originalEnv := core.Env
	defer func() { core.Env = originalEnv }()
	core.Env = &core.EnvStruct{IS_DEV_ARG: false}

	if _, err := MakeUsuarioResponse(makeTestUser(), ""); err == nil {
		t.Fatal("expected an error for an empty CipherKey without the dev argument, got none")
	}
}

func TestMakeUsuarioResponseSendsPlainUserInfoOnADevLaunch(t *testing.T) {
	originalEnv := core.Env
	defer func() { core.Env = originalEnv }()
	core.Env = &core.EnvStruct{IS_DEV_ARG: true}

	response, err := MakeUsuarioResponse(makeTestUser(), "")
	if err != nil {
		t.Fatalf("MakeUsuarioResponse failed: %v", err)
	}
	if _, hasPlain := response["UserInfoPlain"]; !hasPlain {
		t.Fatal("expected UserInfoPlain in the response")
	}
	// The two are exclusive: a client that cannot decrypt must not also receive a payload it would
	// then have to decide about.
	if _, hasCiphered := response["UserInfo"]; hasCiphered {
		t.Fatal("expected no ciphered UserInfo alongside UserInfoPlain")
	}
	if response["UserToken"] == "" {
		t.Fatal("the session token must still be issued")
	}
}

func TestMakeUsuarioResponseCiphersUserInfoWhenGivenAKey(t *testing.T) {
	originalEnv := core.Env
	defer func() { core.Env = originalEnv }()
	core.Env = &core.EnvStruct{IS_DEV_ARG: true}

	response, err := MakeUsuarioResponse(makeTestUser(), "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("MakeUsuarioResponse failed: %v", err)
	}
	if _, hasPlain := response["UserInfoPlain"]; hasPlain {
		t.Fatal("a client that sent a key must get the ciphered payload, not UserInfoPlain")
	}
	if response["UserInfo"] == "" {
		t.Fatal("expected a ciphered UserInfo")
	}
}
