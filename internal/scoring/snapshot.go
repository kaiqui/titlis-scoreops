package scoring

import "encoding/json"

// WorkloadSnapshot holds normalized metrics extracted by the operator from a single Deployment.
// All fields are primitives — no K8s types. TenantID is injected by titlis-api before forwarding.
//
// HPA stabilization fields use -1 as a sentinel for "not configured":
// -1 = HPA behavior not set, 0 = configured to 0 (immediate), positive = configured window in seconds.
type WorkloadSnapshot struct {
	// Identity
	UID         string            `json:"uid"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Cluster     string            `json:"cluster"`
	Kind        string            `json:"kind"`
	Criticality string            `json:"criticality"` // "standard" | "high"
	Labels      map[string]string `json:"labels"`
	TenantID      int64             `json:"tenant_id"`      // injected by titlis-api, not set by operator
	EngineSlug    string            `json:"engine_slug"`    // "kubernetes"
	ClusterTags   []string          `json:"cluster_tags"`   // injected by titlis-api from resource_tags
	NamespaceTags []string          `json:"namespace_tags"` // injected by titlis-api from resource_tags

	// Container
	HasLivenessProbe         bool    `json:"has_liveness_probe"`
	HasReadinessProbe        bool    `json:"has_readiness_probe"`
	CPURequestSet            bool    `json:"cpu_request_set"`
	CPULimitSet              bool    `json:"cpu_limit_set"`
	MemoryRequestSet         bool    `json:"memory_request_set"`
	MemoryLimitSet           bool    `json:"memory_limit_set"`
	CPULimitRatio            float64 `json:"cpu_limit_ratio"`   // limit/request; 0 if not both set
	ImageTag                 string  `json:"image_tag"`          // "latest", "v1.2.3", "sha256:..."
	ReadOnlyRootFS           bool    `json:"read_only_root_fs"`
	RunAsNonRoot             bool    `json:"run_as_non_root"`
	AllowPrivilegeEscalation bool    `json:"allow_privilege_escalation"`
	HasDropCapabilities      bool    `json:"has_drop_capabilities"`
	HasPodSecurityContext    bool    `json:"has_pod_security_context"`

	// Deployment
	Replicas                  int32  `json:"replicas"`
	Strategy                  string `json:"strategy"` // "RollingUpdate" | "Recreate"
	TerminationGracePeriodSec int64  `json:"termination_grace_period_sec"` // 0 = not explicitly set

	HasNetworkPolicy bool `json:"has_network_policy"`

	// HPA
	HasHPA                       bool `json:"has_hpa"`
	HPAHasMetrics                bool `json:"hpa_has_metrics"`
	HPAMinReplicas               int  `json:"hpa_min_replicas"`
	HPAMaxReplicas               int  `json:"hpa_max_replicas"`
	HPACPUTargetPercent          int  `json:"hpa_cpu_target_percent"`  // 0 = not set
	HPAScaleUpStabilizationSec   int  `json:"hpa_scale_up_stabilization_sec"`   // -1 = not configured
	HPAScaleDownStabilizationSec int  `json:"hpa_scale_down_stabilization_sec"` // -1 = not configured
	HPAHasBehaviorPolicies       bool `json:"hpa_has_behavior_policies"`

	// External context (injected by titlis-api from cluster metadata — additive fields)
	HasDatadog  bool   `json:"has_datadog,omitempty"`
	Environment string `json:"environment,omitempty"` // dev | hml | prd | unknown

	// SLO presence (injected by titlis-api from slo_configs table before forwarding to scoreops)
	HasSLO     bool `json:"has_slo,omitempty"`
	SLOHealthy bool `json:"slo_healthy,omitempty"`
}

func (s *WorkloadSnapshot) FromJSON(data []byte) error {
	return json.Unmarshal(data, s)
}
