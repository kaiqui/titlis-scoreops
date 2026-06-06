# CLAUDE.md — titlis-scoreops

> Após toda alteração: `go build ./...` e `go test ./...` devem passar.
> Para rodar localmente: `make run` (requer `SCOREOPS_DATABASE_URL` e `SCOREOPS_INTERNAL_SECRET`).

---

## 1. Visão Geral

O **titlis-scoreops** é o motor de cálculo de scores do Titlis — um serviço Go independente
que centraliza toda a lógica de avaliação de compliance de workloads Kubernetes.

Responsabilidades:
- **Avaliação** — recebe `WorkloadSnapshot` do operator-go, aplica as regras dos pilares,
  calcula score ponderado por pilares
- **Persistência** — persiste snapshots e scores no banco de dados (schema `titlis_scoreops`)
- **Notificação** — após calcular, dispara evento `scorecard_evaluated` via HTTP POST para
  `titlis-api`, que o armazena no `titlis_oltp.app_scorecards`
- **Recalculação** — quando overrides ou pesos mudam, o `RecalcWorker` recalcula scores
  dos últimos snapshots afetados sem precisar de nova avaliação do operator
- **API de configuração** — CRUD de engines, regras, overrides, pesos por tenant

O titlis-scoreops **não acessa a API do Kubernetes**. Ele é acionado passivamente pelo
operator-go via `POST /v1/scoring/evaluate`.

---

## 2. Stack

| Categoria | Tecnologia | Versão |
|---|---|---|
| Linguagem | Go | 1.22 |
| Roteamento HTTP | chi v5 | 5.1.0 |
| Banco | PostgreSQL 15 (pgx v5) | — |
| Config | kelseyhightower/envconfig | 1.4.0 |
| Porta padrão | 8090 | — |

---

## 3. Estrutura do Projeto

```
titlis-scoreops/
├── cmd/scoreops/main.go        # Entrypoint: engine, repos, handlers, router chi
└── internal/
    ├── config/
    │   └── config.go           # Settings (envconfig)
    ├── db/
    │   ├── connect.go          # pgx pool
    │   └── migrate.go          # Migrations embutidas
    ├── domain/                 # Tipos de domínio compartilhados
    ├── handler/
    │   ├── evaluate.go         # POST /v1/scoring/evaluate
    │   ├── engine.go           # CRUD de engines e regras
    │   ├── override.go         # CRUD de overrides por tenant
    │   ├── weight.go           # GET/PUT de pesos por tenant
    │   ├── audit.go            # GET /tenants/{id}/audit
    │   └── scores.go           # GET /tenants/{id}/scores[/{uid}]
    ├── middleware/
    │   └── auth.go             # InternalSecret middleware
    ├── notifier/
    │   ├── titlisapi.go        # Envia scorecard_evaluated para titlis-api
    │   └── noop.go             # NoopNotifier para testes
    ├── pillar/
    │   ├── resilience.go       # Pillar: Resiliência (replicas, HPA, PDB, etc.)
    │   ├── security.go         # Pillar: Segurança (rootless, seccomp, NetworkPolicy)
    │   ├── performance.go      # Pillar: Performance (requests, limits, VPA)
    │   └── operational.go      # Pillar: Operacional (probes, labels, owners)
    ├── repository/
    │   ├── engine_repo.go      # CRUD de engines e EngineRules
    │   ├── override_repo.go    # CRUD de overrides (disable rules per workload/cluster)
    │   ├── weight_repo.go      # CRUD de PillarWeightsConfig
    │   ├── audit_repo.go       # Append de entradas de audit
    │   ├── snapshot_repo.go    # UpsertSnapshot (estado vivo do workload)
    │   └── score_repo.go       # UpsertScore + AppendHistory
    ├── scoring/
    │   ├── engine.go           # ScoreEngine: orquestra pilares, calcula score final
    │   ├── context.go          # ContextResolver: resolve regras ativas e pesos
    │   ├── dag.go              # DAG de dependências entre regras (ex: OPS001 bloqueia OPS002)
    │   ├── result.go           # ScoreResult: score, pilares, findings
    │   └── snapshot.go         # WorkloadSnapshot: estado extraído do Deployment
    └── worker/
        └── recalc.go           # RecalcWorker: recalcula scores quando config muda
```

