CREATE TABLE IF NOT EXISTS titlis_config.queue_snapshots (
    queue_snapshot_id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id                  INT NOT NULL,
    provider                   VARCHAR(50)  NOT NULL,
    external_id                VARCHAR(500) NOT NULL,
    display_name               VARCHAR(255),
    is_dlq                     BOOLEAN NOT NULL DEFAULT false,
    num_undelivered_messages   BIGINT,
    oldest_unacked_age_seconds BIGINT,
    pull_message_count_rate    DOUBLE PRECISION,
    send_message_count_rate    DOUBLE PRECISION,
    ack_message_count_rate     DOUBLE PRECISION,
    dead_letter_message_count  BIGINT,
    has_dlq_configured         BOOLEAN,
    has_snapshot_policy        BOOLEAN,
    has_monitor_backlog        BOOLEAN,
    has_monitor_age            BOOLEAN,
    has_monitor_dlq            BOOLEAN,
    message_retention_sec      BIGINT,
    raw_metrics                JSONB,
    collected_at               TIMESTAMPTZ NOT NULL,
    UNIQUE (tenant_id, provider, external_id)
);

CREATE INDEX IF NOT EXISTS idx_queue_snapshots_tenant
    ON titlis_config.queue_snapshots (tenant_id, provider);

CREATE TABLE IF NOT EXISTS titlis_config.queue_scores (
    queue_score_id     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id          INT          NOT NULL,
    provider           VARCHAR(50)  NOT NULL,
    external_id        VARCHAR(500) NOT NULL,
    overall_score      DOUBLE PRECISION NOT NULL,
    compliance_status  VARCHAR(20)  NOT NULL,
    total_rules        INT,
    passed_rules       INT,
    failed_rules       INT,
    critical_failures  INT,
    error_count        INT,
    warning_count      INT,
    evaluated_at       TIMESTAMPTZ NOT NULL,
    UNIQUE (tenant_id, provider, external_id)
);

CREATE INDEX IF NOT EXISTS idx_queue_scores_tenant
    ON titlis_config.queue_scores (tenant_id, provider);
