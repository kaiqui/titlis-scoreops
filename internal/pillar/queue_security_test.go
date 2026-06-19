package pillar_test

import (
	"testing"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/pillar"
)

func TestQueueSecurityPillar_Evaluate(t *testing.T) {
	p := pillar.NewQueueSecurityPillar()
	if p.Slug() != "security" {
		t.Fatalf("expected slug 'security', got %q", p.Slug())
	}

	allActive := map[string]bool{"QS-001": true, "QS-002": true}
	registry  := domain.LabelRegistry{
		"env":     {"production", "staging"},
		"team":    {"platform", "data"},
		"service": {"api-gateway", "worker"},
	}

	tests := []struct {
		name     string
		snap     domain.QueueSnapshot
		registry domain.LabelRegistry
		wantPass map[string]bool
	}{
		{
			name: "all labels valid and registered",
			snap: domain.QueueSnapshot{
				Labels: map[string]string{
					"env":     "production",
					"team":    "platform",
					"service": "api-gateway",
				},
			},
			registry: registry,
			wantPass: map[string]bool{"QS-001": true, "QS-002": true},
		},
		{
			name: "missing required label",
			snap: domain.QueueSnapshot{
				Labels: map[string]string{
					"env":  "production",
					"team": "platform",
				},
			},
			registry: registry,
			wantPass: map[string]bool{"QS-001": false},
		},
		{
			name: "unregistered value",
			snap: domain.QueueSnapshot{
				Labels: map[string]string{
					"env":     "dev",
					"team":    "platform",
					"service": "api-gateway",
				},
			},
			registry: registry,
			wantPass: map[string]bool{"QS-001": false},
		},
		{
			name: "public subscription",
			snap: domain.QueueSnapshot{
				Labels: map[string]string{
					"env":        "production",
					"team":       "platform",
					"service":    "api-gateway",
					"iam_public": "true",
				},
			},
			registry: registry,
			wantPass: map[string]bool{"QS-001": true, "QS-002": false},
		},
		{
			name: "empty registry — labels present but not validated",
			snap: domain.QueueSnapshot{
				Labels: map[string]string{
					"env":     "any-value",
					"team":    "some-team",
					"service": "some-service",
				},
			},
			registry: domain.LabelRegistry{},
			wantPass: map[string]bool{"QS-001": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := p.Evaluate(tt.snap, allActive, domain.QueueThresholds{}, tt.registry)
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
