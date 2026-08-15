package discovery

import (
	"encoding/json"
	"os"
	"testing"

	"app/agent/routing"
)

type plannerFixture struct {
	Name          string              `json:"name"`
	Surface       routing.SurfaceKind `json:"surface"`
	Request       string              `json:"request"`
	Goal          Goal                `json:"goal"`
	Operation     Operation           `json:"operation"`
	FeatureSearch bool                `json:"feature_search"`
	ToolSearch    bool                `json:"tool_search"`
	Delivery      DeliveryPreference  `json:"delivery"`
}

func TestPlannerFixtureSetIsStructurallyValid(t *testing.T) {
	fixtureJSON, err := os.ReadFile("testdata/planner_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	fixtures := []plannerFixture{}
	if err := json.Unmarshal(fixtureJSON, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) < 7 {
		t.Fatalf("expected at least seven discovery cases, got %d", len(fixtures))
	}
	seen := map[string]bool{}
	for _, fixture := range fixtures {
		if fixture.Name == "" || fixture.Request == "" || seen[fixture.Name] {
			t.Fatalf("invalid or duplicate fixture %q", fixture.Name)
		}
		seen[fixture.Name] = true
		if !validGoal(fixture.Goal) || !validOperation(fixture.Operation) || !validDelivery(fixture.Delivery) {
			t.Fatalf("invalid expected discovery contract in %q", fixture.Name)
		}
		if !validSurface(routing.SurfaceContext{Kind: fixture.Surface}) {
			t.Fatalf("invalid surface in %q", fixture.Name)
		}
	}
}
