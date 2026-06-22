package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/titlis/scoreops/internal/coverage"
	"github.com/titlis/scoreops/internal/middleware"
)

// HTTP-level e2e of the coverage scoring slice: authenticated POST → personalized scorecard JSON.
func TestCoverageEvaluate_HTTP(t *testing.T) {
	h := NewCoverageEvaluateHandler(coverage.NewEngine(nil))
	r := chi.NewRouter()
	r.Use(middleware.InternalSecret("s3cret"))
	r.Post("/v1/scoring/coverage/evaluate", h.Evaluate)
	srv := httptest.NewServer(r)
	defer srv.Close()

	snap := coverage.CoverageSnapshot{
		TenantID: 1, WorkloadUID: "u1", ServiceName: "orders",
		Nature: coverage.Nature{Language: "java", HTTPFacing: true, Criticality: "high"},
		Found: coverage.Found{
			HasSLO: true, HasMonitor: true, HasTracing: true, HasLogs: true,
			MetricCategories: []string{"http", "jvm"},
			CPURequestSet:    true, CPULimitSet: true, MemoryRequestSet: true, MemoryLimitSet: true,
			HasProbes: true, HasHPA: true, HasPDB: true, HasNetworkPolicy: true,
		},
		Capabilities: []string{"monitor", "tracing", "metrics", "logs"},
	}
	body, _ := json.Marshal(snap)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/scoring/coverage/evaluate", bytes.NewReader(body))
	req.Header.Set("X-Internal-Secret", "s3cret")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var res coverage.CoverageResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if res.EngineSlug != "coverage" || res.TrustScore != 100 {
		t.Errorf("unexpected result: engine=%q trust=%v", res.EngineSlug, res.TrustScore)
	}
	if len(res.Findings) == 0 {
		t.Error("expected personalized findings")
	}

	// Without the secret the endpoint must reject.
	req2, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/scoring/coverage/evaluate", bytes.NewReader(body))
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode == http.StatusOK {
		t.Error("expected auth failure without X-Internal-Secret")
	}
}
