[English](README.md)

# forgeloop

Orquestrador de **Loop Engineering** orientado a especificações, TDD estrito e auditoria de segurança automatizada. Escrito em Go **sem dependências externas** (apenas a biblioteca padrão).

## Uma Forja, Seis Etapas

**A Forja (`forgeloop`)** é o ciclo completo. Cada fase interna é uma **Etapa**:

| Etapa | Nome | Agente | Cliente | Modelo Padrão | O que faz |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 0 | `DISCOVERY` | `ONBOARDER` | `claude` | `opus` | Mapeia um projeto existente e consolida a documentação em `<docs>/PROJECT_CONTEXT.md`. |
| 1 | `BLUEPRINT` | `ARCHITECT` | `claude` | `opus` | Planejamento incremental (`obra/superpowers`) e testes automatizados em estado `RED`. |
| 2 | `TEMPER` | `SKEPTIC` | `agy` | `Gemini 3.8 Flash (Medium)` | Revisão crítica a frio e invariantes globais (`google/elephant-goldfish-model`). |
| 3 | `STRIKE` | `BUILDER` | `agy` | `Gemini 3.8 Flash (High)` | Implementação em contexto higienizado (*stateless*). |
| 4 | `QUENCH` | `SENTINEL` | `agy` *(4B)* | `Gemini 3.8 Flash (Medium)` *(4B)* | Suíte determinística da stack (4A) + auditoria AppSec no `git diff` (4B, `cloudflare/security-audit-skill`). |
| 5 | `ASSAY` | `ARBITER` | `agy` | `Gemini 3.8 Flash (High)` | Julgamento semântico, contabilização de tokens e *Circuit Breaker* (`max-attempts`). |

> **Nota:** Na Etapa 4 (`QUENCH`), a fase 4A executa a suíte de testes e linters locais deterministicamente (sem uso de LLM), enquanto a fase 4B executa a auditoria AppSec via agente `SENTINEL` com o cliente e modelo indicados.

As Etapas 3 → 4 → 5 rodam em loop fechado até o ARBITER aprovar, pedir correção de teste, escalar para o arquiteto ou o limite de tentativas ser atingido.

## Requisitos

- Go 1.22+
- CLIs `claude` e `agy` autenticadas via OAuth
- `npx` (Node.js) e `git`

## Uso Rápido

### 1. Preparar o ambiente — `-init`

```bash
go run . -init                 # ou: ./forgeloop -init
go run . -init -force-skills   # reinstala/atualiza todas as skills
```

O `-init`:

1. verifica no `$PATH` os binários configurados (`claude`, `agy`, `npx`, `git`) e mostra `[OK]` ou `[AUSENTE]`;
2. coleta as skills únicas das variáveis `FORGE_*_SKILL` (`.forge/config/models.env` / ambiente);
3. instala as ausentes com `npx skills add <skill> -g -y` (todas, com `-force-skills`). Uma skill conta como presente se existir uma pasta com o nome dela em `skills/`, `.agents/skills/`, `~/.claude/skills/` ou `~/.antigravity/skills/`, ou se o `skills-lock.json` a listar como origem;
4. cria `.forge/{agents,config,state}/`, `<docs>/demands/`, `<docs>/logs/` e `<docs>/references/` e grava o próprio log em `<docs>/logs/`.

Termina com código `1` se faltar algum binário ou se a instalação de alguma skill falhar. Toda execução normal também faz um *pre-flight* rápido e avisa (sem bloquear) se alguma skill configurada não estiver instalada.

### 2. Onboarding de projeto existente — `-setup` (Etapa 0)

```bash
go run . -setup
```

Cria `<docs>/PROJECT_CONTEXT.md`, `<docs>/demands/INDEX.md`, `<docs>/demands/`, `<docs>/logs/` e `<docs>/references/`.

### Detecção Automática e Personalização de Stack

Durante o `-setup`, o `forgeloop` inspeciona deterministicamente marcadores na raiz do projeto:
- `go.mod` → Go (`go test -v -race ./...`, `go vet ./...` e `golangci-lint run` se existir `.golangci.yml` ou `.golangci.yaml`);
- `Gemfile` com `gem 'rails'` → Ruby on Rails (`bundle exec rspec` se existir o diretório `spec/`, senão `bin/rails test`, além de `bundle exec rubocop`);
- `package.json` → Node.js ou TypeScript + Node.js (se existir `tsconfig.json` ou `typescript` em dependencies/devDependencies; `npm run lint` se a chave `lint` estiver em `scripts`).

O agente `ONBOARDER` confirma ou ajusta a stack identificada, e a configuração é gravada em `.forge/config/project.env`:

```env
FORGE_STACK_NAME="Go"
FORGE_TEST_CMD="go test -v -race ./..."
FORGE_TYPE_CMD="go vet ./..."
FORGE_LINT_CMD=""
```

Ao iniciar uma demanda com `-task`, o `forgeloop` exibe a stack salva como primeira opção:

```text
Qual stack deve ser utilizada nesta execução?
  [1] Projeto Atual (Auto-detectado): Go
  [2] Ruby on Rails
  [3] Go
  [4] TypeScript + Node.js
  [5] Personalizada
Escolha uma opção (1-5) [Enter = 1]: 
```

Pressionar Enter seleciona a opção `1`. Se `.forge/config/project.env` estiver ausente ou incompleto, um aviso é exibido e o menu manual (opções 1 a 4) é apresentado como fallback. Você também pode editar `.forge/config/project.env` manualmente para customizar comandos ou definir outras stacks (como Python, Java ou Makefile).

