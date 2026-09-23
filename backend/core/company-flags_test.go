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

// A valued flag declared without a byte width would be saved against no bound at all, and an
// unknown type would be drawn by the browser as a field it does not know how to scale.
func TestLoadCompanyFlagsRefusesMalformedValuedFlag(t *testing.T) {
	for caseName, catalogContent := range map[string]string{
		"no bytes":      "[a]\n1 = { name = \"x\", type = \"i:3\" }\n",
		"odd bytes":     "[a]\n1 = { name = \"x\", type = \"i:3\", bytes = 3 }\n",
		"unknown type":  "[a]\n1 = { name = \"x\", type = \"f:3\", bytes = 2 }\n",
		"bad decimals":  "[a]\n1 = { name = \"x\", type = \"i:abc\", bytes = 2 }\n",
		"type not text": "[a]\n1 = { name = \"x\", type = 3, bytes = 2 }\n",
	} {
		LoadEmbeddedCompanyFlags([]byte(catalogContent))
		if embeddedCompanyFlags.loadErr == nil {
			t.Errorf("%s: the catalog must refuse to load", caseName)
		}
	}
	loadRealCompanyFlagsCatalog(t)
}

// Flag 7 is the real catalog's first valued flag: "i:3" over 2 bytes, so 2.5 is stored as 2500.
func TestValuedCompanyFlagRoundTrip(t *testing.T) {
	loadRealCompanyFlagsCatalog(t)

	sanitizedValues, err := SanitizeCompanyFlagValues([]CompanyFlagValue{{ID: 7, Value: 2500}})
	if err != nil {
		t.Fatalf("a declared valued flag was rejected: %v", err)
	}
	typedValue, isSet := GetCompanyFlagValue(sanitizedValues, 7)
	if !isSet || typedValue != 2.5 {
		t.Errorf("expected 2.5 set, got %v set=%v", typedValue, isSet)
	}
	if _, isSet := GetCompanyFlagValue(sanitizedValues, 2); isSet {
		t.Error("a checkbox flag never carries a value")
	}
}

// Empty and zero are one state: the rule falls back to its default in both, so nothing is stored.
func TestSanitizeCompanyFlagValuesDropsZero(t *testing.T) {
	loadRealCompanyFlagsCatalog(t)

	sanitizedValues, err := SanitizeCompanyFlagValues([]CompanyFlagValue{{ID: 7, Value: 0}})
	if err != nil {
		t.Fatalf("a zero must be dropped, not refused: %v", err)
	}
	if len(sanitizedValues) != 0 {
		t.Errorf("expected nothing stored, got %v", sanitizedValues)
	}
}

func TestSanitizeCompanyFlagValuesRefusesOutOfRange(t *testing.T) {
	loadRealCompanyFlagsCatalog(t)

	if _, err := SanitizeCompanyFlagValues([]CompanyFlagValue{{ID: 7, Value: 40000}}); err == nil {
		t.Error("a value wider than the declared 2 bytes must be refused, not truncated")
	}
}

// The two lists are not interchangeable: a valued id in Flags would be a second on/off nothing
// reads, and a checkbox id in FlagValues a number nothing scales.
func TestCompanyFlagsRefuseTheOtherKind(t *testing.T) {
	loadRealCompanyFlagsCatalog(t)

	if _, err := SanitizeCompanyFlags([]int16{7}); err == nil {
		t.Error("a valued flag must not be storable as a checked checkbox")
	}
	if _, err := SanitizeCompanyFlagValues([]CompanyFlagValue{{ID: 2, Value: 10}}); err == nil {
		t.Error("a checkbox flag must not be storable with a value")
	}
}
