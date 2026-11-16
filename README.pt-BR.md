[English](README.md)

# forgeloop

Orquestrador de **Loop Engineering** orientado a especificações, TDD estrito e auditoria de segurança automatizada. Escrito em Go **sem dependências externas** (apenas a biblioteca padrão) e projetado para operar 100% via sessões de CLI autenticadas por OAuth (`claude` e `agy`) — sem chaves de API.

## Uma Forja, Seis Etapas

**A Forja (`forgeloop`)** é o ciclo completo. Cada fase interna é uma **Etapa**:

| Etapa | Nome | Agente | O que faz |
| :--- | :--- | :--- | :--- |
| 0 | `DISCOVERY` | `ONBOARDER` | Mapeia um projeto existente e consolida a documentação em `<docs>/PROJECT_CONTEXT.md`. |
| 1 | `BLUEPRINT` | `ARCHITECT` | Planejamento incremental (`obra/superpowers`) e testes automatizados em estado `RED`. |
| 2 | `TEMPER` | `SKEPTIC` | Revisão crítica a frio e invariantes globais (`google/elephant-goldfish-model`). |
| 3 | `STRIKE` | `BUILDER` | Implementação em contexto higienizado (*stateless*). |
| 4 | `QUENCH` | `SENTINEL` | Suíte determinística da stack (4A) + auditoria AppSec no `git diff` (4B, `cloudflare/security-audit-skill`). |
| 5 | `ASSAY` | `ARBITER` | Julgamento semântico, contabilização de tokens e *Circuit Breaker* (`max-attempts`). |

As Etapas 3 → 4 → 5 rodam em loop fechado até o ARBITER aprovar, pedir correção de teste, escalar para o arquiteto ou o limite de tentativas ser atingido.

## Requisitos

- Go 1.22+
- CLIs `claude` e `agy` autenticadas via OAuth
- `npx` (Node.js) e `git`

## Uso Rápido

### 1. Preparar o ambiente — `-init`

```bash
go run forgeloop.go -init                 # ou: ./forgeloop -init
go run forgeloop.go -init -force-skills   # reinstala/atualiza todas as skills
```

O `-init`:

1. verifica no `$PATH` os binários configurados (`claude`, `agy`, `npx`, `git`) e mostra `[OK]` ou `[AUSENTE]`;
2. coleta as skills únicas das variáveis `FORGE_*_SKILL` (`.forge/config/models.env` / ambiente);
3. instala as ausentes com `npx skills add <skill> -g -y` (todas, com `-force-skills`). Uma skill conta como presente se existir uma pasta com o nome dela em `skills/`, `.agents/skills/`, `~/.claude/skills/` ou `~/.antigravity/skills/`, ou se o `skills-lock.json` a listar como origem;
4. cria `.forge/{agents,config,state}/`, `<docs>/demands/`, `<docs>/logs/` e `<docs>/references/` e grava o próprio log em `<docs>/logs/`.

Termina com código `1` se faltar algum binário ou se a instalação de alguma skill falhar. Toda execução normal também faz um *pre-flight* rápido e avisa (sem bloquear) se alguma skill configurada não estiver instalada.

### 2. Onboarding de projeto existente — `-setup` (Etapa 0)

```bash
go run forgeloop.go -setup
```

Cria `<docs>/PROJECT_CONTEXT.md`, `<docs>/demands/INDEX.md`, `<docs>/demands/`, `<docs>/logs/` e `<docs>/references/`.

### 3. Iniciar nova demanda — `-task` (Etapas 1 a 5)

```bash
go run forgeloop.go -task "Implementar autenticação JWT com refresh token"
```

Enriqueça a demanda com material de `<docs>/references/` usando `-ref` (repetível). Cada arquivo é validado antes de tudo e entregue ao ARCHITECT como `@caminho`, então é lido antes do planejamento:

```bash
go run forgeloop.go -task "Implementar autenticação JWT com refresh token" \
  -ref auth-spec.md -ref docs/references/openapi.yaml
```

`-ref` aceita um caminho (relativo ao diretório atual ou absoluto) ou apenas o nome de um arquivo dentro de `<docs>/references/` (acompanha `-docs-dir` / `FORGE_DOCS_DIR`). Arquivo inexistente aborta antes de qualquer agente rodar. `-ref` só afeta a Etapa 1, então é ignorado (com aviso) ao iniciar com `-stage 2` ou superior.

Comece de uma Etapa posterior com `-stage` (`-forge` continua funcionando como alias de compatibilidade):

```bash
go run forgeloop.go -stage 2    # a partir de TEMPER (reaproveita .forge/state/PLAN.md / testes existentes)
go run forgeloop.go -stage 3    # a partir de STRIKE
```

### 4. Testar combinações de modelos

Edite `.forge/config/models.env` ou sobrescreva via variáveis de ambiente:

```bash
FORGE_BUILDER_MODEL="GPT-OSS 120B (Medium)" go run forgeloop.go -task "Adicionar paginação por cursor"
```

Cada agente tem `FORGE_<AGENTE>_BIN`, `_MODEL` e `_ARGS` (flags extras do CLI, ex.: permissões necessárias no modo headless). `--dangerously-skip-permissions` desativa os prompts de permissão: use apenas em repositório confiável.

## Estrutura gerada

```text
<docs>/
├── PROJECT_CONTEXT.md     # contexto do projeto (Etapa 0)
├── demands/               # um arquivo por demanda, ligado ao log da execução
│   └── INDEX.md           # | Data | Stack | Demanda | Arquivo | Status |
├── logs/                  # um log completo por execução: AAAA-MM-DD_HHMMSS-<slug>.log
└── references/            # material de referência (criada por -init / -setup)
.forge/
├── agents/                # contratos dos agentes (00_ONBOARDER.md ... 05_ARBITER.md)
├── config/                # models.env (versionado), project.env (ignorado pelo git)
├── state/                 # artefatos transitórios: PLAN, TEST_CASES, DECISION, ... (ignorado)
└── telemetry/             # métricas em CSV (cabeçalho criado no primeiro uso, depois só append)
    ├── runs.csv           # uma linha por execução: tentativas, tokens in/out/total, status final
    └── events.csv         # uma linha por etapa/agente: tokens, duração, tentativa (junta com runs.csv via RunID)
```

Os logs trazem o prompt e a resposta completa de cada agente, a saída integral de testes/linters, os relatórios de revisão/segurança/decisão, durações, contagem de tokens e o código de saída final.

## Códigos de saída

| Código | Significado |
| :--- | :--- |
| `0` | `APPROVE_MERGE` (ou `-init`/`-setup` bem-sucedido) |
| `1` | Erro fatal ou circuit breaker acionado |
| `2` | Especificação bloqueada pelo SKEPTIC (Etapa 2) |
| `3` | `AMEND_TEST` — teste da Etapa 1 precisa de correção |
| `4` | `ESCALATE_ARCHITECT` |

## Testes

```bash
go test -v -race ./... && go vet ./... && go build -o forgeloop forgeloop.go
```

## Licença

Distribuído sob a **Apache License 2.0**. Veja `LICENSE`.
