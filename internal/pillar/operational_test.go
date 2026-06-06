package pillar

import (
	"testing"

	"github.com/titlis/scoreops/internal/scoring"
)

func TestOPS001_Pass(t *testing.T) {
	snap := fullSnap()
	result := checkOPS001(snap)
	if !result.Passed {
		t.Errorf("OPS-001 esperava pass, got fail: %s", result.Message)
	}
	if result.RuleID != "OPS-001" {
		t.Errorf("rule_id incorreto: %s", result.RuleID)
	}
}

func TestOPS001_MissingLabel(t *testing.T) {
	snap := fullSnap()
	delete(snap.Labels, "tags.datadoghq.com/service")
	result := checkOPS001(snap)
	if result.Passed {
		t.Error("OPS-001 esperava fail quando label de service está ausente")
	}
}

func TestOPS001_MissingAdmissionLabel(t *testing.T) {
	snap := fullSnap()
	snap.Labels["admission.datadoghq.com/enabled"] = "false"
	result := checkOPS001(snap)
	if result.Passed {
		t.Error("OPS-001 esperava fail quando admission label não é 'true'")
	}
}

func TestOPS001_NilLabels(t *testing.T) {
	snap := fullSnap()
	snap.Labels = nil
	result := checkOPS001(snap)
	if result.Passed {
		t.Error("OPS-001 esperava fail com labels nil")
	}
}

func TestOPS002_Pass(t *testing.T) {
	snap := fullSnap()
	snap.BackstageComponent = "my-service"
	result := checkOPS002(snap)
	if !result.Passed {
		t.Errorf("OPS-002 esperava pass quando BackstageComponent está preenchido, got: %s", result.Message)
	}
	if result.RuleID != "OPS-002" {
		t.Errorf("rule_id incorreto: %s", result.RuleID)
	}
	if result.ActualValue != "my-service" {
		t.Errorf("actual_value esperava 'my-service', got: %s", result.ActualValue)
	}
}

func TestOPS002_Fail_EmptyComponent(t *testing.T) {
	snap := fullSnap()
	snap.BackstageComponent = ""
	result := checkOPS002(snap)
	if result.Passed {
		t.Error("OPS-002 esperava fail quando BackstageComponent está vazio")
	}
	if result.Severity != "error" {
		t.Errorf("severidade esperada 'error', got: %s", result.Severity)
	}
}

func TestOPS002_Fail_SnapWithoutField(t *testing.T) {
	snap := scoring.WorkloadSnapshot{
		UID:        "uid-x",
		Name:       "svc",
		Namespace:  "ns",
		Cluster:    "cl",
		TenantID:   1,
		EngineSlug: "kubernetes",
	}
	result := checkOPS002(snap)
	if result.Passed {
		t.Error("OPS-002 esperava fail em snapshot sem BackstageComponent")
	}
}

func TestOperationalPillar_EvaluateBothRules(t *testing.T) {
	p := NewOperationalPillar()
	snap := fullSnap()
	snap.BackstageComponent = "my-service"
	active := map[string]bool{"OPS-001": true, "OPS-002": true}

	results := p.Evaluate(snap, active)
	if len(results) != 2 {
		t.Fatalf("esperava 2 resultados, got %d", len(results))
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("regra %s falhou: %s", r.RuleID, r.Message)
		}
	}
}

func TestOperationalPillar_EvaluateOnlyOPS001(t *testing.T) {
	p := NewOperationalPillar()
	snap := fullSnap()
	active := map[string]bool{"OPS-001": true}

	results := p.Evaluate(snap, active)
	if len(results) != 1 {
		t.Fatalf("esperava 1 resultado, got %d", len(results))
	}
	if results[0].RuleID != "OPS-001" {
		t.Errorf("esperava OPS-001, got %s", results[0].RuleID)
	}
}

func TestOperationalPillar_EvaluateOnlyOPS002(t *testing.T) {
	p := NewOperationalPillar()
	snap := fullSnap()
	snap.BackstageComponent = "catalog-entry"
	active := map[string]bool{"OPS-002": true}

	results := p.Evaluate(snap, active)
	if len(results) != 1 {
		t.Fatalf("esperava 1 resultado, got %d", len(results))
	}
	if results[0].RuleID != "OPS-002" {
		t.Errorf("esperava OPS-002, got %s", results[0].RuleID)
	}
}

func TestOperationalPillar_EvaluateNoneActive(t *testing.T) {
	p := NewOperationalPillar()
	results := p.Evaluate(fullSnap(), map[string]bool{})
	if len(results) != 0 {
		t.Errorf("esperava 0 resultados com nenhuma regra ativa, got %d", len(results))
	}
}

func TestOperationalPillar_RuleIDs(t *testing.T) {
	p := NewOperationalPillar()
	ids := p.RuleIDs()
	if len(ids) != 2 {
		t.Fatalf("esperava 2 rule IDs, got %d", len(ids))
	}
	want := map[string]bool{"OPS-001": true, "OPS-002": true}
	for _, id := range ids {
		if !want[id] {
			t.Errorf("rule ID inesperado: %s", id)
		}
	}
}
