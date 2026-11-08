package core

import (
	"errors"
	"sync"
	"time"
)

// The access grant cache takes fareward's round trip off the request path on a long-running
// backend (VPS). Every authenticated request used to wait for one frame that both charged credits
// and returned the user's grant for the route; now a request whose grant is cached runs straight
// away while the same frame is sent in the background, and the reply refreshes the cache.
//
// Enforcement becomes one request behind instead of exact:
//   - A revoked access reaches the cache through the next background reply (or immediately through
//     DropCachedAccessGrants, which every write of users.accesos_computed already triggers).
//   - A credit refusal marks the user, or the whole company, "synchronous": its next requests wait
//     for fareward again and get the exact verdict, until one of them is allowed.
//
// Not used on Lambda: a goroutine still running after the response is frozen with the sandbox, so
// a background charge there could arrive late or never.

// accessGrantTTL bounds how stale a grant can be when it is used after a quiet period: a busy user
// refreshes it on every request, an idle one goes back to the synchronous path after this.
const accessGrantTTL = 60 * time.Second

// maxCachedAccessGrants is a memory ceiling, not a tuning knob: reaching it empties the cache,
// which only costs one synchronous request per active (user, route access set).
const maxCachedAccessGrants = 100_000

type accessGrantKey struct {
	companyID      int32
	userID         int32
	requiredAccess [MaxRequiredAccess]uint16
}

type cachedAccessGrant struct {
	grant    *AccessGrant
	storedAt time.Time
}

// accessSubject with userID == AllCompanyUsers (0) is the whole company.
type accessSubject struct {
	companyID int32
	userID    int32
}

var (
	accessGrantCacheMu sync.RWMutex
	accessGrantCache   = map[accessGrantKey]cachedAccessGrant{}
	// synchronousSubjects must wait for fareward's verdict on every request: their last background
	// charge was refused or could not be answered.
	synchronousSubjects = map[accessSubject]struct{}{}
)

func makeAccessGrantKey(companyID, userID int32, requiredAccess []uint16) accessGrantKey {
	key := accessGrantKey{companyID: companyID, userID: userID}
	copy(key.requiredAccess[:], requiredAccess)
	return key
}

// LookupAccessGrant reports whether the request may skip waiting for fareward, and with which
// grant. A route that requires no access needs no cached grant, only a subject in good standing.
func LookupAccessGrant(companyID, userID int32, requiredAccess []uint16) (*AccessGrant, bool) {
	accessGrantCacheMu.RLock()
	defer accessGrantCacheMu.RUnlock()
	if _, isSynchronous := synchronousSubjects[accessSubject{companyID, userID}]; isSynchronous {
		return nil, false
	}
	if _, isSynchronous := synchronousSubjects[accessSubject{companyID, AllCompanyUsers}]; isSynchronous {
		return nil, false
	}
	if len(requiredAccess) == 0 {
		return nil, true
	}
	cached, found := accessGrantCache[makeAccessGrantKey(companyID, userID, requiredAccess)]
	if !found || time.Since(cached.storedAt) > accessGrantTTL {
		return nil, false
	}
	return cached.grant, true
}

// RecordAccessCheckOutcome feeds one fareward verdict — synchronous or background — back into the
// cache. An allowed verdict stores the grant and clears the subject's synchronous mark; a refusal
// drops or marks, so the next request is decided by fareward again.
func RecordAccessCheckOutcome(
	companyID, userID int32, requiredAccess []uint16, grant *AccessGrant, checkErr error,
) {
	accessGrantCacheMu.Lock()
	defer accessGrantCacheMu.Unlock()

	if checkErr == nil {
		delete(synchronousSubjects, accessSubject{companyID, userID})
		delete(synchronousSubjects, accessSubject{companyID, AllCompanyUsers})
		if len(requiredAccess) == 0 {
			return
		}
		if len(accessGrantCache) >= maxCachedAccessGrants {
			clear(accessGrantCache)
		}
		accessGrantCache[makeAccessGrantKey(companyID, userID, requiredAccess)] = cachedAccessGrant{
			grant: grant, storedAt: time.Now(),
		}
		return
	}

	// The operator's credit refusals never reach its requests, so they must not slow them either.
	if TolerateCreditRefusal(companyID, checkErr) {
		return
	}
	if IsAccessDeniedError(checkErr) {
		// No synchronous mark needed: without a cached grant the next gated request waits anyway.
		dropCachedAccessGrantsLocked(companyID, userID)
		return
	}
	// A credit refusal names its scope; anything else (fareward unreachable) is pinned to the user.
	refusedSubject := accessSubject{companyID, userID}
	var limitExceeded *CreditLimitExceeded
	if errors.As(checkErr, &limitExceeded) && limitExceeded.Company {
		refusedSubject.userID = AllCompanyUsers
	}
	synchronousSubjects[refusedSubject] = struct{}{}
}

// DropCachedAccessGrants forgets a user's grants, or every grant of the company when userID is
// AllCompanyUsers. Called next to the fareward grants push, on every access write.
func DropCachedAccessGrants(companyID, userID int32) {
	accessGrantCacheMu.Lock()
	defer accessGrantCacheMu.Unlock()
	dropCachedAccessGrantsLocked(companyID, userID)
}

func dropCachedAccessGrantsLocked(companyID, userID int32) {
	for key := range accessGrantCache {
		if key.companyID == companyID && (userID == AllCompanyUsers || key.userID == userID) {
			delete(accessGrantCache, key)
		}
	}
}
