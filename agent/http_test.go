package agent

import (
	"strings"
	"testing"
)

func TestResolveTargetKeepsTableBodySelectRowArgument(t *testing.T) {
	handleID, method, args, err := resolveTarget("38", "selectRow", []any{"38:100"})
	if err != nil {
		t.Fatal(err)
	}
	if handleID != 38 || method != "selectRow" || len(args) != 1 || args[0] != "38:100" {
		t.Fatalf("unexpected TableBody routing: handle=%d method=%q args=%v", handleID, method, args)
	}
}

func TestResolveTargetExplainsHowToRecoverFromInventedID(t *testing.T) {
	_, _, _, err := resolveTarget("save-button", "click", nil)
	if err == nil || !strings.Contains(err.Error(), "digits or digits:digits") || strings.Contains(err.Error(), "strconv") {
		t.Fatalf("unexpected invalid-id error: %v", err)
	}
}
