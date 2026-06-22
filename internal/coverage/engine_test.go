package coverage

import "testing"

func byCode(findings []CoverageFinding) map[string]Outcome {
	m := make(map[string]Outcome, len(findings))
	for _, f := range findings {
		m[f.Code] = f.Outcome
	}
	return m
}

func TestEvaluate_PersonalizedForJavaHTTPService(t *testing.T) {
	snap := CoverageSnapshot{
		TenantID: 1, WorkloadUID: "u1", ServiceName: "orders",
		Nature: Nature{Language: "java", HTTPFacing: true, Criticality: "high"},
		Found: Found{
			HasSLO: true, HasMonitor: true, HasTracing: true, HasLogs: true,
			MetricCategories: []string{"http", "jvm"},
			CPURequestSet:    true, CPULimitSet: true, MemoryRequestSet: true, MemoryLimitSet: true,
			HasProbes: true, HasHPA: true, HasPDB: true, HasNetworkPolicy: true,
		},
		Capabilities: []string{"monitor", "tracing", "metrics", "logs"},
	}

	res := NewEngine(nil).Evaluate(snap)
	m := byCode(res.Findings)

	// A Java http-facing service must get the Java/HTTP/tracing/SLO expectations — all passing here.
	for _, c := range []string{
		"COV-JVM-METRICS", "COV-TRACING", "COV-HTTP-METRICS", "COV-SLO", "COV-HPA",
		"COV-RESOURCES", "COV-PROBES", "COV-MONITOR", "COV-LOGS", "COV-NETWORKPOLICY", "COV-PDB",
	} {
		if m[c] != OutcomePass {
			t.Errorf("expected %s = pass, got %q", c, m[c])
		}
	}
	if res.TrustScore != 100 {
		t.Errorf("trust = %v, want 100", res.TrustScore)
	}
	if res.Maturity != 5 {
		t.Errorf("maturity = %d, want 5 (tudo passa)", res.Maturity)
	}
	if res.EngineSlug != "coverage" {
		t.Errorf("engineSlug = %q", res.EngineSlug)
	}
}

func TestEvaluate_MaturityIsWeakestLink(t *testing.T) {
	snap := CoverageSnapshot{
		TenantID: 1, WorkloadUID: "u5",
		Nature: Nature{}, // não-http, não-java, não-scheduled
		Found: Found{ // probes+netpol ok (→ dim 5); resources e PDB faltando (→ dim 1)
			HasProbes: true, HasNetworkPolicy: true,
		},
		Capabilities: nil,
	}
	res := NewEngine(nil).Evaluate(snap)
	if res.Maturity != 1 {
		t.Errorf("maturity = %d, want 1 (elo mais fraco: resources/pdb falhando)", res.Maturity)
	}
}

func TestEvaluate_GoCronJobGetsDifferentScorecard(t *testing.T) {
	snap := CoverageSnapshot{
		TenantID: 1, WorkloadUID: "u2", ServiceName: "cleanup",
		Nature: Nature{Language: "go", Scheduled: true},
		Found: Found{
			CPURequestSet: true, CPULimitSet: true, MemoryRequestSet: true, MemoryLimitSet: true,
			HasProbes: true, HasNetworkPolicy: true, HasLogs: true, HasMonitor: true,
		},
		Capabilities: []string{"monitor", "logs"},
	}

	res := NewEngine(nil).Evaluate(snap)
	m := byCode(res.Findings)

	// These do not apply to a non-http, scheduled, non-Java workload → must NOT be emitted.
	for _, c := range []string{"COV-JVM-METRICS", "COV-TRACING", "COV-HTTP-METRICS", "COV-HPA", "COV-PDB", "COV-SLO"} {
		if _, ok := m[c]; ok {
			t.Errorf("%s should not apply to a Go cronjob", c)
		}
	}
	// These apply to everything.
	for _, c := range []string{"COV-RESOURCES", "COV-PROBES", "COV-NETWORKPOLICY", "COV-MONITOR", "COV-LOGS"} {
		if _, ok := m[c]; !ok {
			t.Errorf("%s should apply", c)
		}
	}
}

func TestEvaluate_MissingCapabilitiesMarkNA(t *testing.T) {
	snap := CoverageSnapshot{
		TenantID: 1, WorkloadUID: "u3",
		Nature: Nature{Language: "java", HTTPFacing: true},
		Found: Found{
			CPURequestSet: true, CPULimitSet: true, MemoryRequestSet: true, MemoryLimitSet: true,
			HasProbes: true, HasNetworkPolicy: true,
		},
		Capabilities: nil, // nenhuma capacidade de observabilidade mensurável
	}

	res := NewEngine(nil).Evaluate(snap)
	m := byCode(res.Findings)

	// Capability-dependent expectations are N/A, never "missing" — no false red.
	for _, c := range []string{"COV-MONITOR", "COV-TRACING", "COV-HTTP-METRICS", "COV-JVM-METRICS", "COV-LOGS"} {
		if m[c] != OutcomeNA {
			t.Errorf("expected %s = na (no capability), got %q", c, m[c])
		}
	}
	// K8s-derived expectations are still evaluated.
	if m["COV-RESOURCES"] != OutcomePass {
		t.Errorf("COV-RESOURCES = %q, want pass", m["COV-RESOURCES"])
	}
	if res.TrustScore <= 0 || res.TrustScore > 100 {
		t.Errorf("trust out of range: %v", res.TrustScore)
	}
}

func TestEvaluate_PartialCapabilityLightsUpMonitorOnly(t *testing.T) {
	snap := CoverageSnapshot{
		TenantID: 1, WorkloadUID: "u4",
		Nature: Nature{HTTPFacing: true},
		Found: Found{
			CPURequestSet: true, CPULimitSet: true, MemoryRequestSet: true, MemoryLimitSet: true,
			HasProbes: true, HasNetworkPolicy: true, HasMonitor: true,
		},
		Capabilities: []string{"monitor"}, // só monitor é mensurável (edges dd_monitor→dd_service)
	}

	m := byCode(NewEngine(nil).Evaluate(snap).Findings)
	if m["COV-MONITOR"] != OutcomePass {
		t.Errorf("COV-MONITOR should be evaluated (pass), got %q", m["COV-MONITOR"])
	}
	if m["COV-TRACING"] != OutcomeNA {
		t.Errorf("COV-TRACING should be na (no tracing capability), got %q", m["COV-TRACING"])
	}
}

func TestEvaluate_TrustScoreIsWeighted(t *testing.T) {
	tpls := []ExpectationTemplate{
		{Code: "A", Pillar: "x", Weight: 10, AppliesWhen: always, Signal: func(Found) bool { return true }},
		{Code: "B", Pillar: "x", Weight: 30, AppliesWhen: always, Signal: func(Found) bool { return false }},
	}
	res := NewEngine(tpls).Evaluate(CoverageSnapshot{TenantID: 1, WorkloadUID: "u"})
	// passed weight 10 / total weight 40 = 25%
	if res.TrustScore != 25 {
		t.Errorf("trust = %v, want 25", res.TrustScore)
	}
}
