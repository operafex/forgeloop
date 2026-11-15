# forgeloop

Orquestrador de **Loop Engineering** orientado a especificações, TDD estrito e auditoria de segurança automatizada, construído em Go sem dependências externas e projetado para operar 100% via sessões autenticadas por OAuth (`claude` e `agy`).

## Ciclo das Forges

- **Forge 0 — Discovery (`ONBOARDER`):** Mapeia projetos existentes e consolida a documentação em `/docs/PROJECT_CONTEXT.md`.
- **Forge 1 — Blueprint (`ARCHITECT`):** Planejamento incremental (`superpowers`) e criação de testes automatizados em estado `RED`.
- **Forge 2 — Temper (`SKEPTIC`):** Revisão crítica a frio e validação de invariantes globais (`elephant-goldfish-model`).
- **Forge 3 — Strike (`BUILDER`):** Implementação em contexto higienizado (*stateless*).
- **Forge 4 — Quench (`SENTINEL`):** Suíte determinística da stack (`4A`) + auditoria AppSec no `git diff` (`4B` via `cloudflare/security-audit-skill`).
- **Forge 5 — Assay (`ARBITER`):** Julgamento semântico, contabilização de tokens e *Circuit Breaker* (`max-attempts`).

## Uso Rápido

### 1. Onboarding de Projeto Existente (Forge 0)
```bash
go run forgeloop.go -setup

```

### 2. Iniciar Nova Demanda (Forges 1 a 5)

```bash
go run forgeloop.go -task "Implementar autenticação JWT com refresh token"

```

### 3. Testar Combinações de Modelos

Edite `.forge/models.env` ou sobrescreva via variáveis de ambiente:

```bash
FORGE_BUILDER_MODEL="GPT-OSS 120B (Medium)" go run forgeloop.go -task "Adicionar paginação por cursor"

```

## Licença

Distribuído sob a **Apache License 2.0**. Veja `LICENSE` para mais detalhes.

