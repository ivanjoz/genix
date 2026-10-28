package agent

import (
	"os"
	"regexp"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestPrunePageToolRoundsKeepsTheLastRoundsAndOneFullSnapshot(t *testing.T) {
	snapshotResult := func(route string) string {
		return pageToolResult(map[string]any{"ok": true, "route": route}, "<main>"+route+"</main>")
	}
	round := func(index string) []*schema.Message {
		return []*schema.Message{
			schema.AssistantMessage("", []schema.ToolCall{{ID: "c" + index, Function: schema.FunctionCall{Name: "get_page"}}}),
			schema.ToolMessage(snapshotResult("/r"+index), "c"+index),
		}
	}
	messages := []*schema.Message{schema.SystemMessage("system"), schema.UserMessage("hola")}
	for _, index := range []string{"1", "2", "3"} {
		messages = append(messages, round(index)...)
	}

	pruned := prunePageToolRounds(messages, 2)
	if len(pruned) != 6 || pruned[2].ToolCalls[0].ID != "c2" {
		t.Fatalf("expected system, user and rounds 2-3, got %d messages starting at %v", len(pruned), pruned[2].ToolCalls)
	}
	if _, _, hasSnapshot := splitPageToolResult(pruned[3].Content); hasSnapshot {
		t.Fatal("an older tool result kept its page snapshot")
	}
	if _, _, hasSnapshot := splitPageToolResult(pruned[5].Content); !hasSnapshot {
		t.Fatal("the latest tool result lost its page snapshot")
	}
	if _, _, hasSnapshot := splitPageToolResult(messages[5].Content); !hasSnapshot {
		t.Fatal("pruning rewrote the caller's message instead of a copy")
	}
}

// The sales tool is offered to every user because GET.sale-summary, whose data it reads, is an
// unmapped GET: the backend's gate lets any signed-in session read it. Mapping the route to an
// access must come with the same gate on the tool (fareward's ChargeAPIAccessOnly in executionTools).
func TestSalesToolGateMatchesTheRoute(t *testing.T) {
	accessCatalog, err := os.ReadFile("../backend/access.toml")
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`backend_apis\s*=\s*"[^"]*GET\.sale-summary\b`).Match(accessCatalog) {
		t.Fatal("GET.sale-summary is mapped to an access now: gate salestools.ToolName on it before offering the tool")
	}
}

func TestCheckPointIDsAreTheOnlyResumeKeys(t *testing.T) {
	if !checkPointIDPattern.MatchString(newCheckPointID()) {
		t.Fatal("a minted checkpoint ID does not pass its own validation")
	}
	for _, forged := range []string{"../../1/2/x", "abc", "ABCDEFGH", ""} {
		if checkPointIDPattern.MatchString(forged) {
			t.Fatalf("%q was accepted as a checkpoint ID", forged)
		}
	}
}
