package pillar_test

import (
	"testing"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/pillar"
)

func TestQueueResiliencePillar_Evaluate(t *testing.T) {
	p := pillar.NewQueueResiliencePillar()
	if p.Slug() != "resilience" {
		t.Fatalf("expected slug 'resilience', got %q", p.Slug())
	}

	allActive := map[string]bool{"QR-001": true, "QR-002": true, "QR-003": true, "QR-004": true, "QR-005": true}

	thresholds := domain.QueueThresholds{
		BacklogWarning:  100,
		BacklogCritical: 500,
		AgeWarningSec:   60,
		AgeCriticalSec:  300,
	}

	tests := []struct {
		name     string
		snap     domain.QueueSnapshot
		wantPass map[string]bool
	}{
		{
			name: "healthy non-dlq subscription",
			snap: domain.QueueSnapshot{
				NumUndeliveredMessages:      5,
				OldestUnackedAgeSec:         10,
				HasDLQConfigured:            true,
				IsDLQ:                       false,
				MessageRetentionDurationSec: 604800,
			},
			wantPass: map[string]bool{"QR-001": true, "QR-002": true, "QR-003": true, "QR-004": true, "QR-005": true},
		},
		{
			name: "backlog at warning",
			snap: domain.QueueSnapshot{
				NumUndeliveredMessages: 150,
				OldestUnackedAgeSec:    10,
				HasDLQConfigured:       true,
			},
			wantPass: map[string]bool{"QR-001": false, "QR-002": true, "QR-003": true, "QR-004": true},
		},
		{
			name: "backlog at critical",
			snap: domain.QueueSnapshot{
				NumUndeliveredMessages: 600,
				OldestUnackedAgeSec:    10,
				HasDLQConfigured:       true,
			},
			wantPass: map[string]bool{"QR-001": false, "QR-002": true, "QR-003": true, "QR-004": true},
		},
		{
			name: "dlq saturated",
			snap: domain.QueueSnapshot{
				IsDLQ:                  true,
				NumUndeliveredMessages: 50,
			},
			wantPass: map[string]bool{"QR-003": true, "QR-004": false},
		},
		{
			name: "dlq healthy",
			snap: domain.QueueSnapshot{
				IsDLQ:                  true,
				NumUndeliveredMessages: 0,
			},
			wantPass: map[string]bool{"QR-003": true, "QR-004": true},
		},
		{
			name: "no dlq configured",
			snap: domain.QueueSnapshot{
				IsDLQ:            false,
				HasDLQConfigured: false,
			},
			wantPass: map[string]bool{"QR-003": false},
		},
		{
			name: "short retention",
			snap: domain.QueueSnapshot{
				MessageRetentionDurationSec: 86400,
			},
			wantPass: map[string]bool{"QR-005": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := p.Evaluate(tt.snap, allActive, thresholds, domain.LabelRegistry{})
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
