-- Seed da regra OPS-002 no pillar Operacional.
-- Idempotente: ON CONFLICT (scoring_engine_id, rule_id) DO NOTHING.
--
-- OPS-002: Backstage Registration — verifica se o workload possui a annotation
--          "backstage.io/kubernetes-id" registrada (campo backstage_component no snapshot).
--          Severity "error": ausência indica falta de governança de catálogo.

WITH eng AS (
    SELECT scoring_engine_id FROM titlis_config.scoring_engines WHERE slug = 'kubernetes'
)
INSERT INTO titlis_config.engine_rules
    (scoring_engine_id, rule_id, pillar, name, severity, enabled_by_default)
SELECT eng.scoring_engine_id, r.rule_id, r.pillar, r.name, r.severity, r.enabled
FROM eng, (VALUES
    ('OPS-002', 'operational', 'Backstage Registration', 'error', TRUE)
) AS r(rule_id, pillar, name, severity, enabled)
ON CONFLICT (scoring_engine_id, rule_id) DO NOTHING;
