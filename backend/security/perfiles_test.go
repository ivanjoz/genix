package security

import (
	"os"
	"testing"

	"app/core"
	coreTypes "app/core/types"
)

// The catalogue is embedded into package main, which a security test cannot reach, so the test
// binary loads the same file off disk. Reading the real access.toml rather than a fixture is the
// point: what is asserted here is that the validator agrees with the file that ships.
func loadRealAccessCatalog(t *testing.T) {
	t.Helper()
	accessCatalogContent, err := os.ReadFile("../access.toml")
	if err != nil {
		t.Fatalf("could not read access.toml: %v", err)
	}
	if helper := core.LoadEmbeddedAccessList(accessCatalogContent); helper == nil {
		t.Fatal("the access catalogue did not load")
	}
	// Acceso 10 (Punto de Venta) is the only access declaring sub-accesses today, and every case
	// below is written against it. If that stops being true the cases need rewriting, not patching.
	accessInfo, found := core.GetEmbeddedAccessHelper().GetAccesoInfo(10)
	if !found || accessInfo.SubAccesosMask != 0b110 {
		t.Fatalf("acceso 10 declares mask %07b; the cases below assume subs 2 and 3", accessInfo.SubAccesosMask)
	}
}

// NEVER trust the client. Each of these stores a bit into every affected user's blob that no UI can
// show and no handler can name, so the handler rejects rather than dropping: a profile silently
// saved without what the operator just ticked is worse than an error.
func TestValidateAccesoGrantsRejections(t *testing.T) {
	loadRealAccessCatalog(t)

	for name, grantRecords := range map[string][]coreTypes.AccesoGrantRecord{
		// The access exists and is granted, but never declared sub-access 9.
		"sub-acceso the catalogue never declared": {
			{AccesoID: 10, Nivel: 4, SubAccesos: []uint8{9}},
		},
		"acceso that does not exist": {
			{AccesoID: 9999, Nivel: 1},
		},
		// "Todos" is synthesized, never declared, so it is allowed on any access that offers
		// sub-accesses at all — and refused on one that offers none.
		"todos on an acceso with no sub-accesos": {
			{AccesoID: 3, Nivel: 4, SubAccesos: []uint8{core.SubAccesoTodosID}},
		},
		"sub-acceso past the ceiling": {
			{AccesoID: 10, Nivel: 4, SubAccesos: []uint8{core.MaxSubAccesoID + 1}},
		},
		"nivel outside 1..4": {
			{AccesoID: 10, Nivel: 5},
		},
		// Which nivel survived a duplicate would depend on the order the records merged in.
		"the same acceso granted twice": {
			{AccesoID: 10, Nivel: 1},
			{AccesoID: 10, Nivel: 4},
		},
	} {
		if err := ValidateAccesoGrants(grantRecords); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestValidateAccesoGrantsAccepted(t *testing.T) {
	loadRealAccessCatalog(t)

	for name, grantRecords := range map[string][]coreTypes.AccesoGrantRecord{
		"nothing granted at all": nil,
		"no sub-accesos":         {{AccesoID: 10, Nivel: 4}},
		"both declared sub-accesos": {
			{AccesoID: 10, Nivel: 4, SubAccesos: []uint8{2, 3}},
		},
		"todos on the acceso that declares sub-accesos": {
			{AccesoID: 10, Nivel: 1, SubAccesos: []uint8{core.SubAccesoTodosID}},
		},
		"several accesos": {
			{AccesoID: 3, Nivel: 1},
			{AccesoID: 10, Nivel: 4, SubAccesos: []uint8{2}},
		},
	} {
		if err := ValidateAccesoGrants(grantRecords); err != nil {
			t.Errorf("%s was rejected: %v", name, err)
		}
	}
}

// The merge is where a profile grant and a direct user grant of the same access meet. Widest wins
// on the nivel and the sub-accesses union, which is what lets a user be given one extra sub-access
// without forking the profile they share with everyone else.
func TestAddAccesoGrantToMergeTakesTheWidest(t *testing.T) {
	grantsByAccesoID := map[int32]*core.AccesoGrant{}

	addAccesoGrantToMerge(grantsByAccesoID, coreTypes.AccesoGrantRecord{
		AccesoID: 10, Nivel: 1, SubAccesos: []uint8{2},
	})
	addAccesoGrantToMerge(grantsByAccesoID, coreTypes.AccesoGrantRecord{
		AccesoID: 10, Nivel: 4, SubAccesos: []uint8{3},
	})

	merged := grantsByAccesoID[10]
	if merged == nil {
		t.Fatal("acceso 10 did not survive the merge")
	}
	if merged.Nivel != 4 {
		t.Errorf("nivel is %d, expected the higher 4", merged.Nivel)
	}
	if merged.SubMask != 0b110 {
		t.Errorf("sub mask is %b, expected subs 2 and 3 united", merged.SubMask)
	}
}

// A corrupt value must never widen a grant, so a nivel outside 1..4 normalizes down to 1 and an
// out-of-range sub-access is dropped rather than shifted into some other sub's bit.
func TestAddAccesoGrantToMergeNormalizesDown(t *testing.T) {
	grantsByAccesoID := map[int32]*core.AccesoGrant{}

	addAccesoGrantToMerge(grantsByAccesoID, coreTypes.AccesoGrantRecord{
		AccesoID: 10, Nivel: 9, SubAccesos: []uint8{0, core.MaxSubAccesoID + 1},
	})

	merged := grantsByAccesoID[10]
	if merged.Nivel != 1 {
		t.Errorf("nivel is %d, expected it normalized down to 1", merged.Nivel)
	}
	if merged.SubMask != 0 {
		t.Errorf("sub mask is %b, expected the out-of-range subs dropped", merged.SubMask)
	}
}
