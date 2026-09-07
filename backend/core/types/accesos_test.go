package types

import (
	"encoding/json"
	"testing"

	"github.com/ivanjoz/colbin"
)

// The bug this guards against is silent in one direction: encoding/json accepts `[2]` into a byte
// slice but writes it back as the base64 string "Ag==", so a sub-access saved correctly came back
// to the browser as four characters of base64.
func TestAccesoGrantJSONRoundTrip(t *testing.T) {
	grantRecords := []AccesoGrantRecord{
		{AccesoID: 10, Nivel: 4, SubAccesos: SubAccesoIDs{2, 3}},
		{AccesoID: 3, Nivel: 1},
	}

	encodedJSON, err := json.Marshal(grantRecords)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	const expectedJSON = `[{"AccesoID":10,"Nivel":4,"SubAccesos":[2,3]},{"AccesoID":3,"Nivel":1}]`
	if string(encodedJSON) != expectedJSON {
		t.Fatalf("json is %s, expected %s", encodedJSON, expectedJSON)
	}

	decodedGrants := []AccesoGrantRecord{}
	if err := json.Unmarshal(encodedJSON, &decodedGrants); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decodedGrants) != 2 || decodedGrants[0].SubAccesos[1] != 3 || decodedGrants[1].SubAccesos != nil {
		t.Fatalf("round trip lost data: %+v", decodedGrants)
	}
}

// The column is a blob written by colbin, so what the database holds has to survive the same trip.
func TestAccesoGrantColbinRoundTrip(t *testing.T) {
	grantRecords := []AccesoGrantRecord{
		{AccesoID: 10, Nivel: 4, SubAccesos: SubAccesoIDs{2, 3}},
		{AccesoID: 3, Nivel: 1},
	}

	blob, err := colbin.Marshal(grantRecords)
	if err != nil {
		t.Fatalf("colbin marshal failed: %v", err)
	}

	decodedGrants := []AccesoGrantRecord{}
	if err := colbin.Unmarshal(blob, &decodedGrants); err != nil {
		t.Fatalf("colbin unmarshal failed: %v", err)
	}

	if len(decodedGrants) != 2 {
		t.Fatalf("got %d grants, expected 2", len(decodedGrants))
	}
	if decodedGrants[0].AccesoID != 10 || decodedGrants[0].Nivel != 4 {
		t.Errorf("first grant is %+v", decodedGrants[0])
	}
	if len(decodedGrants[0].SubAccesos) != 2 ||
		decodedGrants[0].SubAccesos[0] != 2 || decodedGrants[0].SubAccesos[1] != 3 {
		t.Errorf("sub-accesos are %v, expected [2 3]", decodedGrants[0].SubAccesos)
	}
	if len(decodedGrants[1].SubAccesos) != 0 {
		t.Errorf("an access with no sub-accesos came back with %v", decodedGrants[1].SubAccesos)
	}
}
