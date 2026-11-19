[Português (Brasil)](README.pt-BR.md)

# forgeloop

A spec-driven **Loop Engineering** orchestrator with strict TDD and automated security auditing. Written in Go with **zero external dependencies** (standard library only).

## One Forge, Six Stages

**The Forge (`forgeloop`)** is the complete cycle. Each internal phase is a **Stage**:

| Stage | Name | Agent | Client | Default Model | What it does |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 0 | `DISCOVERY` | `ONBOARDER` | `claude` | `opus` | Maps an existing project and consolidates its documentation into `<docs>/PROJECT_CONTEXT.md`. |
| 1 | `BLUEPRINT` | `ARCHITECT` | `claude` | `opus` | Incremental planning (`obra/superpowers`) and automated tests in `RED` state. |
| 2 | `TEMPER` | `SKEPTIC` | `agy` | `Gemini 3.8 Flash (Medium)` | Cold critical review and global invariants (`google/elephant-goldfish-model`). |
| 3 | `STRIKE` | `BUILDER` | `agy` | `Gemini 3.8 Flash (High)` | Implementation in a sanitized (*stateless*) context. |
| 4 | `QUENCH` | `SENTINEL` | `agy` *(4B)* | `Gemini 3.8 Flash (Medium)` *(4B)* | Deterministic stack suite (4A) + AppSec audit of the `git diff` (4B, `cloudflare/security-audit-skill`). |
| 5 | `ASSAY` | `ARBITER` | `agy` | `Gemini 3.8 Flash (High)` | Semantic judgment, token accounting and *Circuit Breaker* (`max-attempts`). |

> **Note:** In Stage 4 (`QUENCH`), phase 4A runs deterministic local stack tests and linters (no LLM), while phase 4B executes the AppSec audit with the `SENTINEL` agent using the indicated client and model.

Stages 3 → 4 → 5 run as a closed loop until the ARBITER approves, asks for a test amendment, escalates to the architect, or the attempt limit trips.

## Requirements

- Go 1.22+
- `claude` and `agy` CLIs
- `npx` (Node.js) and `git`

## Quick Start

### 1. Prepare the environment — `-init`

```bash
go run . -init            # or: ./forgeloop -init
go run . -init -force-skills   # reinstall/update every skill
```

`-init`:

1. checks that the configured binaries (`claude`, `agy`, `npx`, `git`) are on your `$PATH` (`[OK]` / `[AUSENTE]`);
2. collects the unique skills from the `FORGE_*_SKILL` variables (`.forge/config/models.env` / environment);
3. installs the missing ones with `npx skills add <skill> -g -y` (all of them with `-force-skills`). A skill counts as present if a folder with its name exists in `skills/`, `.agents/skills/`, `~/.claude/skills/` or `~/.antigravity/skills/`, or if `skills-lock.json` lists it as a source;
4. creates `.forge/{agents,config,state}/`, `<docs>/demands/`, `<docs>/logs/` and `<docs>/references/` and writes its own log to `<docs>/logs/`.

It exits with code `1` if a binary is missing or a skill failed to install. Every normal run also performs a quick pre-flight check and warns (without blocking) if a configured skill is not installed.

### 2. Onboard an existing project — `-setup` (Stage 0)

```bash
go run . -setup
```

Creates `<docs>/PROJECT_CONTEXT.md`, `<docs>/demands/INDEX.md`, `<docs>/demands/`, `<docs>/logs/` and `<docs>/references/`.

### Auto-Detected & Custom Stacks

When running `-setup`, `forgeloop` deterministically inspects markers in the project root:
- `go.mod` → Go (`go test -v -race ./...`, `go vet ./...`, and `golangci-lint run` if `.golangci.yml` or `.golangci.yaml` exists);
- `Gemfile` with `gem 'rails'` → Ruby on Rails (`bundle exec rspec` if `spec/` directory exists, otherwise `bin/rails test`, plus `bundle exec rubocop`);
- `package.json` → Node.js or TypeScript + Node.js (if `tsconfig.json` exists or `typescript` is in dependencies/devDependencies; `npm run lint` if `lint` is defined in `scripts`).

During `-setup`, the detected stack is shown and you are asked to confirm it before anything is written:

```text
[STACK] Detectada: Go (testes: go test -v -race ./...)
Confirmar esta stack como padrão do projeto? [S/n] (ou digite 'p' para personalizar): 
```

Enter, `S` or `s` confirms the detected stack; `p` (or `n`) opens the stack menu so you can pick a different stack; any other answer aborts the setup without writing. When no stack is detected, no question is asked. The `ONBOARDER` agent then confirms or refines the stack, and the configuration is saved to `.forge/config/project.env`:

```env
FORGE_STACK_NAME="Go"
FORGE_TEST_CMD="go test -v -race ./..."
FORGE_TYPE_CMD="go vet ./..."
FORGE_LINT_CMD=""
```

