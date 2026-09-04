package main

import (
	"app/core"
	"testing"
)

// The daemon answers in SLOTS, because it holds no copy of access.toml and cannot know which access
// a slot stands for. mapGrantedSubAccesos is the whole translation, and it runs on this side
// precisely because this is the side that embeds the catalogue.
func TestMapGrantedSubAccesosTranslatesSlotsToAccesoIDs(t *testing.T) {
	accessInfos := []core.AccessInfo{{ID: 10}, {ID: 3}, {ID: 21}}

	subAccesos := mapGrantedSubAccesos(&core.AccessGrant{
		// Slots 0 and 2 granted; slot 1 was not required by this route's caller.
		GrantedSlots: 0b101,
		SubAccesoBytes: map[int][]byte{
			0: {0x05},       // subs 1 and 3
			2: {0x81, 0x20}, // subs 1 and 13
		},
	}, accessInfos)

	if got, want := len(subAccesos), 2; got != want {
		t.Fatalf("mapped %d accesses; want %d (%v)", got, want, subAccesos)
	}
	if got, want := subAccesos[10], uint16(0b101); got != want {
		t.Errorf("acceso 10 mask = %013b; want %013b", got, want)
	}
	if got, want := subAccesos[21], uint16(0b1000000000001); got != want {
		t.Errorf("acceso 21 mask = %013b; want %013b", got, want)
	}
	// Slot 1 is not in the granted mask, so acceso 3 never authorized this route and must be
	// absent rather than present with an empty mask — the two are different answers.
	if _, present := subAccesos[3]; present {
		t.Error("an ungranted slot must not produce an entry")
	}
}

// A granted access with no sub-accesses is a real case and the common one: it enters the map with an
// empty mask. The distinction the map carries is "holds none of them" versus "this access did not
// authorize the route at all", and only the presence of the key states it.
func TestMapGrantedSubAccesosKeepsAccesosWithoutSubBytes(t *testing.T) {
	subAccesos := mapGrantedSubAccesos(
		&core.AccessGrant{GrantedSlots: 0b1}, []core.AccessInfo{{ID: 7}})

	subMask, present := subAccesos[7]
	if !present {
		t.Fatal("a granted access must be present even with no sub bytes")
	}
	if subMask != 0 {
		t.Errorf("mask = %013b; want zero", subMask)
	}

	usuarioToken := core.UsuarioToken{SubAccesos: subAccesos}
	if usuarioToken.HasSubAcceso(7, 2) {
		t.Error("an access granted with no sub-accesses must hold none")
	}
	if usuarioToken.HasSubAcceso(9, 2) {
		t.Error("an access that never authorized this route must hold no sub-accesses")
	}
}

// resolveRouteAccess sends no frame at all for an unmapped GET, a selfServiceRoutes route, or user
// 1, so there is no grant to translate. A handler on such a route depends on no sub-access, and the
// map has to be empty rather than permissive.
func TestMapGrantedSubAccesosWithoutAGrant(t *testing.T) {
	subAccesos := mapGrantedSubAccesos(nil, []core.AccessInfo{{ID: 10}, {ID: 3}})
	if subAccesos != nil {
		t.Fatalf("no grant must map to no sub-accesses; got %v", subAccesos)
	}

	usuarioToken := core.UsuarioToken{SubAccesos: subAccesos}
	for _, subAccesoID := range []int32{core.SubAccesoTodosID, 2, core.MaxSubAccesoID} {
		if usuarioToken.HasSubAcceso(10, subAccesoID) {
			t.Errorf("sub-acceso %d must not be held without a grant", subAccesoID)
		}
	}
}

// The frame was already accepted when this runs, so the access IS granted; only its detail could
// not be read. Denying the sub-accesses without denying the route is the conservative half of that,
// and it must not drop the entry — dropping it would say the access never authorized the route.
func TestMapGrantedSubAccesosSurvivesUnreadableSubBytes(t *testing.T) {
	subAccesos := mapGrantedSubAccesos(&core.AccessGrant{
		GrantedSlots:   0b1,
		SubAccesoBytes: map[int][]byte{0: {0x81}}, // dangling MORE, nothing follows
	}, []core.AccessInfo{{ID: 10}})

	subMask, present := subAccesos[10]
	if !present {
		t.Fatal("an authorized access must stay in the map even when its sub bytes are corrupt")
	}
	if subMask != 0 {
		t.Errorf("mask = %013b; want zero", subMask)
	}
}
