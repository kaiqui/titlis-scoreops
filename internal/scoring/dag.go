package scoring

import (
	"errors"
	"sort"
)

var ErrCycle = errors.New("cycle detected in DAG")

// DAG is a directed acyclic graph used to order pillar evaluation.
// AddEdge(a, b) means "a must execute before b".
type DAG struct {
	nodes map[string]struct{}
	edges map[string][]string // node → successors (nodes that depend on it)
}

func NewDAG() *DAG {
	return &DAG{
		nodes: make(map[string]struct{}),
		edges: make(map[string][]string),
	}
}

func (d *DAG) AddNode(id string) {
	d.nodes[id] = struct{}{}
	if _, ok := d.edges[id]; !ok {
		d.edges[id] = nil
	}
}

// AddEdge records that from must execute before to.
func (d *DAG) AddEdge(from, to string) {
	d.edges[from] = append(d.edges[from], to)
}

// TopologicalSort returns nodes in execution order using Kahn's algorithm.
// Returns ErrCycle if the graph contains a cycle.
func (d *DAG) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int, len(d.nodes))
	for node := range d.nodes {
		inDegree[node] = 0
	}
	for _, succs := range d.edges {
		for _, s := range succs {
			inDegree[s]++
		}
	}

	queue := make([]string, 0, len(d.nodes))
	for node := range d.nodes {
		if inDegree[node] == 0 {
			queue = append(queue, node)
		}
	}
	sort.Strings(queue)

	result := make([]string, 0, len(d.nodes))
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		succs := d.edges[node]
		sort.Strings(succs)
		for _, s := range succs {
			inDegree[s]--
			if inDegree[s] == 0 {
				queue = append(queue, s)
				sort.Strings(queue)
			}
		}
	}

	if len(result) != len(d.nodes) {
		return nil, ErrCycle
	}
	return result, nil
}
