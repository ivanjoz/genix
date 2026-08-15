package llm

import (
	"strings"
	"testing"
)

func TestSystemPromptRequiresCreateFormPrefillAndSeparateSaveConfirmation(t *testing.T) {
	for _, instruction := range []string{
		"open its New/Create form",
		"fill every field supported by information the user already provided",
		"NOT confirmation to save it",
	} {
		if !strings.Contains(SystemPromptChat, instruction) {
			t.Fatalf("create-record instruction %q missing from system prompt", instruction)
		}
	}
}

func TestSystemPromptReusesFreshToolPageState(t *testing.T) {
	instruction := "If navigate/invoke_batch returns PAGE SNAPSHOT, use it; call get_page only if it is missing, failed, or later changed."
	if !strings.Contains(SystemPromptChat, instruction) {
		t.Fatal("fresh tool page-state instruction missing from system prompt")
	}
}

func TestPageSnapshotGrammarSelectsRowsThroughTableBody(t *testing.T) {
	for _, instruction := range []string{
		`<TableBody id="38" methods="selectRow"><Row id="38:100"/></TableBody>`,
		`invoke_batch [{ID:"38", Method:"selectRow", Args:["38:100"]}]`,
	} {
		if !strings.Contains(PageSnapshotGrammar, instruction) {
			t.Fatalf("TableBody row-selection instruction %q missing", instruction)
		}
	}
	if strings.Contains(PageSnapshotGrammar, `<TableRow`) {
		t.Fatal("legacy TableRow instruction remains in snapshot grammar")
	}
}

func TestInvokeBatchIDSchemaRequiresNumericSnapshotID(t *testing.T) {
	properties := InvokeBatchTool.Function.Parameters["properties"].(map[string]any)
	invocations := properties["invocations"].(map[string]any)
	items := invocations["items"].(map[string]any)
	invocationProperties := items["properties"].(map[string]any)
	idSchema := invocationProperties["ID"].(map[string]any)
	if idSchema["pattern"] != AgentTargetIDPattern {
		t.Fatalf("invoke_batch ID pattern = %v, want %q", idSchema["pattern"], AgentTargetIDPattern)
	}

	for _, validID := range []string{"1", "47", "117:104"} {
		if !IsValidAgentTargetID(validID) {
			t.Errorf("valid agent target ID rejected: %q", validID)
		}
	}
	for _, invalidID := range []string{"", "save-button", "47:", ":104", "47:104:2", "-1", " 47"} {
		if IsValidAgentTargetID(invalidID) {
			t.Errorf("invalid agent target ID accepted: %q", invalidID)
		}
	}
}
