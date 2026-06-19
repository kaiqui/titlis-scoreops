package pillar_test

import (
	"testing"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/pillar"
)

func TestQueueObservabilityPillar_Evaluate(t *testing.T) {
	p := pillar.NewQueueObservabilityPillar()
	if p.Slug() != "observability" {
		t.Fatalf("expected slug 'observability', got %q", p.Slug())
	}

	allActive := map[string]bool{"QObs-001": true, "QObs-002": true, "QObs-003": true, "QObs-004": true}

	tests := []struct {
		name     string
		snap     domain.QueueSnapshot
		wantPass map[string]bool
	}{
		{
			name: "fully observable non-dlq",
			snap: domain.QueueSnapshot{
				HasMonitorBacklog: true,
				HasMonitorAge:     true,
				HasMonitorDLQ:     false,
				IsDLQ:             false,
				ExternalID:        "projects/proj/subscriptions/my-sub",
				TopicID:           "my-topic",
				ProjectID:         "my-project",
			},
			wantPass: map[string]bool{"QObs-001": true, "QObs-002": true, "QObs-003": true, "QObs-004": true},
		},
		{
			name: "missing backlog monitor",
			snap: domain.QueueSnapshot{
				HasMonitorBacklog: false,
				HasMonitorAge:     true,
				ExternalID:        "projects/proj/subscriptions/my-sub",
				TopicID:           "my-topic",
				ProjectID:         "my-project",
			},
			wantPass: map[string]bool{"QObs-001": false, "QObs-002": true},
		},
		{
			name: "dlq without dlq monitor",
			snap: domain.QueueSnapshot{
				IsDLQ:             true,
				HasMonitorBacklog: true,
				HasMonitorAge:     true,
				HasMonitorDLQ:     false,
				ExternalID:        "projects/proj/subscriptions/my-dlq",
				TopicID:           "my-topic",
				ProjectID:         "my-project",
			},
			wantPass: map[string]bool{"QObs-003": false},
		},
		{
			name: "dlq with dlq monitor",
			snap: domain.QueueSnapshot{
				IsDLQ:             true,
				HasMonitorBacklog: true,
				HasMonitorAge:     true,
				HasMonitorDLQ:     true,
				ExternalID:        "projects/proj/subscriptions/my-dlq",
				TopicID:           "my-topic",
				ProjectID:         "my-project",
			},
			wantPass: map[string]bool{"QObs-003": true},
		},
		{
			name: "missing correlation tags",
			snap: domain.QueueSnapshot{
				HasMonitorBacklog: true,
				HasMonitorAge:     true,
				ExternalID:        "projects/proj/subscriptions/my-sub",
				TopicID:           "",
				ProjectID:         "",
			},
			wantPass: map[string]bool{"QObs-004": false},
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
