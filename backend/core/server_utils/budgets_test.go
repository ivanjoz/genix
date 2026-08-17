package server_utils

import (
	"bytes"
	"testing"
)

func TestBudgetMutationPayloadMatchesTheWireOffsets(t *testing.T) {
	payload, err := encodeBudgetMutation(0x123456, BudgetIncreaseCurrent, 300, 25)
	if err != nil {
		t.Fatalf("encodeBudgetMutation refused valid values: %v", err)
	}
	want := []byte{
		0x12, 0x34, 0x56,
		0x03,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x2C,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x19,
	}
	if !bytes.Equal(payload, want) {
		t.Fatalf("budget payload = % X; want % X", payload, want)
	}
}

func TestBudgetMutationRejectsUnknownOperation(t *testing.T) {
	if _, err := encodeBudgetMutation(1, BudgetOperation(4), 1, 1); err == nil {
		t.Fatal("unknown budget operation was accepted")
	}
}