### 3. Iniciar nova demanda — `-task` (Etapas 1 a 5)

```bash
go run . -task "Implementar autenticação JWT com refresh token"
```

Enriqueça a demanda com material de `<docs>/references/` usando `-ref` (repetível). Cada arquivo é validado antes de tudo e entregue ao ARCHITECT como `@caminho`, então é lido antes do planejamento:

```bash
go run . -task "Implementar autenticação JWT com refresh token" \
  -ref auth-spec.md -ref docs/references/openapi.yaml
```

`-ref` aceita um caminho (relativo ao diretório atual ou absoluto) ou apenas o nome de um arquivo dentro de `<docs>/references/` (acompanha `-docs-dir` / `FORGE_DOCS_DIR`). Arquivo inexistente aborta antes de qualquer agente rodar. `-ref` só afeta a Etapa 1, então é ignorado (com aviso) ao iniciar com `-stage 2` ou superior.

Comece de uma Etapa posterior com `-stage` (`-forge` continua funcionando como alias de compatibilidade):

```bash
go run . -stage 2    # a partir de TEMPER (reaproveita .forge/state/PLAN.md / testes existentes)
go run . -stage 3    # a partir de STRIKE
```

### 4. Testar combinações de modelos

Edite `.forge/config/models.env` ou sobrescreva via variáveis de ambiente (veja detalhes completos na seção [Personalização de Modelos](#personalização-de-modelos)):

```bash
FORGE_BUILDER_MODEL="GPT-OSS 120B (Medium)" go run . -task "Adicionar paginação por cursor"
```

Cada agente tem `FORGE_<AGENTE>_BIN`, `_MODEL` e `_ARGS` (flags extras do CLI, ex.: permissões necessárias no modo headless). `--dangerously-skip-permissions` desativa os prompts de permissão: use apenas em repositório confiável.

## Personalização de Modelos

Os clientes de CLI, modelos e parâmetros de cada agente são centralizados e personalizáveis no arquivo:

📁 **`.forge/config/models.env`**

Esse arquivo é lido automaticamente na inicialização do `forgeloop` e pode ser versionado no repositório com as preferências do projeto.

### Variáveis de Configuração por Agente

Cada agente possui variáveis dedicadas para configurar o cliente de CLI (`_BIN`), o modelo (`_MODEL`), argumentos extras de execução (`_ARGS`) e, quando aplicável, a skill utilizada (`_SKILL`):

| Etapa | Agente | Cliente / Binário | Modelo | Argumentos Extras | Skill |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 0 | `ONBOARDER` | `FORGE_ONBOARDER_BIN` | `FORGE_ONBOARDER_MODEL` | `FORGE_ONBOARDER_ARGS` | — |
| 1 | `ARCHITECT` | `FORGE_ARCHITECT_BIN` | `FORGE_ARCHITECT_MODEL` | `FORGE_ARCHITECT_ARGS` | `FORGE_ARCHITECT_SKILL` |
| 2 | `SKEPTIC` | `FORGE_SKEPTIC_BIN` | `FORGE_SKEPTIC_MODEL` | `FORGE_SKEPTIC_ARGS` | `FORGE_SKEPTIC_SKILL` |
| 3 | `BUILDER` | `FORGE_BUILDER_BIN` | `FORGE_BUILDER_MODEL` | `FORGE_BUILDER_ARGS` | — |
| 4B | `SENTINEL` | `FORGE_SENTINEL_BIN` | `FORGE_SENTINEL_MODEL` | `FORGE_SENTINEL_ARGS` | `FORGE_SENTINEL_SKILL` |
| 5 | `ARBITER` | `FORGE_ARBITER_BIN` | `FORGE_ARBITER_MODEL` | `FORGE_ARBITER_ARGS` | — |

Além dos agentes, o `.forge/config/models.env` controla variáveis globais do fluxo:
- `FORGE_MAX_ATTEMPTS`: Limite de tentativas no loop de correção (padrão: `3`).
- `FORGE_DEFAULT_DOCS_DIR`: Diretório padrão de documentação (padrão: `docs`).

### Formas de Personalização

A precedência de configuração segue: **Flags CLI > Variáveis de Ambiente > `.forge/config/models.env` > Valores padrão em código**.

1. **Editar o arquivo `.forge/config/models.env`** (recomendado para configurações permanentes do projeto):
   ```env
   # Exemplo: ajustar modelos da Etapa 3 e Etapa 5
   FORGE_BUILDER_MODEL="Gemini 3.8 Flash (High)"
   FORGE_ARBITER_MODEL="Gemini 3.8 Flash (High)"
   ```

2. **Sobrescrever via variáveis de ambiente** (ideal para testes pontuais sem alterar arquivos):
   ```bash
   FORGE_BUILDER_MODEL="GPT-OSS 120B (Medium)" go run . -task "Adicionar paginação por cursor"
   ```

3. **Sobrescrever via flags do CLI**:
   ```bash
   go run . -builder-model "Gemini 3.8 Flash (High)" -arbiter-model "Gemini 3.8 Flash (High)" -task "Adicionar paginação por cursor"
   ```

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
go test -v -race ./... && go vet ./... && go build -o forgeloop .
```

## Licença

Distribuído sob a **Apache License 2.0**. Veja `LICENSE`.
