package exec

import "testing"

func TestObservabilityLogViewRebuildCommandIsRegistered(t *testing.T) {
	if ExecHandlers["fn-rebuild-observability-log-view"] == nil {
		t.Fatal("observability log view rebuild command is not registered")
	}
}
