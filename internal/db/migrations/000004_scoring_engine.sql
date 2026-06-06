CREATE TABLE IF NOT EXISTS titlis_config.workload_snapshots (
    workload_snapshot_id BIGSERIAL PRIMARY KEY,
    tenant_id            INT NOT NULL,
    engine_slug          VARCHAR(64) NOT NULL,
    workload_uid         VARCHAR(256) NOT NULL,
    cluster              VARCHAR(256) NOT NULL,
    namespace            VARCHAR(256) NOT NULL,
    workload_name        VARCHAR(256) NOT NULL,
    metrics_json         JSONB NOT NULL,
    rules_hash           VARCHAR(64) NOT NULL,
    received_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, engine_slug, workload_uid)
);

CREATE TABLE IF NOT EXISTS titlis_config.workload_scores (
    workload_score_id BIGSERIAL PRIMARY KEY,
    tenant_id         INT NOT NULL,
    engine_slug       VARCHAR(64) NOT NULL,
    workload_uid      VARCHAR(256) NOT NULL,
    cluster           VARCHAR(256) NOT NULL,
    namespace         VARCHAR(256) NOT NULL,
    workload_name     VARCHAR(256) NOT NULL,
    overall_score     NUMERIC(5,2) NOT NULL,
    compliance_status VARCHAR(32) NOT NULL,
    critical_issues   INT NOT NULL DEFAULT 0,
    error_issues      INT NOT NULL DEFAULT 0,
    warning_issues    INT NOT NULL DEFAULT 0,
    passed_checks     INT NOT NULL DEFAULT 0,
    total_checks      INT NOT NULL DEFAULT 0,
    pillar_json       JSONB NOT NULL,
    findings_json     JSONB NOT NULL,
    rules_hash        VARCHAR(64) NOT NULL,
    calculated_at     TIMESTAMPTZ NOT NULL,
    UNIQUE (tenant_id, engine_slug, workload_uid)
);

CREATE TABLE IF NOT EXISTS titlis_config.score_history (
    score_history_id BIGSERIAL PRIMARY KEY,
    tenant_id        INT NOT NULL,
    engine_slug      VARCHAR(64) NOT NULL,
    workload_uid     VARCHAR(256) NOT NULL,
    cluster          VARCHAR(256) NOT NULL,
    namespace        VARCHAR(256) NOT NULL,
    overall_score    NUMERIC(5,2) NOT NULL,
    pillar_json      JSONB NOT NULL,
    findings_json    JSONB NOT NULL,
    trigger_type     VARCHAR(64) NOT NULL,
    rules_hash       VARCHAR(64) NOT NULL,
    calculated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS workload_snapshots_tenant_cluster_ns
    ON titlis_config.workload_snapshots (tenant_id, engine_slug, cluster, namespace);

CREATE INDEX IF NOT EXISTS workload_scores_tenant_cluster_ns
    ON titlis_config.workload_scores (tenant_id, engine_slug, cluster, namespace);

CREATE INDEX IF NOT EXISTS score_history_tenant_uid_time
    ON titlis_config.score_history (tenant_id, workload_uid, calculated_at DESC);
