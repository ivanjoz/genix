package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ivanjoz/colbin"
)

// The bridge authenticates two very different callers with the same shared
// secret (SSE_BRIDGE_APIKEY in this process, SECRET_PHRASE in the backend that
// issues the tokens — one value, two deployment names):
//
//   - the browser, with the session token the backend already issues. The token
//     is self-contained (colbin payload + HMAC), so identity is verified without
//     any database access — the bridge never talks to ScyllaDB.
//   - the backend (Lambda or VPS), with a timestamped HMAC header. No new secret
//     to distribute: both processes already read the same credentials file.
//
// The bridge only establishes *identity*. Permissions ("accesos") stay in the
// backend, which already evaluated them when it accepted the turn.

const (
	serviceAuthHeaderName = "X-Bridge-Auth"
	// serviceAuthMessagePrefix is domain separation: it keeps this signature from
	// ever being valid for another HMAC the project computes with the same key.
	serviceAuthMessagePrefix = "sse-bridge:v1|"
	// serviceAuthMaxSkewSeconds tolerates clock drift between the Lambda and the
	// bridge host while keeping a captured header from being replayable forever.
	serviceAuthMaxSkewSeconds = 300
)

// UserToken mirrors core.UsuarioToken in the backend byte for byte. colbin
// derives each field's wire id from a hash of its Go name, so the field names,
// their order and the `cb:"-"` skip must stay identical to
// backend/core/usuario-accesos.go:18 or decoding silently yields zero values.
type UserToken struct {
	CompanyID int32  `json:"c"`
	ID        int32  `json:"i"`
	Created   int32  `json:"e"`
	Hash      uint64 `json:"h"`
	User      string `json:"u"`
	Error     string `json:"-" cb:"-"` // transient; never serialized into the token
}

// computeUserTokenHash is the exact mirror of core.ComputeUsuarioTokenHash. Any
// change on the backend side must be replicated here — a mismatch rejects every
// client.
func computeUserTokenHash(userToken UserToken, secretPhrase string) uint64 {
	hashMac := hmac.New(sha256.New, []byte(secretPhrase))
	tokenPayloadBuffer := make([]byte, 12)
	binary.BigEndian.PutUint32(tokenPayloadBuffer[0:4], uint32(userToken.CompanyID))
	binary.BigEndian.PutUint32(tokenPayloadBuffer[4:8], uint32(userToken.ID))
	binary.BigEndian.PutUint32(tokenPayloadBuffer[8:12], uint32(userToken.Created))
	hashMac.Write([]byte("usrToken:v1"))
	hashMac.Write(tokenPayloadBuffer)
	hashMac.Write([]byte(userToken.User))
	return binary.BigEndian.Uint64(hashMac.Sum(nil)[:8])
}

// decodeBase64URLAlphabet undoes the backend's MakeB64UrlEncode substitution
// (core/helpers.go:1421) before standard base64 decoding.
func decodeBase64URLAlphabet(encodedContent string) string {
	encodedContent = strings.ReplaceAll(encodedContent, "_", "/")
	encodedContent = strings.ReplaceAll(encodedContent, "-", "+")
	encodedContent = strings.ReplaceAll(encodedContent, "~", "=")
	return encodedContent
}

// authenticateUserRequest extracts and verifies the session token carried by an
// `Authorization: Bearer <token>` header, returning the identity it proves.
func authenticateUserRequest(request *http.Request, secretPhrase string) (UserToken, error) {
	authorizationHeader := strings.TrimSpace(request.Header.Get("Authorization"))
	if len(authorizationHeader) < 8 {
		return UserToken{}, errors.New("falta el token de sesión")
	}

	encodedToken := strings.TrimSpace(strings.TrimPrefix(authorizationHeader, "Bearer "))
	tokenBytes, decodeError := base64.StdEncoding.DecodeString(decodeBase64URLAlphabet(encodedToken))
	if decodeError != nil || len(tokenBytes) < 8 {
		return UserToken{}, errors.New("el token de sesión no es un base64 válido")
	}

	userToken := UserToken{}
	if unmarshalError := colbin.Unmarshal(tokenBytes, &userToken); unmarshalError != nil {
		return UserToken{}, errors.New("no se pudo decodificar el token de sesión: " + unmarshalError.Error())
	}
	if userToken.CompanyID <= 0 || userToken.ID <= 0 {
		return UserToken{}, errors.New("el token de sesión no identifica a un usuario")
	}
	if userToken.Hash != computeUserTokenHash(userToken, secretPhrase) {
		return UserToken{}, errors.New("la firma del token de sesión no es válida")
	}
	return userToken, nil
}

// MakeServiceAuthHeader builds the value the backend sends on `X-Bridge-Auth`.
// Exported (and duplicated in backend/agent/bridge.go, which lives in another Go
// module and cannot import this one) so the two ends stay readable side by side.
func MakeServiceAuthHeader(secretPhrase string, unixSeconds int64) string {
	timestampText := strconv.FormatInt(unixSeconds, 10)
	hashMac := hmac.New(sha256.New, []byte(secretPhrase))
	hashMac.Write([]byte(serviceAuthMessagePrefix + timestampText))
	return timestampText + "." + hex.EncodeToString(hashMac.Sum(nil))
}

// verifyServiceAuthRequest validates the backend's signature and its freshness.
func verifyServiceAuthRequest(request *http.Request, secretPhrase string) error {
	headerValue := strings.TrimSpace(request.Header.Get(serviceAuthHeaderName))
	if len(headerValue) == 0 {
		return errors.New("falta la cabecera " + serviceAuthHeaderName)
	}

	separatorIndex := strings.IndexByte(headerValue, '.')
	if separatorIndex <= 0 {
		return errors.New("cabecera de autenticación de servicio malformada")
	}

	signedUnixSeconds, parseError := strconv.ParseInt(headerValue[:separatorIndex], 10, 64)
	if parseError != nil {
		return errors.New("timestamp de autenticación de servicio inválido")
	}

	elapsedSeconds := time.Now().Unix() - signedUnixSeconds
	if elapsedSeconds < 0 {
		elapsedSeconds = -elapsedSeconds
	}
	if elapsedSeconds > serviceAuthMaxSkewSeconds {
		return errors.New("la autenticación de servicio expiró (" + strconv.FormatInt(elapsedSeconds, 10) + "s de desfase)")
	}

	// Constant-time comparison: a byte-by-byte early exit would leak the expected
	// signature to a caller able to time many attempts.
	expectedHeaderValue := MakeServiceAuthHeader(secretPhrase, signedUnixSeconds)
	if !hmac.Equal([]byte(headerValue), []byte(expectedHeaderValue)) {
		return errors.New("la firma de autenticación de servicio no es válida")
	}
	return nil
}
