-- Seed das 26 regras padrão do motor kubernetes.
-- Idempotente: ON CONFLICT (engine_id, rule_id) DO NOTHING.
-- Fonte: titlis-operator-go/src/internal/scorecard/rules.go defaultRules()

WITH eng AS (
    SELECT id FROM titlis_config.scoring_engines WHERE slug = 'kubernetes'
)
INSERT INTO titlis_config.engine_rules
    (engine_id, rule_id, pillar, name, severity, enabled_by_default)
SELECT eng.id, r.rule_id, r.pillar, r.name, r.severity, TRUE
FROM eng, (VALUES
    -- Resilience
    ('RES-001', 'resilience', 'Liveness Probe',               'error'),
    ('RES-002', 'resilience', 'Readiness Probe',              'error'),
    ('RES-003', 'resilience', 'CPU Request',                  'error'),
    ('RES-004', 'resilience', 'CPU Limit',                    'warning'),
    ('RES-005', 'resilience', 'Memory Request',               'error'),
    ('RES-006', 'resilience', 'Memory Limit',                 'warning'),
    ('RES-007', 'resilience', 'HPA Exists',                   'warning'),
    ('RES-008', 'resilience', 'HPA Has Metrics',              'warning'),
    ('RES-009', 'resilience', 'Termination Grace Period',     'info'),
    ('RES-010', 'resilience', 'Run As Non Root',              'error'),
    ('RES-011', 'resilience', 'Pod Security Context',         'warning'),
    ('RES-012', 'resilience', 'Network Policy',               'warning'),
    ('RES-013', 'resilience', 'Minimum Replicas',             'warning'),
    ('RES-014', 'resilience', 'Deployment Strategy',          'warning'),
    ('RES-016', 'resilience', 'HPA Min Replicas',             'warning'),
    ('RES-017', 'resilience', 'HPA ScaleUp Stabilization',   'warning'),
    ('RES-018', 'resilience', 'HPA ScaleDown Stabilization', 'warning'),
    ('RES-019', 'resilience', 'HPA Behavior Policies',        'warning'),
    -- Security
    ('SEC-001', 'security',   'No Latest Image Tag',          'error'),
    ('SEC-002', 'security',   'Read-Only Root Filesystem',    'warning'),
    ('SEC-003', 'security',   'No Privilege Escalation',      'error'),
    ('SEC-004', 'security',   'Drop Capabilities',            'warning'),
    -- Performance
    ('PERF-001', 'performance', 'CPU Limit Ratio',            'warning'),
    ('PERF-002', 'performance', 'HPA CPU Target Range',       'info'),
    ('PERF-003', 'performance', 'HPA CPU Target Ceiling',     'info'),
    -- Operational
    ('OPS-001', 'operational', 'Datadog Labels',              'warning')
) AS r(rule_id, pillar, name, severity)
ON CONFLICT (engine_id, rule_id) DO NOTHING;
