package core

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"strings"

	"github.com/ivanjoz/colbin"
	"github.com/ivanjoz/genix-orm/utils"
	"golang.org/x/crypto/blake2s"
)

// WebCustomerToken is the session of a web store customer (sales.WebCustomer). It is signed with
// the same key as the back-office UsuarioToken but under its own domain, so the two kinds can never
// stand in for each other: a customer token fails CheckUser's hash on every back-office route, and
// a back-office token fails CheckWebCustomer here.
//
// It travels in the X-Customer-Token header, never in Authorization, and only to "p-" routes:
// those skip the back-office gate entirely, and each customer route checks this token itself.
type WebCustomerToken struct {
	CompanyID  int32  `cb:"1"`
	CustomerID int32  `cb:"2"`
	Created    int32  `cb:"3"`
	Hash       []byte `cb:"4"`
}

// webCustomerTokenDomain separates this tag from the back-office one (usuarioTokenDomain).
const webCustomerTokenDomain = "webCustomerToken:v1"

// webCustomerTokenMaxAgeSUnits is 90 days, in SUnixTime units (2 seconds each).
const webCustomerTokenMaxAgeSUnits = int32(90 * 24 * 3600 / 2)

func computeWebCustomerTokenHash(token WebCustomerToken) []byte {
	tokenKey := sha256.Sum256([]byte(Env.SECRET_PHRASE))
	hasher, err := blake2s.New128(tokenKey[:])
	if err != nil {
		Log("computeWebCustomerTokenHash:: blake2s rechazó la clave derivada", err)
		return nil
	}
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(token.CompanyID))
	binary.BigEndian.PutUint32(payload[4:8], uint32(token.CustomerID))
	binary.BigEndian.PutUint32(payload[8:12], uint32(token.Created))
	hasher.Write([]byte(webCustomerTokenDomain))
	hasher.Write(payload)
	return hasher.Sum(nil)
}

// MakeWebCustomerToken signs a session for one customer of one company.
func MakeWebCustomerToken(companyID, customerID int32) (string, error) {
	token := WebCustomerToken{CompanyID: companyID, CustomerID: customerID, Created: SUnixTime()}
	token.Hash = computeWebCustomerTokenHash(token)
	tokenBytes, err := colbin.Marshal(token)
	if err != nil {
		return "", Err("No se pudo generar la sesión del cliente:", err)
	}
	return utils.Base64.EncodeToString(tokenBytes), nil
}

// CheckWebCustomer reads and verifies the X-Customer-Token header. The token must belong to the
// company the request is for, so a session of one store cannot act on another.
func CheckWebCustomer(req *HandlerArgs, companyID int32) (*WebCustomerToken, error) {
	tokenBase64 := strings.TrimSpace(MapGetKeys(req.Headers, "X-Customer-Token", "x-customer-token"))
	if len(tokenBase64) < 8 {
		return nil, Err("Debe iniciar sesión como cliente.")
	}
	tokenBytes, _ := utils.Base64.DecodeString(tokenBase64)
	token := WebCustomerToken{}
	if len(tokenBytes) < 8 || colbin.Unmarshal(tokenBytes, &token) != nil {
		return nil, Err("La sesión del cliente es inválida.")
	}
	// Constant-time for the same reason as CheckUser: the tag is the whole credential.
	if subtle.ConstantTimeCompare(token.Hash, computeWebCustomerTokenHash(token)) != 1 {
		return nil, Err("La sesión del cliente no es válida.")
	}
	if token.CompanyID != companyID || token.CustomerID <= 0 {
		return nil, Err("La sesión del cliente pertenece a otra tienda.")
	}
	if SUnixTime()-token.Created > webCustomerTokenMaxAgeSUnits {
		return nil, Err("La sesión del cliente expiró. Vuelva a iniciar sesión.")
	}
	return &token, nil
}
