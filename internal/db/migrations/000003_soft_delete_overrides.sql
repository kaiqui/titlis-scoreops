-- Adiciona suporte a soft delete na tabela rule_overrides.
-- Rodar manualmente em produção antes do deploy do scoreops com essa versão.
ALTER TABLE titlis_config.rule_overrides
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_rule_overrides_active
    ON titlis_config.rule_overrides (tenant_id, scoring_engine_id)
    WHERE deleted_at IS NULL;
