package core

import (
	"errors"
	"testing"
)

func resetAccessGrantCache() {
	clear(accessGrantCache)
	clear(synchronousSubjects)
}

func TestAccessGrantCacheServesAStoredGrant(t *testing.T) {
	resetAccessGrantCache()
	requiredAccess := []uint16{41, 42}
	if _, found := LookupAccessGrant(7, 3, requiredAccess); found {
		t.Fatal("an empty cache must send the request to fareward")
	}
	grant := &AccessGrant{GrantedSlots: 0b01}
	RecordAccessCheckOutcome(7, 3, requiredAccess, grant, nil)
	if cached, found := LookupAccessGrant(7, 3, requiredAccess); !found || cached != grant {
		t.Fatalf("stored grant not served: found=%t", found)
	}
	// Another access set on the same user is a different route decision.
	if _, found := LookupAccessGrant(7, 3, []uint16{41}); found {
		t.Fatal("a grant must only answer the access set it was issued for")
	}
	// An ungated route needs no grant, only a subject in good standing.
	if _, found := LookupAccessGrant(7, 99, nil); !found {
		t.Fatal("an ungated route must not wait for fareward")
	}
}

func TestCompanyCreditRefusalMakesEveryUserSynchronousUntilAllowed(t *testing.T) {
	resetAccessGrantCache()
	requiredAccess := []uint16{41}
	RecordAccessCheckOutcome(7, 3, requiredAccess, &AccessGrant{GrantedSlots: 1}, nil)

	RecordAccessCheckOutcome(7, 5, nil, nil, &CreditLimitExceeded{Company: true})
	if _, found := LookupAccessGrant(7, 3, requiredAccess); found {
		t.Fatal("a company refusal must make its other users wait for fareward too")
	}
	if _, found := LookupAccessGrant(8, 3, nil); !found {
		t.Fatal("another company must not be affected")
	}

	// The next synchronous request that fareward allows lifts the mark.
	RecordAccessCheckOutcome(7, 3, requiredAccess, &AccessGrant{GrantedSlots: 1}, nil)
	if _, found := LookupAccessGrant(7, 3, requiredAccess); !found {
		t.Fatal("an allowed verdict must clear the synchronous mark")
	}
}

func TestUserRefusalAndUnreachableDaemonOnlyPinTheUser(t *testing.T) {
	resetAccessGrantCache()
	RecordAccessCheckOutcome(7, 3, nil, nil, &CreditLimitExceeded{Company: false})
	RecordAccessCheckOutcome(7, 4, nil, nil, errors.New("fareward unavailable"))
	for _, userID := range []int32{3, 4} {
		if _, found := LookupAccessGrant(7, userID, nil); found {
			t.Fatalf("user %d must wait for fareward", userID)
		}
	}
	if _, found := LookupAccessGrant(7, 5, nil); !found {
		t.Fatal("a user refusal must not reach the rest of the company")
	}
}

func TestOperatorCreditRefusalIsNotPinned(t *testing.T) {
	resetAccessGrantCache()
	RecordAccessCheckOutcome(OperatorCompanyID, 3, nil, nil, &CreditLimitExceeded{Company: true})
	if _, found := LookupAccessGrant(OperatorCompanyID, 3, nil); !found {
		t.Fatal("the operator's tolerated credit refusal must not slow its requests")
	}
}

func TestRevokedAccessDropsTheCachedGrant(t *testing.T) {
	resetAccessGrantCache()
	requiredAccess := []uint16{41}
	RecordAccessCheckOutcome(7, 3, requiredAccess, &AccessGrant{GrantedSlots: 1}, nil)
	RecordAccessCheckOutcome(7, 3, requiredAccess, nil, &AccessDenied{})
	if _, found := LookupAccessGrant(7, 3, requiredAccess); found {
		t.Fatal("a denial must drop the grant so the next request is decided by fareward")
	}
}

func TestDropCachedAccessGrantsByUserAndByCompany(t *testing.T) {
	resetAccessGrantCache()
	requiredAccess := []uint16{41}
	for _, userID := range []int32{3, 4} {
		RecordAccessCheckOutcome(7, userID, requiredAccess, &AccessGrant{GrantedSlots: 1}, nil)
	}
	RecordAccessCheckOutcome(8, 3, requiredAccess, &AccessGrant{GrantedSlots: 1}, nil)

	DropCachedAccessGrants(7, 3)
	if _, found := LookupAccessGrant(7, 3, requiredAccess); found {
		t.Fatal("user 3 grant must be dropped")
	}
	if _, found := LookupAccessGrant(7, 4, requiredAccess); !found {
		t.Fatal("user 4 grant must survive a single-user drop")
	}

	DropCachedAccessGrants(7, InvalidateAllCompanyUsers)
	if _, found := LookupAccessGrant(7, 4, requiredAccess); found {
		t.Fatal("a company-wide drop must reach every user")
	}
	if _, found := LookupAccessGrant(8, 3, requiredAccess); !found {
		t.Fatal("a company-wide drop must not reach another company")
	}
}
