package scoring

import (
	"testing"
)

func TestDAG_TopologicalSort_NoDeps(t *testing.T) {
	d := NewDAG()
	d.AddNode("resilience")
	d.AddNode("security")
	d.AddNode("performance")
	d.AddNode("operational")

	order, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(order))
	}
	// With no edges, nodes come out sorted alphabetically (deterministic)
	expected := []string{"operational", "performance", "resilience", "security"}
	for i, got := range order {
		if got != expected[i] {
			t.Errorf("position %d: got %q, want %q", i, got, expected[i])
		}
	}
}

func TestDAG_TopologicalSort_WithDeps(t *testing.T) {
	d := NewDAG()
	d.AddNode("a")
	d.AddNode("b")
	d.AddNode("c")
	d.AddEdge("a", "b") // a before b
	d.AddEdge("b", "c") // b before c

	order, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pos := func(s string) int {
		for i, v := range order {
			if v == s {
				return i
			}
		}
		return -1
	}
	if pos("a") >= pos("b") {
		t.Errorf("expected a before b, got %v", order)
	}
	if pos("b") >= pos("c") {
		t.Errorf("expected b before c, got %v", order)
	}
}

func TestDAG_TopologicalSort_Cycle(t *testing.T) {
	d := NewDAG()
	d.AddNode("a")
	d.AddNode("b")
	d.AddEdge("a", "b")
	d.AddEdge("b", "a") // cycle

	_, err := d.TopologicalSort()
	if err != ErrCycle {
		t.Fatalf("expected ErrCycle, got %v", err)
	}
}
