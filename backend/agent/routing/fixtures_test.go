package routing

import (
	"encoding/json"
	"os"
	"testing"
)

type classifierFixture struct {
	Name                 string              `json:"name"`
	Surface              SurfaceKind         `json:"surface"`
	Request              string              `json:"request"`
	ExpectedIntent       Intent              `json:"expected_intent"`
	ResponseLanguage     Language            `json:"response_language"`
	ExpectedCapability   CapabilityName      `json:"expected_capability"`
	ExpectedBuilderScope BuilderContextScope `json:"expected_builder_scope"`
	ExpectedOffsets      []int               `json:"expected_offsets"`
}

func TestClassifierFixtureSetIsStructurallyValid(t *testing.T) {
	fixtureJSON, err := os.ReadFile("testdata/classifier_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	fixtures := []classifierFixture{}
	if err := json.Unmarshal(fixtureJSON, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) < 16 {
		t.Fatalf("expected at least 16 labeled cases, got %d", len(fixtures))
	}
	seenNames := map[string]bool{}
	for _, fixture := range fixtures {
		if fixture.Name == "" || fixture.Request == "" || seenNames[fixture.Name] {
			t.Fatalf("invalid or duplicate fixture name %q", fixture.Name)
		}
		seenNames[fixture.Name] = true
		if !validSurfaceKind(fixture.Surface) || !validIntent(fixture.ExpectedIntent) {
			t.Fatalf("invalid expected route in fixture %q", fixture.Name)
		}
		if fixture.ResponseLanguage != LanguageSpanish && fixture.ResponseLanguage != LanguageEnglish {
			t.Fatalf("invalid response language in fixture %q", fixture.Name)
		}
		if fixture.ExpectedCapability != "" && !KnownCapability(fixture.ExpectedCapability) {
			t.Fatalf("unknown capability in fixture %q", fixture.Name)
		}
		if fixture.ExpectedBuilderScope != BuilderScopeNone && fixture.ExpectedBuilderScope != BuilderScopeFullPage && fixture.ExpectedBuilderScope != BuilderScopeSelectedSection {
			t.Fatalf("invalid builder scope in fixture %q", fixture.Name)
		}
	}
}
