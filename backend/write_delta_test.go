package main

import (
	"maps"
	"testing"
)

func TestParseWriteDeltaRequest(t *testing.T) {
	route, query, err := parseWriteDeltaRequest("crm-opportunities~pipeline-id~3~up~12.4096.13.512")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedQuery := map[string]string{"pipeline-id": "3", "up": "12.4096.13.512"}
	if route != "crm-opportunities" || !maps.Equal(query, expectedQuery) {
		t.Fatalf("got route=%q query=%v", route, query)
	}

	route, query, err = parseWriteDeltaRequest("crm-lost-reasons")
	if err != nil || route != "crm-lost-reasons" || len(query) != 0 {
		t.Fatalf("bare route: route=%q query=%v err=%v", route, query, err)
	}

	for _, malformed := range []string{"", "~up~1", "crm-pipelines~up"} {
		if _, _, err := parseWriteDeltaRequest(malformed); err == nil {
			t.Fatalf("expected an error for %q", malformed)
		}
	}
}
