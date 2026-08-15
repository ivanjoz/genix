package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"app/agent/llm"
)

func TestFormatPromptExchangeIncludesPlannerResponseAndExecutorTools(t *testing.T) {
	messages := []llm.Message{{Role: "user", Content: "crear cliente"}}
	plannerLog := formatPromptExchange(messages, nil, `{"goal":"manage_record"}`, "")
	if !strings.Contains(plannerLog, "=== RAW RESPONSE ===") || !strings.Contains(plannerLog, `"goal":"manage_record"`) {
		t.Fatalf("planner response missing from log: %s", plannerLog)
	}

	executorTools := []llm.Tool{{Type: "function", Function: llm.ToolFunction{Name: "navigate"}}}
	executorLog := formatPromptExchange(messages, executorTools, "", "")
	if !strings.Contains(executorLog, "=== REQUEST TOOLS ===") || !strings.Contains(executorLog, `"name": "navigate"`) {
		t.Fatalf("executor tools missing from log: %s", executorLog)
	}
}

func TestAllocateNextPromptTurnPersistsDailyIndex(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"1_promp_1_planner.txt", "1_promp_2_executor.txt", "1_promp_1786589509650712686.txt"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	turn, err := allocateNextPromptTurn(directory)
	if err != nil || turn != 2 {
		t.Fatalf("unexpected next prompt turn: turn=%d err=%v", turn, err)
	}
	secondTurn, err := allocateNextPromptTurn(directory)
	if err != nil || secondTurn != 3 {
		t.Fatalf("unexpected persisted prompt turn: turn=%d err=%v", secondTurn, err)
	}
	indexContent, err := os.ReadFile(filepath.Join(directory, promptLogIndexName))
	if err != nil || string(indexContent) != "3\n" {
		t.Fatalf("unexpected prompt index: content=%q err=%v", indexContent, err)
	}
}

func TestPromptLogFileNameUsesTurnThenCall(t *testing.T) {
	if name := promptLogFileName(7, 1, "planner"); name != "7_promp_1_planner.txt" {
		t.Fatalf("unexpected planner filename: %s", name)
	}
	if name := promptLogFileName(7, 2, "executor"); name != "7_promp_2_executor.txt" {
		t.Fatalf("unexpected executor filename: %s", name)
	}
}
