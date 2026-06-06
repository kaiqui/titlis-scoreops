CREATE SCHEMA IF NOT EXISTS titlis_config;

-- migration tracking
CREATE TABLE IF NOT EXISTS titlis_config.schema_migrations (
    version     VARCHAR(64) PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- engine registry
CREATE TABLE IF NOT EXISTS titlis_config.scoring_engines (
    scoring_engine_id SERIAL PRIMARY KEY,
    slug              VARCHAR(64)  NOT NULL UNIQUE,
    name              VARCHAR(128) NOT NULL,
    description       TEXT,
    enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- rules per engine
CREATE TABLE IF NOT EXISTS titlis_config.engine_rules (
    engine_rule_id     SERIAL PRIMARY KEY,
    scoring_engine_id  INT NOT NULL REFERENCES titlis_config.scoring_engines(scoring_engine_id) ON DELETE CASCADE,
    rule_id            VARCHAR(128) NOT NULL,
    pillar             VARCHAR(64)  NOT NULL,
    name               VARCHAR(256) NOT NULL,
    description        TEXT,
    severity           VARCHAR(32)  NOT NULL DEFAULT 'medium',
    enabled_by_default BOOLEAN NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (scoring_engine_id, rule_id)
);

DO $$ BEGIN
    CREATE TYPE titlis_config.scope_type AS ENUM (
        'tenant',
        'cluster',
        'namespace',
        'workload'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- rule overrides per tenant/granularity
CREATE TABLE IF NOT EXISTS titlis_config.rule_overrides (
    rule_override_id  BIGSERIAL PRIMARY KEY,
    tenant_id         INT NOT NULL,
    scoring_engine_id INT NOT NULL REFERENCES titlis_config.scoring_engines(scoring_engine_id) ON DELETE CASCADE,
    rule_id           VARCHAR(128) NOT NULL,
    scope             titlis_config.scope_type NOT NULL,
    cluster_name      VARCHAR(256),
    namespace         VARCHAR(256),
    workload_uid      VARCHAR(256),
    enabled           BOOLEAN NOT NULL,
    reason            TEXT,
    created_by        VARCHAR(256),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique key por granularidade: usa COALESCE p/ tratar NULLs como '' na comparação
CREATE UNIQUE INDEX IF NOT EXISTS idx_rule_overrides_unique
    ON titlis_config.rule_overrides
    (tenant_id, scoring_engine_id, rule_id, scope,
     COALESCE(cluster_name, ''), COALESCE(namespace, ''), COALESCE(workload_uid, ''));

CREATE INDEX IF NOT EXISTS idx_rule_overrides_tenant_engine
    ON titlis_config.rule_overrides (tenant_id, scoring_engine_id, scope);

CREATE INDEX IF NOT EXISTS idx_rule_overrides_cluster
    ON titlis_config.rule_overrides (tenant_id, scoring_engine_id, cluster_name)
    WHERE scope IN ('cluster', 'namespace', 'workload');

-- pillar weights per tenant/engine
CREATE TABLE IF NOT EXISTS titlis_config.pillar_weights (
    pillar_weight_id  SERIAL PRIMARY KEY,
    tenant_id         INT NOT NULL,
    scoring_engine_id INT NOT NULL REFERENCES titlis_config.scoring_engines(scoring_engine_id) ON DELETE CASCADE,
    pillar            VARCHAR(64) NOT NULL,
    weight            SMALLINT NOT NULL CHECK (weight >= 5 AND weight <= 60),
    updated_by        VARCHAR(256) NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, scoring_engine_id, pillar)
);

-- audit log (append-only)
CREATE TABLE IF NOT EXISTS titlis_config.config_audit_log (
    config_audit_log_id BIGSERIAL PRIMARY KEY,
    tenant_id           INT NOT NULL,
    actor               VARCHAR(256) NOT NULL,
    action              VARCHAR(64)  NOT NULL,
    entity_type         VARCHAR(64)  NOT NULL,
    entity_id           TEXT NOT NULL,
    before_json         JSONB,
    after_json          JSONB NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_tenant_created
    ON titlis_config.config_audit_log (tenant_id, created_at DESC);

-- seed engine kubernetes (idempotente)
INSERT INTO titlis_config.scoring_engines (slug, name, description)
VALUES ('kubernetes', 'Kubernetes Compliance', 'Avaliação de compliance de workloads Kubernetes')
ON CONFLICT (slug) DO NOTHING;
