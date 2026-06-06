-- Seed das regras do pillar Observabilidade (OBS-001, OBS-002).
-- Idempotente: ON CONFLICT (scoring_engine_id, rule_id) DO NOTHING.
--
-- OBS-001: SLO Configurado — verifica se existe ao menos um SLOConfig para o namespace.
--          Skip automático quando has_datadog=false (lógica no pillar, sem peso no score).
-- OBS-002: SLO Sincronizado — verifica se o SLO está em estado saudável no Datadog.
--          Skip quando has_slo=false (sem penalidade dupla com OBS-001).

WITH eng AS (
    SELECT scoring_engine_id FROM titlis_config.scoring_engines WHERE slug = 'kubernetes'
)
INSERT INTO titlis_config.engine_rules
    (scoring_engine_id, rule_id, pillar, name, severity, enabled_by_default)
SELECT eng.scoring_engine_id, r.rule_id, r.pillar, r.name, r.severity, r.enabled
FROM eng, (VALUES
    ('OBS-001', 'observability', 'SLO Configurado',   'warning', TRUE),
    ('OBS-002', 'observability', 'SLO Sincronizado',  'warning', TRUE)
) AS r(rule_id, pillar, name, severity, enabled)
ON CONFLICT (scoring_engine_id, rule_id) DO NOTHING;
