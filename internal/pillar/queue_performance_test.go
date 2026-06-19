package pillar_test

import (
	"testing"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/pillar"
)

func TestQueuePerformancePillar_Evaluate(t *testing.T) {
	p := pillar.NewQueuePerformancePillar()
	if p.Slug() != "performance" {
		t.Fatalf("expected slug 'performance', got %q", p.Slug())
	}

	allActive := map[string]bool{"QP-001": true, "QP-002": true, "QP-003": true}
	thresholds := domain.QueueThresholds{AgeWarningSec: 60}

	tests := []struct {
		name     string
		snap     domain.QueueSnapshot
		wantPass map[string]bool
	}{
		{
			name: "healthy queue",
			snap: domain.QueueSnapshot{
				PullMessageCountRate: 1.5,
				SendMessageCountRate: 2.0,
				AckMessageCountRate:  1.9,
				OldestUnackedAgeSec:  10,
			},
			wantPass: map[string]bool{"QP-001": true, "QP-002": true, "QP-003": true},
		},
		{
			name: "no consumer",
			snap: domain.QueueSnapshot{
				PullMessageCountRate: 0,
				SendMessageCountRate: 2.0,
				AckMessageCountRate:  0,
				OldestUnackedAgeSec:  10,
			},
			wantPass: map[string]bool{"QP-001": false, "QP-002": false},
		},
		{
			name: "low ack rate",
			snap: domain.QueueSnapshot{
				PullMessageCountRate: 1.0,
				SendMessageCountRate: 2.0,
				AckMessageCountRate:  1.0, // 50% — below 80%
				OldestUnackedAgeSec:  10,
			},
			wantPass: map[string]bool{"QP-002": false},
		},
		{
			name: "zero send rate skips QP-002",
			snap: domain.QueueSnapshot{
				PullMessageCountRate: 1.0,
				SendMessageCountRate: 0,
				AckMessageCountRate:  0,
			},
			wantPass: map[string]bool{"QP-002": true},
		},
		{
			name: "age exceeds warning threshold",
			snap: domain.QueueSnapshot{
				PullMessageCountRate: 1.0,
				SendMessageCountRate: 1.0,
				AckMessageCountRate:  1.0,
				OldestUnackedAgeSec:  120,
			},
			wantPass: map[string]bool{"QP-003": false},
		},
		{
			name: "zero threshold skips QP-003",
			snap: domain.QueueSnapshot{
				PullMessageCountRate: 1.0,
				SendMessageCountRate: 1.0,
				AckMessageCountRate:  1.0,
				OldestUnackedAgeSec:  9999,
			},
			wantPass: map[string]bool{"QP-003": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			thr := thresholds
			if tt.name == "zero threshold skips QP-003" {
				thr = domain.QueueThresholds{AgeWarningSec: 0}
			}
			results := p.Evaluate(tt.snap, allActive, thr, domain.LabelRegistry{})
			byID := map[string]bool{}
			for _, r := range results {
				byID[r.RuleID] = r.Passed
			}
			for id, want := range tt.wantPass {
				got, ok := byID[id]
				if !ok {
					t.Errorf("rule %s missing from results", id)
					continue
				}
				if got != want {
					t.Errorf("rule %s: want passed=%v, got passed=%v", id, want, got)
				}
			}
		})
	}
}
