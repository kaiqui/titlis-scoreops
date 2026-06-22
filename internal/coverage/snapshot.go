// Package coverage is the Discovery Engine's downstream scoring (D5): it turns a per-service
// CoverageSnapshot (assembled by titlis-api from the asset graph) into a PERSONALIZED scorecard.
//
// Findings are 100% deterministic and rule-based — no AI. The personalization comes from selecting
// and instantiating stable expectation templates by the service's discovered nature (see template.go).
// AI (titlis-ai) only narrates the result later; it never generates findings.
package coverage

// Nature is what the service IS, derived from the discovered graph by titlis-api.
type Nature struct {
	Language    string `json:"language"`
	HTTPFacing  bool   `json:"httpFacing"`
	Stateful    bool   `json:"stateful"`
	Scheduled   bool   `json:"scheduled"`
	Criticality string `json:"criticality"` // "standard" | "high"
	HasQueueDep bool   `json:"hasQueueDep"`
}

// Found is what was actually discovered around the service.
type Found struct {
	HasSLO           bool     `json:"hasSlo"`
	SLOHealthy       bool     `json:"sloHealthy"`
	HasMonitor       bool     `json:"hasMonitor"`
	MonitorCount     int      `json:"monitorCount"`
	HasTracing       bool     `json:"hasTracing"`
	HasLogs          bool     `json:"hasLogs"`
	MetricCategories []string `json:"metricCategories"`
	CPURequestSet    bool     `json:"cpuRequestSet"`
	CPULimitSet      bool     `json:"cpuLimitSet"`
	MemoryRequestSet bool     `json:"memoryRequestSet"`
	MemoryLimitSet   bool     `json:"memoryLimitSet"`
	HasProbes        bool     `json:"hasProbes"`
	HasHPA           bool     `json:"hasHpa"`
	HasPDB           bool     `json:"hasPdb"`
	HasNetworkPolicy bool     `json:"hasNetworkPolicy"`
}

// CoverageSnapshot is the flat per-service input to the engine. titlis-api builds it from the graph
// (CTE over discovered_asset/asset_relation); scoreops never touches the graph itself.
//
// Capabilities lists the observability signals that are actually MEASURABLE for this service right
// now (e.g. "monitor", "slo", "tracing", "metrics", "logs"). A template whose
// RequiresCapability is not present becomes N/A — never a false "missing". This is per-capability
// honesty, not a single global "datadog connected" flag.
type CoverageSnapshot struct {
	TenantID     int64    `json:"tenantId"`
	WorkloadUID  string   `json:"workloadUid"`
	ServiceName  string   `json:"serviceName"`
	Namespace    string   `json:"namespace"`
	Cluster      string   `json:"cluster"`
	Nature       Nature   `json:"nature"`
	Found        Found    `json:"found"`
	Capabilities []string `json:"capabilities"`
}

func (s *CoverageSnapshot) hasCapability(name string) bool {
	if name == "" {
		return true
	}
	for _, c := range s.Capabilities {
		if c == name {
			return true
		}
	}
	return false
}

func (f Found) hasMetricCategory(cat string) bool {
	for _, c := range f.MetricCategories {
		if c == cat {
			return true
		}
	}
	return false
}
