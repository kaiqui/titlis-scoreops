package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/titlis/scoreops/internal/config"
	"github.com/titlis/scoreops/internal/db"
	"github.com/titlis/scoreops/internal/handler"
	"github.com/titlis/scoreops/internal/middleware"
	"github.com/titlis/scoreops/internal/notifier"
	"github.com/titlis/scoreops/internal/pillar"
	"github.com/titlis/scoreops/internal/repository"
	"github.com/titlis/scoreops/internal/scoring"
	"github.com/titlis/scoreops/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToUpper(cfg.LogLevel))); err != nil {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBPoolMax)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	// scoring engine
	engine := scoring.NewScoreEngine()
	engine.RegisterPillar(pillar.NewResiliencePillar())
	engine.RegisterPillar(pillar.NewSecurityPillar())
	engine.RegisterPillar(pillar.NewPerformancePillar())
	engine.RegisterPillar(pillar.NewOperationalPillar())
	engine.RegisterPillar(pillar.NewObservabilityPillar())

	resolver := scoring.NewContextResolver(pool, engine.Pillars())

	// repositories
	engineRepo   := repository.NewEngineRepo(pool)
	overrideRepo := repository.NewOverrideRepo(pool)
	weightRepo   := repository.NewWeightRepo(pool)
	auditRepo    := repository.NewAuditRepo(pool)
	snapshotRepo := repository.NewSnapshotRepo(pool)
	scoreRepo    := repository.NewScoreRepo(pool)

	// notifier
	var scoreNotifier notifier.ScorecardNotifier
	if cfg.TitlisAPIURL != "" {
		scoreNotifier = notifier.NewTitlisAPINotifier(cfg.TitlisAPIURL, cfg.InternalSecret)
		slog.Info("notifier configured", "url", cfg.TitlisAPIURL)
	} else {
		scoreNotifier = notifier.NoopNotifier{}
		slog.Info("notifier disabled — SCOREOPS_TITLISAPI_URL not set")
	}

	// recalc worker
	recalcWorker := worker.NewRecalcWorker(256, snapshotRepo, scoreRepo, engine, resolver, scoreNotifier)
	go recalcWorker.Start(ctx)

	// handlers
	engineH := handler.NewEngineHandler(engineRepo, auditRepo)
	overrideH := handler.NewOverrideHandler(overrideRepo, auditRepo, engineRepo, snapshotRepo, recalcWorker)
	weightH := handler.NewWeightHandler(weightRepo, auditRepo, engineRepo, snapshotRepo, recalcWorker, cfg.PillarMinWeight, cfg.PillarMaxWeight)
	auditH := handler.NewAuditHandler(auditRepo)
	evaluateH := handler.NewEvaluateHandler(engine, resolver, snapshotRepo, scoreRepo, scoreNotifier)
	scoresH := handler.NewScoresHandler(scoreRepo)
	tagPolicyRepo := repository.NewTagPolicyRepo(pool)
	tagPolicyH := handler.NewTagPolicyHandler(tagPolicyRepo)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(requestLogger)

	// healthz — sem autenticação (usado pelos probes do Kubernetes)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, `{"status":"unhealthy"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// rotas autenticadas — exigem X-Internal-Secret
	r.Group(func(r chi.Router) {
		r.Use(middleware.InternalSecret(cfg.InternalSecret))

		// scoring
		r.Post("/v1/scoring/evaluate", evaluateH.Evaluate)

		// engines
		r.Get("/engines", engineH.List)
		r.Post("/engines", engineH.Create)
		r.Patch("/engines/{slug}/enabled", engineH.PatchEnabled)
		r.Get("/engines/{slug}/rules", engineH.ListRules)
		r.Post("/engines/{slug}/rules", engineH.CreateRule)

		// overrides, weights, audit e scores (por tenant)
		r.Route("/tenants/{tenantId}", func(r chi.Router) {
			r.Get("/overrides", overrideH.List)
			r.Post("/overrides", overrideH.Create)
			r.Delete("/overrides/{id}", overrideH.Delete)
			r.Get("/overrides/resolve", overrideH.Resolve)

			r.Get("/weights", weightH.Get)
			r.Put("/weights", weightH.Set)

			r.Get("/audit", auditH.List)

			r.Get("/scores", scoresH.List)
			r.Get("/scores/{workloadUid}", scoresH.Get)

			r.Get("/tag-policies", tagPolicyH.List)
			r.Post("/tag-policies", tagPolicyH.Create)
			r.Delete("/tag-policies/{id}", tagPolicyH.Delete)
		})
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("scoreops starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		t := time.Now()
		defer func() {
			slog.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(t).Milliseconds(),
				"request_id", chimiddleware.GetReqID(r.Context()),
			)
		}()
		next.ServeHTTP(ww, r)
	})
}
