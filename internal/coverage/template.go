package coverage

// ExpectationTemplate is a stable, parameterized coverage rule. The Code is stable; what makes the
// scorecard PERSONALIZED is which templates AppliesWhen selects for a service's discovered nature,
// plus the per-service evidence. RequiresProvider drives N/A (not "missing") when a source is off.
type ExpectationTemplate struct {
	Code               string
	Pillar             string
	Severity           string
	Weight             float64
	Remediable         bool
	RequiresCapability string // e.g. "monitor","tracing","metrics","logs"; "" = always evaluable
	AppliesWhen        func(Nature) bool
	Signal             func(Found) bool
	PassMessage        string
	FailMessage        string
	NAMessage          string
}

// --- applicability predicates (the "by nature" selection) ---

func always(Nature) bool          { return true }
func httpFacing(n Nature) bool     { return n.HTTPFacing }
func httpOrHighCrit(n Nature) bool { return n.HTTPFacing || n.Criticality == "high" }
func notScheduled(n Nature) bool   { return !n.Scheduled }
func isJava(n Nature) bool         { return n.Language == "java" }
func hasQueueDep(n Nature) bool    { return n.HasQueueDep }

// DefaultTemplates is the v1 catalog (code-defined; a tenant-scoped DB-backed catalog is a future
// layer per the D5 plan). Codes are stable.
func DefaultTemplates() []ExpectationTemplate {
	return []ExpectationTemplate{
		{
			Code: "COV-RESOURCES", Pillar: "performance", Severity: "error", Weight: 10, Remediable: true,
			AppliesWhen: always,
			Signal:      func(f Found) bool { return f.CPURequestSet && f.CPULimitSet && f.MemoryRequestSet && f.MemoryLimitSet },
			PassMessage: "✅ requests/limits de CPU e memória definidos",
			FailMessage: "❌ requests/limits de CPU/memória incompletos",
		},
		{
			Code: "COV-PROBES", Pillar: "operational", Severity: "error", Weight: 8, Remediable: true,
			AppliesWhen: always, Signal: func(f Found) bool { return f.HasProbes },
			PassMessage: "✅ liveness e readiness probes configurados",
			FailMessage: "❌ probes ausentes",
		},
		{
			Code: "COV-NETWORKPOLICY", Pillar: "security", Severity: "warning", Weight: 6, Remediable: true,
			AppliesWhen: always, Signal: func(f Found) bool { return f.HasNetworkPolicy },
			PassMessage: "✅ NetworkPolicy presente no namespace",
			FailMessage: "❌ sem NetworkPolicy",
		},
		{
			Code: "COV-HPA", Pillar: "resilience", Severity: "warning", Weight: 6, Remediable: true,
			AppliesWhen: httpFacing, Signal: func(f Found) bool { return f.HasHPA },
			PassMessage: "✅ HPA configurado",
			FailMessage: "❌ serviço http-facing sem HPA",
		},
		{
			Code: "COV-PDB", Pillar: "resilience", Severity: "warning", Weight: 5, Remediable: true,
			AppliesWhen: notScheduled, Signal: func(f Found) bool { return f.HasPDB },
			PassMessage: "✅ PodDisruptionBudget configurado",
			FailMessage: "❌ sem PodDisruptionBudget",
		},
		{
			Code: "COV-SLO", Pillar: "observability", Severity: "warning", Weight: 8, Remediable: false,
			AppliesWhen: httpOrHighCrit, Signal: func(f Found) bool { return f.HasSLO },
			PassMessage: "✅ SLO configurado",
			FailMessage: "❌ serviço http-facing/crítico sem SLO",
		},
		{
			Code: "COV-MONITOR", Pillar: "observability", Severity: "warning", Weight: 7, Remediable: false,
			RequiresCapability: "monitor", AppliesWhen: always, Signal: func(f Found) bool { return f.HasMonitor },
			PassMessage: "✅ monitor(es) cobrindo o serviço",
			FailMessage: "❌ nenhum monitor cobrindo o serviço",
			NAMessage:   "⏭ cobertura de monitores não avaliável (fonte de monitor não conectada)",
		},
		{
			Code: "COV-TRACING", Pillar: "observability", Severity: "warning", Weight: 6, Remediable: false,
			RequiresCapability: "tracing", AppliesWhen: httpFacing, Signal: func(f Found) bool { return f.HasTracing },
			PassMessage: "✅ tracing distribuído presente",
			FailMessage: "❌ serviço http-facing sem tracing distribuído",
			NAMessage:   "⏭ tracing não avaliável (fonte de tracing não conectada)",
		},
		{
			Code: "COV-HTTP-METRICS", Pillar: "observability", Severity: "info", Weight: 5, Remediable: false,
			RequiresCapability: "metrics", AppliesWhen: httpFacing,
			Signal:      func(f Found) bool { return f.hasMetricCategory("http") },
			PassMessage: "✅ métricas HTTP (req/err/latency) presentes",
			FailMessage: "❌ sem métricas HTTP",
			NAMessage:   "⏭ métricas HTTP não avaliáveis (fonte de métricas não conectada)",
		},
		{
			Code: "COV-JVM-METRICS", Pillar: "observability", Severity: "info", Weight: 4, Remediable: false,
			RequiresCapability: "metrics", AppliesWhen: isJava,
			Signal:      func(f Found) bool { return f.hasMetricCategory("jvm") },
			PassMessage: "✅ métricas JVM presentes",
			FailMessage: "❌ serviço Java sem métricas JVM",
			NAMessage:   "⏭ métricas JVM não avaliáveis (fonte de métricas não conectada)",
		},
		{
			Code: "COV-QUEUE-MONITOR", Pillar: "observability", Severity: "warning", Weight: 6, Remediable: false,
			RequiresCapability: "monitor", AppliesWhen: hasQueueDep, Signal: func(f Found) bool { return f.HasMonitor },
			PassMessage: "✅ filas do serviço monitoradas",
			FailMessage: "❌ dependência de fila sem monitor",
			NAMessage:   "⏭ monitor de fila não avaliável (fonte de monitor não conectada)",
		},
		{
			Code: "COV-LOGS", Pillar: "observability", Severity: "info", Weight: 4, Remediable: false,
			RequiresCapability: "logs", AppliesWhen: always, Signal: func(f Found) bool { return f.HasLogs },
			PassMessage: "✅ logs coletados",
			FailMessage: "❌ sem logs coletados",
			NAMessage:   "⏭ coleta de logs não avaliável (fonte de logs não conectada)",
		},
	}
}
