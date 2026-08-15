package core

import "testing"

// APIRouteNames is built by an init() that inverts APIRouteIDs, so what needs checking is that the
// inversion actually ran and is total — a package-level map that stayed empty would leave every
// dashboard label and every "r118" token unreadable, silently.
func TestAPIRouteNamesInvertsEveryID(t *testing.T) {
	if len(APIRouteNames) == 0 {
		t.Fatal("APIRouteNames is empty; the init() inversion did not run")
	}
	if len(APIRouteNames) != len(APIRouteIDs) {
		t.Fatalf("%d names for %d routes: two routes share an ID and one overwrote the other",
			len(APIRouteNames), len(APIRouteIDs))
	}
	for route, id := range APIRouteIDs {
		if APIRouteNames[id] != route {
			t.Errorf("id %d reads back as %q, not %q", id, APIRouteNames[id], route)
		}
	}
}

// Zero means unknown, so it must never be handed out — a route numbered zero would be
// indistinguishable from a 404 in every row that stored it.
func TestAPIRouteIDsAreUsableAsKeys(t *testing.T) {
	for route, id := range APIRouteIDs {
		if id <= 0 {
			t.Errorf("route %q has ID %d; zero and below are reserved for unknown", route, id)
		}
		if id > MaxAPIRouteID {
			t.Errorf("route %q has ID %d, above the declared maximum %d", route, id, MaxAPIRouteID)
		}
	}
	if APIRouteID("GET.this-route-does-not-exist") != 0 {
		t.Error("an unknown route resolved to something other than zero")
	}
}
