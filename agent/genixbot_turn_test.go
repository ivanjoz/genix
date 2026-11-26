package agent

import (
	"encoding/json"
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

func TestOptionsText(t *testing.T) {
	cases := map[string]string{
		`[]`:   "sin resultados",
		`null`: "",
		`[{"ID":1,"Value":"A"},{"ID":2,"Value":"B"}]`: "[1] A, [2] B",
		`[{"ID":1,"Value":"A"},{"ID":2,"Value":"A"},{"ID":3,"Value":"A"},{"ID":4,"Value":"A"},{"ID":5,"Value":"A"},{"ID":6,"Value":"A"},{"ID":7,"Value":"A"},{"ID":8,"Value":"A"},{"ID":9,"Value":"A"},{"ID":10,"Value":"A"},{"ID":11,"Value":"A"},{"ID":12,"Value":"A"}]`: "[1] A, [2] A, [3] A, [4] A, [5] A, [6] A, [7] A, [8] A, [9] A, [10] A (+2 más)",
	}
	for value, want := range cases {
		if got := optionsText(json.RawMessage(value)); got != want {
			t.Errorf("optionsText(%s) = %q, want %q", value, got, want)
		}
	}
}
