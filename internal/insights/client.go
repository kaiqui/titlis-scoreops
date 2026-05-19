package insights

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type HpaRecommendation struct {
	WorkloadUID  string  `json:"workload_uid"`
	MinReplicas  int     `json:"min_replicas"`
	MaxReplicas  int     `json:"max_replicas"`
	TargetCPUPct int     `json:"target_cpu_pct"`
	TargetMemPct int     `json:"target_mem_pct,omitempty"`
	Source       string  `json:"source"`
	Confidence   float64 `json:"confidence"`
	WindowDays   int     `json:"window_days,omitempty"`
	Notes        string  `json:"notes,omitempty"`
}

type RecommendationRequest struct {
	TenantID    int64
	WorkloadUID string
	Environment string
	Criticality string
	HasDatadog  bool
}

// Client abstracts the titlis-insights recommendation API.
type Client interface {
	GetHpaRecommendation(ctx context.Context, req RecommendationRequest) (HpaRecommendation, error)
}

// HTTPClient calls the real titlis-insights service.
type HTTPClient struct {
	baseURL string
	secret  string
	http    *http.Client
}

// NewHTTPClient creates a client from a full base URL (e.g. "http://titlis-insights:8091").
func NewHTTPClient(baseURL, secret string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		secret:  secret,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func normalizeEnvironment(env string) string {
	switch env {
	case "dev", "hml", "prd":
		return env
	default:
		return "prd"
	}
}

func normalizeCriticality(crit string) string {
	switch crit {
	case "low", "medium", "high", "critical":
		return crit
	case "standard", "":
		return "medium"
	default:
		return "medium"
	}
}

func (c *HTTPClient) GetHpaRecommendation(ctx context.Context, req RecommendationRequest) (HpaRecommendation, error) {
	env := normalizeEnvironment(req.Environment)
	crit := normalizeCriticality(req.Criticality)
	u := fmt.Sprintf("%s/v1/recommendations/hpa?tenant_id=%d&workload_uid=%s&environment=%s&criticality=%s",
		c.baseURL, req.TenantID,
		url.QueryEscape(req.WorkloadUID),
		url.QueryEscape(env),
		url.QueryEscape(crit),
	)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return HpaRecommendation{}, err
	}
	httpReq.Header.Set("X-Internal-Secret", c.secret)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return HpaRecommendation{}, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return HpaRecommendation{}, fmt.Errorf("insights api %d", resp.StatusCode)
	}
	var reco HpaRecommendation
	if err := json.NewDecoder(resp.Body).Decode(&reco); err != nil {
		return HpaRecommendation{}, fmt.Errorf("decode: %w", err)
	}
	return reco, nil
}

// NoopClient always returns "skipped" — used when titlis-insights is not configured.
type NoopClient struct{}

func (NoopClient) GetHpaRecommendation(_ context.Context, _ RecommendationRequest) (HpaRecommendation, error) {
	return HpaRecommendation{Source: "skipped", Notes: "insights não configurado"}, nil
}
