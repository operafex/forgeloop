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
- CLIs `claude` e `agy`
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

Durante o `-setup`, a stack detectada é exibida e a confirmação é solicitada antes de qualquer gravação:

```text
[STACK] Detectada: Go (testes: go test -v -race ./...)
Confirmar esta stack como padrão do projeto? [S/n] (ou digite 'p' para personalizar): 
```

Enter, `S` ou `s` confirma a stack detectada; `p` (ou `n`) abre o menu de stacks para escolher outra; qualquer outra resposta aborta o setup sem gravar. Quando nenhuma stack é detectada, não há pergunta. Em seguida, o agente `ONBOARDER` confirma ou ajusta a stack, e a configuração é gravada em `.forge/config/project.env`:

```env
FORGE_STACK_NAME="Go"
FORGE_TEST_CMD="go test -v -race ./..."
FORGE_TYPE_CMD="go vet ./..."
FORGE_LINT_CMD=""
```

Ao iniciar uma demanda com `-task`, `-stage` ou `-demand`, o `forgeloop` usa a stack salva em `.forge/config/project.env` em silêncio — o fluxo não exige confirmação de stack a cada rodada:

```text
[INFO] Stack do projeto: Go (via project.env)
```

Para trocar a stack, use `-select-stack`: ele abre o menu mesmo com um `project.env` válido e salva a escolha. O menu também abre automaticamente quando o `project.env` está ausente ou incompleto.

```bash
go run . -task "Implementar autenticação JWT" -select-stack
```

O menu do `-select-stack` é assim:

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

### Retomada de Execução e Vinculação de Demandas

Você pode retomar execuções interrompidas ou vincular demandas existentes:

- `-resume`: Restaura o estado exato gravado em `.forge/state/RESUME.env` após interrupções (como limites de cota).
- `-stage <1..5>`: Inicia a execução diretamente em qualquer etapa (1 a 5).
- `-demand <arquivo.md>`: Vincula uma demanda existente em `<docsDir>/demands/` (ex.: `-demand 2026-11-17-feature.md`) sem gerar duplicações no índice.

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

### Autenticação e Fallback OmniRoute

Por padrão (`FORGE_AUTH_MODE="api_key"`, definido em `.forge/config/models.env`), os agentes se autenticam com chaves de API dos provedores que residem **exclusivamente** em `.forge/config/project.env` (ignorado pelo git, gravado com permissão `0600`), isolando os custos por projeto:
- `ANTHROPIC_API_KEY`: chave da Anthropic (`FORGE_API_KEY` é aceita como alias).
- `OPENAI_API_KEY`: chave da OpenAI.
- `GEMINI_API_KEY`: chave do Google Gemini.

Todas as chaves configuradas são injetadas no subprocesso de cada agente. A validação pre-flight do modo `api_key` exige ao menos uma delas (variáveis do ambiente têm precedência sobre o arquivo). O `forgeloop` emite um aviso se encontrar chaves de API no `.forge/config/models.env` versionado.

Em caso de limite de cota ou rate limit, o `forgeloop` realiza fallback transparente para o `OmniRoute`:
- `FORGE_OMNIROUTE_ENABLED`: Ativa o fallback automático via OmniRoute quando `true` (padrão: `false`).
- `FORGE_OMNIROUTE_URL`: Endereço do servidor OmniRoute (padrão: `http://localhost:20128`).
- `FORGE_OMNIROUTE_MODEL`: Modelo roteado via OmniRoute (padrão: `claude-sonnet-4-6`).

### Cross-Auditing (Multi-LLM)

Utilize até 5 modelos diferentes nas etapas. É fortemente recomendado usar no mínimo 2 provedores distintos — ex.: Anthropic no BUILDER e OpenAI/Gemini no ARBITER — para evitar viés de confirmação na auditoria do código. Configure cada etapa via `FORGE_<AGENTE>_BIN` / `FORGE_<AGENTE>_MODEL` no `.forge/config/models.env` e guarde as chaves dos provedores correspondentes no `.forge/config/project.env`.

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
    ├── TELEMETRY.md       # .forge/telemetry/TELEMETRY.md: KPIs + histórico, regenerado a partir dos CSVs ao final de cada Forja
    ├── runs.csv           # uma linha por execução: tentativas, tokens in/out/total, status final
    └── events.csv         # uma linha por etapa/agente: tokens, duração, tentativa (junta com runs.csv via RunID)
```

Os logs trazem o prompt e a resposta completa de cada agente, a saída integral de testes/linters, os relatórios de revisão/segurança/decisão, durações, contagem de tokens e o código de saída final.

## Códigos de saída

Quando o SKEPTIC responde `SPEC_STATUS: REVISION_REQUIRED`, o motor volta sozinho à Etapa 1 (o ARCHITECT recebe o parecer e corrige plano e testes) até `-max-attempts`; o código `2` só ocorre quando essas tentativas se esgotam.

| Código | Significado |
| :--- | :--- |
| `0` | `APPROVE_MERGE` (ou `-init`/`-setup` bem-sucedido) |
| `1` | Erro fatal ou circuit breaker acionado |
| `2` | Especificação bloqueada pelo SKEPTIC (Etapa 2) após esgotar `-max-attempts` |
| `3` | `AMEND_TEST` — teste da Etapa 1 precisa de correção |
| `4` | `ESCALATE_ARCHITECT` |
| `5` | Limite de cota ou tokens atingido (estado gravado em `RESUME.env`, retomável com `-resume`) |

## Testes

```bash
go test -v -race ./... && go vet ./... && go build -o forgeloop .
```

## Licença

Distribuído sob a **Apache License 2.0**. Veja `LICENSE`.