---

## 4. Fluxo de Avaliação

```
POST /v1/scoring/evaluate  (autenticado por X-Internal-Secret)
  ↓
EvaluateHandler.Evaluate(w, r)
  ├── json.Decode → WorkloadSnapshot
  ├── resolver.ResolveActiveRules(ctx, tenantID, engineSlug, snap)
  │   ├── Busca regras do engine no banco
  │   ├── Aplica overrides (cluster-level e workload-level)
  │   └── Retorna activeRules + hash (MD5 das regras ativas)
  ├── resolver.ResolveWeights(ctx, tenantID, engineSlug)
  ├── engine.Evaluate(snap, activeRules, weights)
  │   ├── Para cada pillar registrado → pillar.Evaluate(snap, rules)
  │   ├── Calcula score ponderado por pillar
  │   └── Retorna ScoreResult{Score, Pillars, Findings, ...}
  ├── snapshots.UpsertSnapshot(ctx, snap, rulesHash)
  ├── scores.UpsertScore(ctx, result)
  ├── scores.AppendHistory(ctx, result, "operator_push")
  └── notif.Notify(ctx, result)
      └── POST http://titlis-api/v1/internal/scoreops/scorecard-evaluated
          → titlis-api persiste em app_scorecards
```

---

## 5. API Endpoints

**Autenticação:** header `X-Internal-Secret` obrigatório em todos os endpoints exceto `/healthz`.

```
GET  /healthz                                → { status: "ok" } (sem auth)

# Avaliação
POST /v1/scoring/evaluate                    → ScoreResult

# Engines e regras
GET  /engines                                → []Engine
POST /engines                                → Engine (201)
PATCH /engines/{slug}/enabled               → 200
GET  /engines/{slug}/rules                  → []EngineRule
POST /engines/{slug}/rules                  → EngineRule (201)

# Por tenant
GET  /tenants/{tenantId}/overrides          → []Override
POST /tenants/{tenantId}/overrides          → Override (201)
DELETE /tenants/{tenantId}/overrides/{id}   → 204
GET  /tenants/{tenantId}/overrides/resolve  → activeRules para um workload

GET  /tenants/{tenantId}/weights            → []PillarWeight
PUT  /tenants/{tenantId}/weights            → []PillarWeight

GET  /tenants/{tenantId}/audit              → []AuditEntry (paginado)

GET  /tenants/{tenantId}/scores             → []ScoreRecord (filtros: cluster, namespace)
GET  /tenants/{tenantId}/scores/{workloadUid} → ScoreRecord
```

---

## 6. RecalcWorker

Quando overrides ou pesos mudam, o recalc é necessário para refletir no dashboard sem
precisar de novo evento do operator. O `RecalcWorker` opera com um canal buffered (256):

```go
recalcWorker.Enqueue(workloadUID, tenantID, engineSlug)
```

O worker busca o último snapshot do workload, resolve as novas regras/pesos, recalcula
e persiste o novo score + notifica titlis-api.

---

## 7. Variáveis de Ambiente

```bash
SCOREOPS_PORT=8090
SCOREOPS_DATABASE_URL=postgres://titlis:titlis@localhost:5432/titlis?sslmode=disable
SCOREOPS_INTERNAL_SECRET=<segredo-compartilhado>
SCOREOPS_PILLAR_MIN_WEIGHT=5
SCOREOPS_PILLAR_MAX_WEIGHT=60
SCOREOPS_LOG_LEVEL=info

# Notificação para titlis-api (opcional — sem essa var, usa NoopNotifier)
SCOREOPS_TITLISAPI_URL=http://titlis-api.titlis-system.svc.cluster.local:8080
SCOREOPS_TITLISAPI_KEY=<api-key-do-operator>
```

Se `SCOREOPS_TITLISAPI_URL` não for definido, o serviço funciona em modo standalone
(calcula e persiste scores mas não notifica titlis-api).

---

## 8. Banco de Dados

