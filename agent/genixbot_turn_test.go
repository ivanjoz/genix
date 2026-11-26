package agent

import (
	"testing"

	"genix-agent/genixbot"
)

func TestActionError(t *testing.T) {
	cases := map[string]string{
		"unknown handle id: 566":                           genixbot.ErrorUnknownID,
		"handle 12 (Checkbox) does not implement setValue": "Checkbox no tiene el método setValue",
		"opción 9 no encontrada":                           "opción 9 no encontrada",
	}
	for browserError, want := range cases {
		if got := actionError(browserError); got != want {
			t.Errorf("actionError(%q) = %q, want %q", browserError, got, want)
		}
	}
}
