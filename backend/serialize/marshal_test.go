package serialize

import (
	"testing"
)

type Node struct {
	Value int
	Next  *Node
}

func TestCycleDetection(t *testing.T) {
	n1 := &Node{Value: 1}
	n2 := &Node{Value: 2}
	n1.Next = n2
	n2.Next = n1 // Cycle!

	bytes, err := Marshal(n1)
	if err != nil {
		t.Fatalf("Marshal failed with cycle: %v", err)
	}

	if len(bytes) == 0 {
		t.Fatalf("Marshal returned empty bytes")
	}
	
	t.Logf("Marshaled cyclic structure successfully: %s", string(bytes))
}
