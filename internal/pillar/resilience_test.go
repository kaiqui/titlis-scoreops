package pillar

import (
	"testing"

	"github.com/titlis/scoreops/internal/scoring"
)

func fullSnap() scoring.WorkloadSnapshot {
	return scoring.WorkloadSnapshot{
		UID:                          "uid-1",
		Name:                         "my-app",
		Namespace:                    "production",
		Cluster:                      "prod-cluster",
		Kind:                         "Deployment",
		Criticality:                  "standard",
		EngineSlug:                   "kubernetes",
		TenantID:                     1,
		HasLivenessProbe:             true,
		HasReadinessProbe:            true,
		CPURequestSet:                true,
		CPULimitSet:                  true,
		MemoryRequestSet:             true,
		MemoryLimitSet:               true,
		CPULimitRatio:                2.0,
		ImageTag:                     "v1.2.3",
		ReadOnlyRootFS:               true,
		RunAsNonRoot:                 true,
		AllowPrivilegeEscalation:     false,
		HasDropCapabilities:          true,
		HasPodSecurityContext:        true,
		Replicas:                     3,
		Strategy:                     "RollingUpdate",
		TerminationGracePeriodSec:    30,
		HasNetworkPolicy:             true,
		HasHPA:                       true,
		HPAHasMetrics:                true,
		HPAMinReplicas:               2,
		HPACPUTargetPercent:          60,
		HPAScaleUpStabilizationSec:   0,
		HPAScaleDownStabilizationSec: 300,
		HPAHasBehaviorPolicies:       true,
		Labels: map[string]string{
			"tags.datadoghq.com/env":               "prod",
			"tags.datadoghq.com/service":            "my-app",
			"tags.datadoghq.com/version":            "1.2.3",
			"admission.datadoghq.com/enabled":       "true",
		},
	}
}

func activeAll() map[string]bool {
	return map[string]bool{
		"RES-001": true, "RES-002": true, "RES-003": true, "RES-004": true,
		"RES-005": true, "RES-006": true, "RES-007": true, "RES-008": true,
		"RES-009": true, "RES-010": true, "RES-011": true, "RES-012": true,
		"RES-013": true, "RES-014": true, "RES-016": true, "RES-017": true,
		"RES-018": true, "RES-019": true,
	}
}

func TestResiliencePillar_FullCompliantSnap(t *testing.T) {
	p := NewResiliencePillar()
	// Use high criticality so RES-017/018/019 are included
	snap := fullSnap()
	snap.Criticality = "high"

	results := p.Evaluate(snap, activeAll())
	for _, r := range results {
		if !r.Passed {
			t.Errorf("rule %s should pass on full compliant snap, got msg: %s", r.RuleID, r.Message)
		}
	}
}

func TestResiliencePillar_HighOnlyRulesSkippedForStandard(t *testing.T) {
	p := NewResiliencePillar()
	snap := fullSnap()
	snap.Criticality = "standard"

	results := p.Evaluate(snap, activeAll())
	for _, r := range results {
		if r.RuleID == "RES-017" || r.RuleID == "RES-018" || r.RuleID == "RES-019" {
			t.Errorf("rule %s should not appear for standard criticality", r.RuleID)
		}
	}
}

func TestRES001_LivenessProbe(t *testing.T) {
	snap := fullSnap()
	snap.HasLivenessProbe = false
	r := checkRES001(snap)
	if r.Passed {
		t.Error("expected RES-001 to fail when liveness probe is absent")
	}
}

func TestRES007_HPAExists(t *testing.T) {
	snap := fullSnap()
	snap.HasHPA = false
	r := checkRES007(snap)
	if r.Passed {
		t.Error("expected RES-007 to fail when HPA is absent")
	}
}

func TestRES009_GracePeriodZeroFails(t *testing.T) {
	snap := fullSnap()
	snap.TerminationGracePeriodSec = 0 // not configured
	r := checkRES009(snap)
	if r.Passed {
		t.Error("expected RES-009 to fail when grace period is 0 (not configured)")
	}
}

func TestRES013_SingleReplicaFails(t *testing.T) {
	snap := fullSnap()
	snap.Replicas = 1
	r := checkRES013(snap)
	if r.Passed {
		t.Errorf("expected RES-013 to fail with 1 replica, got msg: %s", r.Message)
	}
}

func TestRES016_NoHPAFails(t *testing.T) {
	snap := fullSnap()
	snap.HasHPA = false
	r := checkRES016(snap)
	if r.Passed {
		t.Error("expected RES-016 to fail when HPA is absent")
	}
}

func TestRES017_SlowScaleUpFails(t *testing.T) {
	snap := fullSnap()
	snap.HPAScaleUpStabilizationSec = 60 // too high → bad
	r := checkRES017(snap)
	if r.Passed {
		t.Error("expected RES-017 to fail when scaleUp stabilization > 0")
	}
}

func TestRES017_NotConfiguredFails(t *testing.T) {
	snap := fullSnap()
	snap.HPAScaleUpStabilizationSec = -1 // sentinel: not configured
	r := checkRES017(snap)
	if r.Passed {
		t.Error("expected RES-017 to fail when scaleUp stabilization is not configured")
	}
}

func TestRES018_LowScaleDownFails(t *testing.T) {
	snap := fullSnap()
	snap.HPAScaleDownStabilizationSec = 60 // less than 300 → bad
	r := checkRES018(snap)
	if r.Passed {
		t.Errorf("expected RES-018 to fail with scaleDown=%d", snap.HPAScaleDownStabilizationSec)
	}
}

func TestRES010_AllowRootFails(t *testing.T) {
	snap := fullSnap()
	snap.RunAsNonRoot = false
	r := checkRES010(snap)
	if r.Passed {
		t.Error("expected RES-010 to fail when RunAsNonRoot is false")
	}
}

func TestResiliencePillar_InactiveRuleSkipped(t *testing.T) {
	p := NewResiliencePillar()
	snap := fullSnap()
	active := activeAll()
	delete(active, "RES-001") // disable liveness probe rule

	results := p.Evaluate(snap, active)
	for _, r := range results {
		if r.RuleID == "RES-001" {
			t.Error("RES-001 should be skipped when inactive")
		}
	}
}
