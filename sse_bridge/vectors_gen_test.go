package main

// Throwaway generator: prints colbin-encoded UserToken bytes so the Rust port in
// server_utils/src/bridge/token.rs can pin them as test vectors. Delete with the
// rest of this module once the Rust bridge is verified.

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ivanjoz/colbin"
)

func TestPrintUserTokenVectors(t *testing.T) {
	cases := []UserToken{
		{CompanyID: 7, ID: 42, Created: 1234, User: "tester"},
		{CompanyID: 1, ID: 1, Created: 0, User: ""},
		{CompanyID: 2147483647, ID: 2147483647, Created: 2147483647, User: "x"},
		{CompanyID: 999999, ID: 12345, Created: 1700000000, User: "ñandú@example.com"},
		{CompanyID: 128, ID: 127, Created: 65536, User: "a-very-long-user-name-for-width-testing"},
	}

	for _, userToken := range cases {
		userToken.Hash = computeUserTokenHash(userToken, testApiKey)
		encoded, marshalError := colbin.Marshal(userToken)
		if marshalError != nil {
			t.Fatalf("marshal: %v", marshalError)
		}
		fmt.Printf("VECTOR company=%d user=%d created=%d hash=%d user_name=%q hex=%s\n",
			userToken.CompanyID, userToken.ID, userToken.Created, userToken.Hash,
			userToken.User, hex.EncodeToString(encoded))
	}

	// Field ids the Rust side must derive from the same FNV-1a + probe algorithm.
	for _, fieldName := range []string{"CompanyID", "ID", "Created", "Hash", "User"} {
		fmt.Printf("FIELDID %s = %d\n", fieldName, fnv8ForVectors(fieldName))
	}
}

// fnv8ForVectors mirrors colbin's unexported fnv8 so the expected ids are visible here.
func fnv8ForVectors(name string) uint8 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	hash := uint32(offset32)
	for i := 0; i < len(name); i++ {
		hash ^= uint32(name[i])
		hash *= prime32
	}
	return uint8(hash ^ (hash >> 8) ^ (hash >> 16) ^ (hash >> 24))
}
