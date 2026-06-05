package pillar

import (
	"strings"

	"github.com/titlis/scoreops/internal/scoring"
)

type SecurityPillar struct{}

func NewSecurityPillar() *SecurityPillar { return &SecurityPillar{} }

func (p *SecurityPillar) Slug() string { return "security" }

func (p *SecurityPillar) RuleIDs() []string {
	return []string{"SEC-001", "SEC-002", "SEC-003", "SEC-004"}
}

func (p *SecurityPillar) Evaluate(snap scoring.WorkloadSnapshot, active map[string]bool) []scoring.RuleResult {
	checks := []struct {
		id string
		fn func() scoring.RuleResult
	}{
		{"SEC-001", func() scoring.RuleResult { return checkSEC001(snap) }},
		{"SEC-002", func() scoring.RuleResult { return checkSEC002(snap) }},
		{"SEC-003", func() scoring.RuleResult { return checkSEC003(snap) }},
		{"SEC-004", func() scoring.RuleResult { return checkSEC004(snap) }},
	}

	var results []scoring.RuleResult
	for _, c := range checks {
		if active[c.id] {
			results = append(results, c.fn())
		}
	}
	return results
}

func checkSEC001(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	tag := snap.ImageTag
	passed := tag != "" && !strings.HasSuffix(tag, ":latest") && tag != "latest"
	msg := "✅ Imagem usa tag versionada"
	if !passed {
		msg = "❌ Imagem usa tag :latest"
	}
	return ruleResult("SEC-001", "No Latest Image Tag", "error", 9.0, false, passed, msg, tag)
}

func checkSEC002(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.ReadOnlyRootFS
	msg := "✅ ReadOnlyRootFilesystem habilitado"
	if !passed {
		msg = "❌ ReadOnlyRootFilesystem não habilitado — adicione securityContext.readOnlyRootFilesystem: true"
	}
	return ruleResult("SEC-002", "Read-Only Root Filesystem", "warning", 6.0, true, passed, msg, "")
}

func checkSEC003(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := !snap.AllowPrivilegeEscalation
	msg := "✅ AllowPrivilegeEscalation desabilitado"
	if !passed {
		msg = "❌ AllowPrivilegeEscalation habilitado — adicione securityContext.allowPrivilegeEscalation: false"
	}
	return ruleResult("SEC-003", "No Privilege Escalation", "error", 8.0, true, passed, msg, "")
}

func checkSEC004(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.HasDropCapabilities
	msg := "✅ Capabilities dropped configurados"
	if !passed {
		msg = "❌ Nenhuma capability foi dropped — adicione securityContext.capabilities.drop: [\"ALL\"]"
	}
	return ruleResult("SEC-004", "Drop Capabilities", "warning", 5.0, true, passed, msg, "")
}
