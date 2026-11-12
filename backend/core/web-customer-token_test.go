package core

import (
	"testing"

	"github.com/ivanjoz/colbin"
	"github.com/ivanjoz/genix-orm/utils"
)

func TestWebCustomerTokenRoundTrip(t *testing.T) {
	useTokenTestEnv(t)
	token, err := MakeWebCustomerToken(7, 42)
	if err != nil {
		t.Fatal(err)
	}
	req := &HandlerArgs{Headers: map[string]string{"x-customer-token": token}}
	decoded, err := CheckWebCustomer(req, 7)
	if err != nil || decoded.CustomerID != 42 {
		t.Fatalf("expected customer 42, got %+v (err %v)", decoded, err)
	}
	if _, err := CheckWebCustomer(req, 8); err == nil {
		t.Fatal("a session of store 7 must not open store 8")
	}
}

// The two token kinds share a key, so what keeps them apart is the hash domain: a back-office
// token signed for user 42 must not pass as web customer 42, nor the reverse.
func TestTokenKindsCannotStandInForEachOther(t *testing.T) {
	useTokenTestEnv(t)

	backOfficeToken := UsuarioToken{CompanyID: 7, ID: 42, Created: SUnixTime()}
	backOfficeToken.Hash = ComputeUsuarioTokenHash(backOfficeToken)
	forgedCustomer := WebCustomerToken{CompanyID: 7, CustomerID: 42, Created: backOfficeToken.Created, Hash: backOfficeToken.Hash}
	forgedBytes, _ := colbin.Marshal(forgedCustomer)
	req := &HandlerArgs{Headers: map[string]string{"x-customer-token": utils.Base64.EncodeToString(forgedBytes)}}
	if _, err := CheckWebCustomer(req, 7); err == nil {
		t.Fatal("a back-office hash must not validate a web customer token")
	}

	customerToken := WebCustomerToken{CompanyID: 7, CustomerID: 42, Created: SUnixTime()}
	customerToken.Hash = computeWebCustomerTokenHash(customerToken)
	forgedUser := UsuarioToken{CompanyID: 7, ID: 42, Created: customerToken.Created, Hash: customerToken.Hash}
	if string(ComputeUsuarioTokenHash(forgedUser)) == string(forgedUser.Hash) {
		t.Fatal("a web customer hash must not validate a back-office token")
	}
}

func useTokenTestEnv(t *testing.T) {
	previousEnv := Env
	Env = &EnvStruct{SECRET_PHRASE: "test-phrase"}
	t.Cleanup(func() { Env = previousEnv })
}
