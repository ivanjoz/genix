package main

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math"
)

// Channel token: the single identifier of one browser tab's stream, carrying
// the three values that make it unique — company, user and tab.
//
//	bytes = uvarint(companyID) ‖ uvarint(userID) ‖ 6 random bytes (tab)
//	token = base64url(bytes), unpadded
//
// Typical ids fit in one byte each, so the whole thing is 11 characters. The
// varints are what keep it short: a decimal company id costs 6-7 characters on
// its own.
//
// This is an *identifier*, never a credential. The browser still proves who it
// is with its session token, and both this process and the backend check that
// the identity inside the token matches the authenticated one — otherwise
// editing a token would be a way to address another tenant's tab.
//
// Mirrored in backend/agent/channel.go and frontend/core/agent/channel.ts. The
// three copies must agree byte for byte; there is no shared module to enforce
// it (the bridge and the backend are separate Go modules).

// tabRandomBytes is the tab's entropy: 6 bytes = 48 bits, and exactly 8
// characters once base64url-encoded, which is the tab id's budget.
const tabRandomBytes = 6

// DecodeChannelToken parses a channel token into its three parts.
//
// It rejects non-canonical encodings (an overlong varint decodes to the same
// number but yields a different token string). That rejection is what makes the
// token a bijection with the triple, which in turn is what lets the token be
// used directly as the registry key: two distinct strings can never name the
// same channel.
func DecodeChannelToken(channelToken string) (companyID int32, userID int32, tabID string, decodeError error) {
	tokenBytes, base64Error := base64.RawURLEncoding.DecodeString(channelToken)
	if base64Error != nil {
		return 0, 0, "", errors.New("el token de canal no es base64url válido")
	}

	companyValue, companyByteCount := binary.Uvarint(tokenBytes)
	if companyByteCount <= 0 {
		return 0, 0, "", errors.New("el token de canal no contiene un company id")
	}
	userValue, userByteCount := binary.Uvarint(tokenBytes[companyByteCount:])
	if userByteCount <= 0 {
		return 0, 0, "", errors.New("el token de canal no contiene un user id")
	}

	tabBytes := tokenBytes[companyByteCount+userByteCount:]
	if len(tabBytes) != tabRandomBytes {
		return 0, 0, "", errors.New("el token de canal no contiene un tab id de 6 bytes")
	}
	if companyValue == 0 || companyValue > math.MaxInt32 || userValue == 0 || userValue > math.MaxInt32 {
		return 0, 0, "", errors.New("el token de canal contiene identificadores fuera de rango")
	}

	companyID, userID = int32(companyValue), int32(userValue)
	tabID = base64.RawURLEncoding.EncodeToString(tabBytes)

	// Canonicality check by round-trip: cheaper to write than validating each
	// varint by hand, and it can't miss a case.
	if EncodeChannelToken(companyID, userID, tabID) != channelToken {
		return 0, 0, "", errors.New("el token de canal no está codificado de forma canónica")
	}
	return companyID, userID, tabID, nil
}

// EncodeChannelToken builds the token for one tab. tabID is the 8-character
// base64url form of the tab's 6 random bytes; an invalid one yields "".
func EncodeChannelToken(companyID, userID int32, tabID string) string {
	tabBytes, base64Error := base64.RawURLEncoding.DecodeString(tabID)
	if base64Error != nil || len(tabBytes) != tabRandomBytes || companyID <= 0 || userID <= 0 {
		return ""
	}

	tokenBytes := binary.AppendUvarint(nil, uint64(companyID))
	tokenBytes = binary.AppendUvarint(tokenBytes, uint64(userID))
	tokenBytes = append(tokenBytes, tabBytes...)
	return base64.RawURLEncoding.EncodeToString(tokenBytes)
}
