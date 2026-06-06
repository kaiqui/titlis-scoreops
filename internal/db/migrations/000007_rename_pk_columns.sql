-- 000007: Aplica os renomes de PK/FK executados pelo DBA em produção via setimo_script_dba.sql.
-- Idempotente: usa blocos DO para só renomear se a coluna antiga ainda existir.

DO $$ BEGIN
    -- scoring_engines: id → scoring_engine_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'scoring_engines' AND column_name = 'id'
    ) THEN
        ALTER TABLE titlis_config.scoring_engines RENAME COLUMN id TO scoring_engine_id;
    END IF;

    -- engine_rules: id → engine_rule_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'engine_rules' AND column_name = 'id'
    ) THEN
        ALTER TABLE titlis_config.engine_rules RENAME COLUMN id TO engine_rule_id;
    END IF;

    -- engine_rules: engine_id → scoring_engine_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'engine_rules' AND column_name = 'engine_id'
    ) THEN
        ALTER TABLE titlis_config.engine_rules RENAME COLUMN engine_id TO scoring_engine_id;
    END IF;

    -- rule_overrides: id → rule_override_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'rule_overrides' AND column_name = 'id'
    ) THEN
        ALTER TABLE titlis_config.rule_overrides RENAME COLUMN id TO rule_override_id;
    END IF;

    -- rule_overrides: engine_id → scoring_engine_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'rule_overrides' AND column_name = 'engine_id'
    ) THEN
        ALTER TABLE titlis_config.rule_overrides RENAME COLUMN engine_id TO scoring_engine_id;
    END IF;

    -- pillar_weights: id → pillar_weight_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'pillar_weights' AND column_name = 'id'
    ) THEN
        ALTER TABLE titlis_config.pillar_weights RENAME COLUMN id TO pillar_weight_id;
    END IF;

    -- pillar_weights: engine_id → scoring_engine_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'pillar_weights' AND column_name = 'engine_id'
    ) THEN
        ALTER TABLE titlis_config.pillar_weights RENAME COLUMN engine_id TO scoring_engine_id;
    END IF;

    -- config_audit_log: id → config_audit_log_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'config_audit_log' AND column_name = 'id'
    ) THEN
        ALTER TABLE titlis_config.config_audit_log RENAME COLUMN id TO config_audit_log_id;
    END IF;

    -- workload_snapshots: id → workload_snapshot_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'workload_snapshots' AND column_name = 'id'
    ) THEN
        ALTER TABLE titlis_config.workload_snapshots RENAME COLUMN id TO workload_snapshot_id;
    END IF;

    -- workload_scores: id → workload_score_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'workload_scores' AND column_name = 'id'
    ) THEN
        ALTER TABLE titlis_config.workload_scores RENAME COLUMN id TO workload_score_id;
    END IF;

    -- score_history: id → score_history_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'score_history' AND column_name = 'id'
    ) THEN
        ALTER TABLE titlis_config.score_history RENAME COLUMN id TO score_history_id;
    END IF;

    -- tag_rule_policies: id → tag_rule_policie_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'titlis_config' AND table_name = 'tag_rule_policies' AND column_name = 'id'
    ) THEN
        ALTER TABLE titlis_config.tag_rule_policies RENAME COLUMN id TO tag_rule_policie_id;
    END IF;
END $$;
