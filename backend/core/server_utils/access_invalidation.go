package server_utils

import (
	"context"
	"errors"
	"fmt"
)

// Opcode 0x06: drop a user's cached authorization grants.
//
//	[opcode:1][company:u24][user:u24][hmac:8]
//
// The daemon caches `users.accesos_computed` for ten minutes so it can answer the route gate without
// reading ScyllaDB. This is what keeps that TTL a backstop rather than the mechanism: the backend
// sends it right after rewriting the column, so a revoked access stops working immediately instead
// of at the end of the window.
//
// Unanswered, like the request log, and for the same kind of reason — the TTL already bounds the
// damage if the frame is lost, so a user save must not wait on the daemon to acknowledge it. The
// payload layout is mirrored in server_utils/src/limiter/access.rs.

const invalidateAccessPayloadSize = 6

// InvalidateAllCompanyUsers is the wildcard for the userID argument. User IDs start at 1, so zero is
// free to mean "every cached user of this company".
const InvalidateAllCompanyUsers int32 = 0

var ErrAccessInvalidationNotConfigured = errors.New("server utils is not configured")

// InvalidateUserAccess tells the daemon to re-read one user's grants, or every user of a company.
//
// The error is worth logging and not worth failing a save over: the write that prompted this already
// succeeded, and the TTL is the fallback. A caller that treated this as fatal would roll back a
// correct user edit because a cache hint did not land.
func InvalidateUserAccess(ctx context.Context, companyID, userID int32) error {
	client := serverUtils()
	if client == nil {
		return ErrAccessInvalidationNotConfigured
	}
	payload, err := encodeAccessInvalidation(companyID, userID)
	if err != nil {
		return err
	}
	return client.send(ctx, opcodeInvalidateUserAccess, payload)
}

func encodeAccessInvalidation(companyID, userID int32) ([]byte, error) {
	if companyID <= 0 || companyID > 0xFF_FFFF {
		return nil, fmt.Errorf("company ID %d must fit positive uint24", companyID)
	}
	// Zero is the wildcard, so only the upper bound applies to the user.
	if userID < 0 || userID > 0xFF_FFFF {
		return nil, fmt.Errorf("user ID %d must fit uint24", userID)
	}
	payload := make([]byte, invalidateAccessPayloadSize)
	writeUint24(payload[0:3], uint32(companyID))
	writeUint24(payload[3:6], uint32(userID))
	return payload, nil
}