When you start a demand with `-task`, `-stage` or `-demand`, `forgeloop` uses the stack saved in `.forge/config/project.env` silently — no stack confirmation is required on each run:

```text
[INFO] Stack do projeto: Go (via project.env)
```

To change the stack, pass `-select-stack`: it opens the menu even when `project.env` is valid and saves your choice. The menu also opens automatically when `project.env` is missing or incomplete.

```bash
go run . -task "Implement JWT authentication" -select-stack
```

The `-select-stack` menu looks like this:

```text
Qual stack deve ser utilizada nesta execução?
  [1] Projeto Atual (Auto-detectado): Go
  [2] Ruby on Rails
  [3] Go
  [4] TypeScript + Node.js
  [5] Personalizada
Escolha uma opção (1-5) [Enter = 1]: 
```

Pressing Enter defaults to option `1`. If `.forge/config/project.env` is missing or incomplete, a warning is displayed and the manual menu (options 1 to 4) is presented as fallback, and the chosen stack is saved. You can also edit `.forge/config/project.env` manually to override commands or configure custom stacks (such as Python, Java, or Makefile).

### 3. Run a new demand — `-task` (Stages 1 to 5)

```bash
go run . -task "Implement JWT authentication with refresh tokens"
```

Enrich the demand with material from `<docs>/references/` using `-ref` (repeatable). Each file is validated up front and handed to the ARCHITECT as `@path`, so it is read before planning:

```bash
go run . -task "Implement JWT authentication with refresh tokens" \
  -ref auth-spec.md -ref docs/references/openapi.yaml
```

`-ref` accepts either a path (relative to the current directory or absolute) or just a file name inside `<docs>/references/` (follows `-docs-dir` / `FORGE_DOCS_DIR`). A missing file aborts before any agent runs. `-ref` only affects Stage 1, so it is ignored (with a warning) when you start with `-stage 2` or later.

Start from a later Stage with `-stage` (`-forge` is kept as a compatibility alias):

```bash
go run . -stage 2    # TEMPER onwards (reuses existing .forge/state/PLAN.md / tests)
go run . -stage 3    # STRIKE onwards
```

### Resuming Execution & Connecting Existing Demands

You can resume interrupted runs or connect existing demands:

- `-resume`: Restores the exact execution state saved in `.forge/state/RESUME.env` when a run is paused (e.g. on quota limits).
- `-stage <1..5>`: Starts execution at any stage (1 to 5).
- `-demand <file.md>`: Connects an existing demand from `<docsDir>/demands/` (e.g. `-demand 2026-11-17-feature.md`) without creating a duplicate record.

### 4. Try different model combinations

