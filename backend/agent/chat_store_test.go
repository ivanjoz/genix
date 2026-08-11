package agent

import (
	"testing"

	"app/agent/types"
)

func TestAssembleCompletedTurnsPairsRowsAndAssignsOffsets(t *testing.T) {
	rows := []types.AgentMessage{
		{Role: RoleUser, Message: "old interrupted"},
		{Role: RoleUser, Message: "¿Cómo confirmo una OC?", AttachedContent: "/logistics/purchase-orders"},
		{Role: RoleAgent, Message: "Abre la orden.", Summary: "opened order"},
		{Role: RoleUser, Message: "new interrupted"},
		{Role: RoleUser, Message: "¿Qué es un arqueo?", AttachedContent: "/finance/cash-banks"},
		{Role: RoleAgent, Message: "Es una conciliación de efectivo."},
	}

	completed := assembleCompletedTurns(rows, 5)
	if len(completed) != 2 {
		t.Fatalf("expected two completed turns, got %+v", completed)
	}
	if completed[0].Offset != -2 || completed[1].Offset != -1 {
		t.Fatalf("unexpected offsets: %+v", completed)
	}
	if completed[0].Route != "/logistics/purchase-orders" || completed[0].ActionSummary != "opened order" {
		t.Fatalf("turn metadata was not preserved: %+v", completed[0])
	}
}

func TestAssembleCompletedTurnsKeepsNewestLimit(t *testing.T) {
	rows := []types.AgentMessage{}
	for index := 1; index <= 7; index++ {
		rows = append(rows,
			types.AgentMessage{Role: RoleUser, Message: string(rune('a' + index - 1))},
			types.AgentMessage{Role: RoleAgent, Message: "reply"},
		)
	}
	completed := assembleCompletedTurns(rows, 5)
	if len(completed) != 5 || completed[0].UserMessage != "c" || completed[4].Offset != -1 {
		t.Fatalf("unexpected completed-turn limit: %+v", completed)
	}
}

func TestNextMessageTimestampIsStrictlyIncreasing(t *testing.T) {
	session := &AgentSession{}
	previous := session.nextMessageTimestamp()
	for range 10 {
		next := session.nextMessageTimestamp()
		if next <= previous {
			t.Fatalf("timestamp did not increase: previous=%d next=%d", previous, next)
		}
		previous = next
	}
}
