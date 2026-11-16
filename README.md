[Português (Brasil)](README.pt-BR.md)

# forgeloop

A spec-driven **Loop Engineering** orchestrator with strict TDD and automated security auditing. Written in Go with **zero external dependencies** (standard library only) and designed to run 100% on OAuth-authenticated CLI sessions (`claude` and `agy`) — no API keys.

## One Forge, Six Stages

**The Forge (`forgeloop`)** is the complete cycle. Each internal phase is a **Stage**:

| Stage | Name | Agent | What it does |
| :--- | :--- | :--- | :--- |
| 0 | `DISCOVERY` | `ONBOARDER` | Maps an existing project and consolidates its documentation into `<docs>/PROJECT_CONTEXT.md`. |
| 1 | `BLUEPRINT` | `ARCHITECT` | Incremental planning (`obra/superpowers`) and automated tests in `RED` state. |
| 2 | `TEMPER` | `SKEPTIC` | Cold critical review and global invariants (`google/elephant-goldfish-model`). |
| 3 | `STRIKE` | `BUILDER` | Implementation in a sanitized (*stateless*) context. |
| 4 | `QUENCH` | `SENTINEL` | Deterministic stack suite (4A) + AppSec audit of the `git diff` (4B, `cloudflare/security-audit-skill`). |
| 5 | `ASSAY` | `ARBITER` | Semantic judgment, token accounting and *Circuit Breaker* (`max-attempts`). |

Stages 3 → 4 → 5 run as a closed loop until the ARBITER approves, asks for a test amendment, escalates to the architect, or the attempt limit trips.

## Requirements

- Go 1.22+
- `claude` and `agy` CLIs, logged in via OAuth
- `npx` (Node.js) and `git`

## Quick Start

### 1. Prepare the environment — `-init`

```bash
go run forgeloop.go -init            # or: ./forgeloop -init
go run forgeloop.go -init -force-skills   # reinstall/update every skill
```

`-init`:

1. checks that the configured binaries (`claude`, `agy`, `npx`, `git`) are on your `$PATH` (`[OK]` / `[AUSENTE]`);
2. collects the unique skills from the `FORGE_*_SKILL` variables (`.forge/config/models.env` / environment);
3. installs the missing ones with `npx skills add <skill> -g -y` (all of them with `-force-skills`). A skill counts as present if a folder with its name exists in `skills/`, `.agents/skills/`, `~/.claude/skills/` or `~/.antigravity/skills/`, or if `skills-lock.json` lists it as a source;
4. creates `.forge/{agents,config,state}/`, `<docs>/demands/`, `<docs>/logs/` and `<docs>/references/` and writes its own log to `<docs>/logs/`.

It exits with code `1` if a binary is missing or a skill failed to install. Every normal run also performs a quick pre-flight check and warns (without blocking) if a configured skill is not installed.

### 2. Onboard an existing project — `-setup` (Stage 0)

```bash
go run forgeloop.go -setup
```

Creates `<docs>/PROJECT_CONTEXT.md`, `<docs>/demands/INDEX.md`, `<docs>/demands/`, `<docs>/logs/` and `<docs>/references/`.

### 3. Run a new demand — `-task` (Stages 1 to 5)

```bash
go run forgeloop.go -task "Implement JWT authentication with refresh tokens"
```

Start from a later Stage with `-stage` (`-forge` is kept as a compatibility alias):

```bash
go run forgeloop.go -stage 2    # TEMPER onwards (reuses the existing .forge/state/PLAN.md / tests)
go run forgeloop.go -stage 3    # STRIKE onwards
```

### 4. Try different model combinations

Edit `.forge/config/models.env` or override with environment variables:

```bash
FORGE_BUILDER_MODEL="GPT-OSS 120B (Medium)" go run forgeloop.go -task "Add cursor pagination"
```

Each agent has `FORGE_<AGENT>_BIN`, `_MODEL` and `_ARGS` (extra CLI flags, e.g. permission flags needed for headless mode). `--dangerously-skip-permissions` disables permission prompts: use it only in trusted repositories.

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
└── TELEMETRY.md           # one row per run: tokens per stage/loop + consolidated total
```

Logs contain the prompt and the full response of every agent, the complete test/linter output, the generated review/security/decision reports, durations, token counts and the final exit code.

## Exit codes

| Code | Meaning |
| :--- | :--- |
| `0` | `APPROVE_MERGE` (or a successful `-init`/`-setup`) |
| `1` | Fatal error or circuit breaker tripped |
| `2` | Specification blocked by the SKEPTIC (Stage 2) |
| `3` | `AMEND_TEST` — a Stage 1 test needs fixing |
| `4` | `ESCALATE_ARCHITECT` |

## Tests

```bash
go test -v -race ./... && go vet ./... && go build -o forgeloop forgeloop.go
```

## License

Distributed under the **Apache License 2.0**. See `LICENSE`.