Edit `.forge/config/models.env` or override with environment variables (see full details in the [Model Customization](#model-customization) section):

```bash
FORGE_BUILDER_MODEL="GPT-OSS 120B (Medium)" go run . -task "Add cursor pagination"
```

Each agent has `FORGE_<AGENT>_BIN`, `_MODEL` and `_ARGS` (extra CLI flags, e.g. permission flags needed for headless mode). `--dangerously-skip-permissions` disables permission prompts: use it only in trusted repositories.

## Model Customization

Default CLI clients, models, and parameters for each agent are centralized and customized in:

📁 **`.forge/config/models.env`**

This file is automatically loaded on `forgeloop` startup and can be committed to the repository to share project defaults.

### Configuration Variables per Agent

Each agent has dedicated environment variables to configure its CLI binary (`_BIN`), model (`_MODEL`), extra CLI arguments (`_ARGS`), and skill (`_SKILL` when applicable):

| Stage | Agent | Client / Binary | Model | Extra Arguments | Skill |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 0 | `ONBOARDER` | `FORGE_ONBOARDER_BIN` | `FORGE_ONBOARDER_MODEL` | `FORGE_ONBOARDER_ARGS` | — |
| 1 | `ARCHITECT` | `FORGE_ARCHITECT_BIN` | `FORGE_ARCHITECT_MODEL` | `FORGE_ARCHITECT_ARGS` | `FORGE_ARCHITECT_SKILL` |
| 2 | `SKEPTIC` | `FORGE_SKEPTIC_BIN` | `FORGE_SKEPTIC_MODEL` | `FORGE_SKEPTIC_ARGS` | `FORGE_SKEPTIC_SKILL` |
| 3 | `BUILDER` | `FORGE_BUILDER_BIN` | `FORGE_BUILDER_MODEL` | `FORGE_BUILDER_ARGS` | — |
| 4B | `SENTINEL` | `FORGE_SENTINEL_BIN` | `FORGE_SENTINEL_MODEL` | `FORGE_SENTINEL_ARGS` | `FORGE_SENTINEL_SKILL` |
| 5 | `ARBITER` | `FORGE_ARBITER_BIN` | `FORGE_ARBITER_MODEL` | `FORGE_ARBITER_ARGS` | — |

In addition to agents, `.forge/config/models.env` manages global loop settings:
- `FORGE_MAX_ATTEMPTS`: Maximum attempts in the correction loop before circuit breaker trips (default: `3`).
- `FORGE_DEFAULT_DOCS_DIR`: Default documentation directory (default: `docs`).

### Authentication & OmniRoute Fallback

By default (`FORGE_AUTH_MODE="api_key"`, set in `.forge/config/models.env`), agents authenticate with provider API keys that live **exclusively** in `.forge/config/project.env` (git-ignored, written with `0600` permissions), isolating costs per project:
- `ANTHROPIC_API_KEY`: Anthropic key (`FORGE_API_KEY` is accepted as an alias).
- `OPENAI_API_KEY`: OpenAI key.
- `GEMINI_API_KEY`: Google Gemini key.

All configured keys are injected into every agent subprocess. The `api_key` pre-flight check requires at least one of them (environment variables take precedence over the file). `forgeloop` prints a warning if any API key is found in the versioned `.forge/config/models.env`.

When token or quota limits are reached, `forgeloop` can transparently route requests through `OmniRoute`:
- `FORGE_OMNIROUTE_ENABLED`: Enables automatic OmniRoute fallback when set to `true` (default: `false`).
- `FORGE_OMNIROUTE_URL`: OmniRoute server address (default: `http://localhost:20128`).
- `FORGE_OMNIROUTE_MODEL`: Model routed via OmniRoute (default: `claude-sonnet-4-6`).

### Cross-Auditing (Multi-LLM)

Use up to 5 different models across the stages. It is strongly recommended to use at least 2 distinct providers — e.g. Anthropic for the BUILDER and OpenAI/Gemini for the ARBITER — to avoid confirmation bias when auditing the code. Configure each stage through `FORGE_<AGENT>_BIN` / `FORGE_<AGENT>_MODEL` in `.forge/config/models.env`, and store the matching provider keys in `.forge/config/project.env`.

### How to Customize

Configuration precedence is: **CLI Flags > Environment Variables > `.forge/config/models.env` > Hardcoded defaults**.

1. **Edit `.forge/config/models.env` directly** (recommended for persistent project defaults):
   ```env
   # Example: customize Stage 3 and Stage 5 models
   FORGE_BUILDER_MODEL="Gemini 3.8 Flash (High)"
   FORGE_ARBITER_MODEL="Gemini 3.8 Flash (High)"
   ```

2. **Override via environment variables** (ideal for a single run without touching config files):
   ```bash
   FORGE_BUILDER_MODEL="GPT-OSS 120B (Medium)" go run . -task "Add cursor pagination"
   ```

3. **Override via CLI flags**:
   ```bash
   go run . -builder-model "Gemini 3.8 Flash (High)" -arbiter-model "Gemini 3.8 Flash (High)" -task "Add cursor pagination"
   ```

## Output layout

```text
<docs>/
├── PROJECT_CONTEXT.md     # project context (Stage 0)
├── demands/               # one file per demand, linked to its execution log
│   └── INDEX.md           # | Data | Stack | Demanda | Arquivo | Status |
├── logs/                  # one full log per run: YYYY-MM-DD_HHMMSS-<slug>.log
└── references/            # reference material (created by -init / -setup)
.forge/
├── agents/                # agent contracts (00_ONBOARDER.md ... 05_ARBITER.md)
├── config/                # models.env (versioned), project.env (git-ignored)
├── state/                 # transient artifacts: PLAN, TEST_CASES, DECISION, ... (git-ignored)
└── telemetry/             # CSV metrics (header written on first use, then append-only)
    ├── TELEMETRY.md       # .forge/telemetry/TELEMETRY.md: KPIs + run history, regenerated from the CSVs at the end of every Forge run
    ├── runs.csv           # one row per run: attempts, tokens in/out/total, final status
    └── events.csv         # one row per stage/agent execution: tokens, duration, attempt (joins runs.csv by RunID)
```

Logs contain the prompt and the full response of every agent, the complete test/linter output, the generated review/security/decision reports, durations, token counts and the final exit code.

## Exit codes

When the SKEPTIC answers `SPEC_STATUS: REVISION_REQUIRED`, the engine automatically goes back to Stage 1 (the ARCHITECT receives the review and fixes the plan and tests) until `-max-attempts` is reached; exit code `2` only happens when those attempts are exhausted.

| Code | Meaning |
| :--- | :--- |
| `0` | `APPROVE_MERGE` (or a successful `-init`/`-setup`) |
| `1` | Fatal error or circuit breaker tripped |
| `2` | Specification blocked by the SKEPTIC (Stage 2) after exhausting `-max-attempts` |
| `3` | `AMEND_TEST` — a Stage 1 test needs fixing |
| `4` | `ESCALATE_ARCHITECT` |
| `5` | Quota or token limit reached (state saved in `RESUME.env`, resumable with `-resume`) |

## Tests

```bash
go test -v -race ./... && go vet ./... && go build -o forgeloop .
```

## License

Distributed under the **Apache License 2.0**. See `LICENSE`.
