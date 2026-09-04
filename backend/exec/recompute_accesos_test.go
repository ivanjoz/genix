package exec

import "testing"

// Deploy-blocking: without this command every stored accesos_computed stays little-endian and the
// new reader denies every user. A rename that lost the registration would only surface at deploy
// time, on a database nobody can authorize against.
func TestRecomputeUserAccesosCommandIsRegistered(t *testing.T) {
	if ExecHandlers["fn-recompute-user-accesos"] == nil {
		t.Fatal("the user accesos recompute command is not registered")
	}
}
