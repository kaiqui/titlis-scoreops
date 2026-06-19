package pillar_test

import (
	"testing"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/pillar"
)

func TestQueueOperationalPillar_Evaluate(t *testing.T) {
	p := pillar.NewQueueOperationalPillar()
	if p.Slug() != "operational" {
		t.Fatalf("expected slug 'operational', got %q", p.Slug())
	}

	allActive := map[string]bool{"QO-001": true, "QO-002": true, "QO-003": true, "QO-004": true}

	tests := []struct {
		name     string
		snap     domain.QueueSnapshot
		wantPass map[string]bool
	}{
		{
			name: "compliant non-dlq subscription",
			snap: domain.QueueSnapshot{
				ExternalID:           "projects/proj/subscriptions/payments-sub",
				DisplayName:          "payments-sub",
				HasSnapshotPolicy:    true,
				IsDLQ:                false,
				SendMessageCountRate: 1.0,
			},
			wantPass: map[string]bool{"QO-001": true, "QO-002": true, "QO-003": true, "QO-004": true},
		},
		{
			name: "bad naming convention",
			snap: domain.QueueSnapshot{
				ExternalID:  "projects/proj/subscriptions/payments",
				DisplayName: "payments",
			},
			wantPass: map[string]bool{"QO-001": false},
		},
		{
			name: "dlq named correctly",
			snap: domain.QueueSnapshot{
				ExternalID:  "projects/proj/subscriptions/payments-dlq",
				DisplayName: "payments-dlq",
				IsDLQ:       true,
			},
			wantPass: map[string]bool{"QO-001": true},
		},
		{
			name: "no snapshot policy",
			snap: domain.QueueSnapshot{
				ExternalID:        "projects/proj/subscriptions/my-sub",
				DisplayName:       "my-sub",
				HasSnapshotPolicy: false,
			},
			wantPass: map[string]bool{"QO-002": false},
		},
		{
			name: "dlq with pending messages",
			snap: domain.QueueSnapshot{
				IsDLQ:                  true,
				ExternalID:             "projects/proj/subscriptions/payments-dlq",
				DeadLetterMessageCount: 5,
				SendMessageCountRate:   0,
			},
			wantPass: map[string]bool{"QO-003": false},
		},
		{
			name: "no send throughput",
			snap: domain.QueueSnapshot{
				ExternalID:           "projects/proj/subscriptions/idle-sub",
				SendMessageCountRate: 0,
			},
			wantPass: map[string]bool{"QO-004": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := p.Evaluate(tt.snap, allActive, domain.QueueThresholds{}, domain.LabelRegistry{})
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
