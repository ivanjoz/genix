package security

import (
	"os"
	"testing"

	"app/core"
	"app/security/types"
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
func TestValidateProfileSubAccesosRejections(t *testing.T) {
	loadRealAccessCatalog(t)

	for name, profile := range map[string]types.Profile{
		// The access exists and is granted, but never declared sub-access 9.
		"sub-acceso the catalogue never declared": {
			Accesos: []int32{104}, SubAccesos: []int32{1009},
		},
		"acceso that does not exist": {
			Accesos: []int32{104}, SubAccesos: []int32{999902},
		},
		// A sub-access qualifies a permission rather than granting one, so one with no parent has
		// nothing to qualify. The merge would drop it anyway; accepting it here would store a grant
		// that silently does nothing.
		"sub-acceso without its parent acceso": {
			Accesos: []int32{34}, SubAccesos: []int32{1002},
		},
		// "Todos" is synthesized, never declared, so it is allowed on any access that offers
		// sub-accesses at all — and refused on one that offers none.
		"todos on an acceso with no sub-accesos": {
			Accesos: []int32{34}, SubAccesos: []int32{300 + core.SubAccesoTodosID},
		},
		"sub-acceso past the ceiling": {
			Accesos: []int32{104}, SubAccesos: []int32{1000 + core.MaxSubAccesoID + 1},
		},
	} {
		if err := validateProfileSubAccesos(profile); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestValidateProfileSubAccesosAccepted(t *testing.T) {
	loadRealAccessCatalog(t)

	for name, profile := range map[string]types.Profile{
		"no sub-accesos at all":     {Accesos: []int32{104}},
		"both declared sub-accesos": {Accesos: []int32{104}, SubAccesos: []int32{1002, 1003}},
		"todos on the acceso that declares sub-accesos": {
			Accesos: []int32{101}, SubAccesos: []int32{1000 + core.SubAccesoTodosID},
		},
	} {
		if err := validateProfileSubAccesos(profile); err != nil {
			t.Errorf("%s was rejected: %v", name, err)
		}
	}
}
