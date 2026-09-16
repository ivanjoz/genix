package core

import (
	"os"
	"testing"
)

// The real catalog, not a fixture: a flag declared without a name or with a reused id breaks the
// company parameters save for every tenant, and the binary embeds this exact file.
func loadRealCompanyFlagsCatalog(t *testing.T) {
	t.Helper()
	companyFlagsContent, err := os.ReadFile("../company_flags.toml")
	if err != nil {
		t.Fatalf("could not read company_flags.toml: %v", err)
	}
	LoadEmbeddedCompanyFlags(companyFlagsContent)
	if embeddedCompanyFlags.loadErr != nil {
		t.Fatalf("company_flags.toml did not load: %v", embeddedCompanyFlags.loadErr)
	}
}

func TestCompanyFlagsCatalogLoads(t *testing.T) {
	loadRealCompanyFlagsCatalog(t)

	if _, flagFound := GetCompanyFlagName(1); !flagFound {
		t.Error("flag 1 is declared in the catalog but did not load")
	}
	if _, flagFound := GetCompanyFlagName(999); flagFound {
		t.Error("flag 999 is not declared, so it must not resolve")
	}
}

func TestSanitizeCompanyFlagsDeduplicatesAndSorts(t *testing.T) {
	loadRealCompanyFlagsCatalog(t)

	sanitizedFlags, err := SanitizeCompanyFlags([]int16{3, 1, 3})
	if err != nil {
		t.Fatalf("declared flags were rejected: %v", err)
	}
	if len(sanitizedFlags) != 2 || sanitizedFlags[0] != 1 || sanitizedFlags[1] != 3 {
		t.Errorf("expected [1 3], got %v", sanitizedFlags)
	}
}

func TestSanitizeCompanyFlagsRejectsUnknownID(t *testing.T) {
	loadRealCompanyFlagsCatalog(t)

	if _, err := SanitizeCompanyFlags([]int16{1, 999}); err == nil {
		t.Error("an id absent from the catalog must be refused, not stored")
	}
}

// A duplicated id across sections is the format's one hazard: the stored list would mean two
// different rules, so the loader has to refuse the whole catalog.
func TestLoadCompanyFlagsRefusesDuplicatedID(t *testing.T) {
	LoadEmbeddedCompanyFlags([]byte("[a]\n1 = { name = \"x\" }\n\n[b]\n1 = { name = \"y\" }\n"))
	if embeddedCompanyFlags.loadErr == nil {
		t.Error("an id declared in two sections must fail the load")
	}
	// Leave the package-level catalog in its real state for whatever test runs next.
	loadRealCompanyFlagsCatalog(t)
}
