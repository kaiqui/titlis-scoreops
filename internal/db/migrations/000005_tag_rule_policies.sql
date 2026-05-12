-- 000005: Políticas de scoring por tag.
-- Cada política desabilita regras (por rule_id OU por severity) para todos os
-- clusters/namespaces que carregam a tag correspondente.
-- As tags em si ficam em titlis_oltp.cluster_tags e namespace_tags (titlis-api).

CREATE TABLE IF NOT EXISTS titlis_config.tag_rule_policies (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   INT          NOT NULL,
    tag         VARCHAR(100) NOT NULL,
    rule_id     VARCHAR(128),
    severity    VARCHAR(32),
    action      VARCHAR(32)  NOT NULL DEFAULT 'disable',
    created_by  VARCHAR(256),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    CONSTRAINT tag_rule_policies_rule_xor_severity CHECK (
        (rule_id IS NOT NULL AND severity IS NULL) OR
        (rule_id IS NULL     AND severity IS NOT NULL)
    )
);

-- Garante unicidade de policy ativa por (tenant, tag, dimensão).
-- NULLS NOT DISTINCT requer PG 15+; scoreops já exige PG 15.
CREATE UNIQUE INDEX IF NOT EXISTS uq_tag_rule_policies_active
    ON titlis_config.tag_rule_policies (tenant_id, tag, COALESCE(rule_id, ''), COALESCE(severity, ''))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tag_rule_policies_tenant_tag
    ON titlis_config.tag_rule_policies (tenant_id, tag)
    WHERE deleted_at IS NULL;