O scoreops usa um schema próprio no PostgreSQL (`titlis_config`).
Migrations são aplicadas via `db.Migrate()` no startup (Go embeds SQL files em `internal/db/migrations/`).

**Tabelas principais:**
- `scoring_engines` — motores de scoring (PK: `scoring_engine_id`)
- `engine_rules` — regras por engine (PK: `engine_rule_id`, FK: `scoring_engine_id`)
- `rule_overrides` — desabilitações por escopo (PK: `rule_override_id`, FK: `scoring_engine_id`)
- `pillar_weights` — pesos por `(tenant_id, scoring_engine_id, pillar)`
- `workload_snapshots` — último estado extraído (PK: `workload_snapshot_id`)
- `workload_scores` — score atual por `(tenant_id, engine_slug, workload_uid)` (PK: `workload_score_id`)
- `score_history` — histórico append-only (PK: `score_history_id`)
- `config_audit_log` — log de mudanças de config (PK: `config_audit_log_id`)
- `tag_rule_policies` — políticas por tag (PK: `tag_rule_policie_id`)

### Convenção de nomenclatura de colunas (decisão do DBA — obrigatória)

**PKs:** nunca use `id` genérico. O nome deve ser `<nome_da_tabela>_id`.
```sql
-- Correto
CREATE TABLE titlis_config.minha_tabela (
    minha_tabela_id SERIAL PRIMARY KEY,
    ...
);

-- Errado — rejeitado em review
CREATE TABLE titlis_config.minha_tabela (
    id SERIAL PRIMARY KEY,  -- NÃO USE
    ...
);
```

**FKs:** o nome da coluna deve ser idêntico ao PK que ela referencia.
```sql
-- Correto: FK referencia scoring_engines.scoring_engine_id → coluna se chama scoring_engine_id
minha_tabela_id INT NOT NULL REFERENCES titlis_config.scoring_engines(scoring_engine_id)

-- Errado — rejeitado em review
engine_id INT NOT NULL REFERENCES titlis_config.scoring_engines(scoring_engine_id)  -- NÃO USE
```

### Adicionando uma nova migration

1. Cria `internal/db/migrations/<próximo_número>_<descricao>.sql`
2. O bootstrap do `db.Migrate()` aplica automaticamente na ordem numérica
3. Use `IF NOT EXISTS` / `ON CONFLICT` para manter idempotência
4. Se a migration renomeia colunas em banco já existente, use blocos `DO $$ BEGIN ... END $$` para verificar antes de renomear (veja `000007_rename_pk_columns.sql` como referência)

---

## 9. Adicionar um Novo Pillar

1. Cria `internal/pillar/<nome>.go` implementando a interface `scoring.Pillar`:
   ```go
   type Pillar interface {
       Name() string
       Rules() []scoring.Rule
       Evaluate(snap scoring.WorkloadSnapshot, active []scoring.Rule) scoring.PillarResult
   }
   ```
2. Registra no `main.go`:
   ```go
   engine.RegisterPillar(pillar.NewMeuPillar())
   ```
3. Adiciona as regras do pillar no banco via `POST /engines/{slug}/rules`
4. Adiciona documentação da regra via `KnowledgeSeeder` em titlis-ai (contexto RAG)

---

## 10. Adicionar uma Nova Regra

Regras são declaradas pelo pillar via `Rules()` e armazenadas no banco.
Para adicionar uma nova regra a um pillar existente:
1. Implementa a lógica de check em `internal/pillar/<nome>.go`
2. Declara a regra em `Rules()` com `ID`, `Pillar`, `Severity`, `EnabledByDefault`
3. A regra é automaticamente sincronizada no banco na primeira avaliação (ou via endpoint)

---

## 11. O Que Não Fazer

- **Nunca** acesse o Kubernetes API diretamente — o scoreops recebe snapshots passivamente
- **Nunca** modifique scores sem passar pelo `engine.Evaluate` — a lógica de score é o DAG
- **Nunca** omita `tenantID` em queries — isolamento multi-tenant é obrigatório
- **Nunca** delete registros de `score_history` — é append-only por design
- **Nunca** retorne `api_key` ou `github_token` em respostas JSON
