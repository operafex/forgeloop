# Demanda: Automatizar a detecção da stack, salvar isso nas configurações do projeto e atualizar os readmes

- **Data:** 2026-11-16 15:33
- **Stack:** Go
- **Bindings:** `F1:opus | F2:GPT-OSS 120B (Medium) | F3:Gemini 3.8 Flash (High) | F4B:Claude Sonnet 4.6 (Thinking) | F5:Claude Sonnet 4.6 (Thinking)`
- **Status:** `APPROVED`
- **Tentativas:** `1`
- **Tokens Totais:** `2042285`
- **Log de Execução:** [`2026-11-16_153320-automatizar-a-detecção-da-stack-salvar-isso-n.log`](../logs/2026-11-16_153320-automatizar-a-detecção-da-stack-salvar-isso-n.log)

## Descrição Original

Automatizar a detecção da stack, salvar isso nas configurações do projeto e atualizar os readmes

## Blueprint & Critérios de Aceite

> Plano completo: `.forge/state/PLAN.md` · Matriz: `.forge/state/TEST_CASES.md` · Base: `docs/references/onboarding-auto-stack.md`

### Resumo técnico

- **Arquivo novo `stack.go`** (`package main`, stdlib, < 500 linhas). Concentra o código de stack. `selectStack` sai de `forgeloop.go` e vem para cá.
- **Detecção determinística no Go** (`detectStack(root) (StackConfig, bool)`), com prioridade `go.mod` → `Gemfile` com rails → `package.json` (TypeScript se houver `tsconfig.json` ou dependência `typescript`; lint só se existir `scripts.lint`).
- **Persistência segura** em `.forge/config/project.env`:
  - `writeProjectEnv(path, updates)` faz merge e preserva as outras linhas.
  - Chaves são validadas com `^[A-Z][A-Z0-9_]*$`. Valores com `\n`, `\r` ou `"` são rejeitados.
  - Isso corrige o `os.WriteFile` que apagava o arquivo inteiro no `-setup`.
  - Chaves gravadas: `FORGE_STACK_NAME`, `FORGE_TEST_CMD`, `FORGE_TYPE_CMD` e `FORGE_LINT_CMD`.
- **`-setup`:** `setupProjectEnv(envPath, root, docsDir, existingDocs)` grava as chaves de docs e a stack detectada. O ONBOARDER valida ou corrige a stack (contrato `00_ONBOARDER.md` atualizado).
- **`-task`:** `selectStack(reader, out, envPath) (StackConfig, error)` lê o `project.env` com `loadProjectStack`.
  - Com stack salva: mostra `[1] Projeto Atual (Auto-detectado): <nome>` (Enter = 1) e as opções 2–5.
  - Sem stack salva: mostra `[AVISO] … (motivo: …)` e o menu manual 1–4.
  - Não chama mais `fatal` dentro da função.
- **READMEs EN/PT-BR:** nova seção depois do `-setup`. Os comandos passam a ser `go run .` e `go build -o forgeloop .`, porque o build de arquivo único não compilaria `stack.go`.

### Tasks

1. `detectStack` + presets (`goStack`, `railsStack`, `nodeStack`, `tsStack`)
2. `loadProjectStack` + `parseEnvContent` + `errProjectStackIncomplete`
3. `stackEnvValues` + `writeProjectEnv` (merge validado)
4. `setupProjectEnv` + integração em `runProjectSetup`
5. Novo `selectStack` com opção auto-detectada e fallback
6. Contrato do ONBOARDER
7. READMEs e comandos de build multi-arquivo
8. Validação: `gofmt`, `go vet ./...`, `go test -v -race -count=1 ./...`, `go build -o forgeloop .`

### Critérios de aceite

- [ ] 52 testes novos (57 cenários) em `stack_test.go`, `stack_env_test.go` e `stack_menu_test.go` passam, sem regressão nos existentes.
- [ ] O `-setup` em um repositório Go grava a stack em `project.env` sem perder `FORGE_DOCS_DIR`/`FORGE_EXISTING_DOCS` nem chaves customizadas.
- [ ] O `-task` pré-seleciona a stack salva (Enter = 1). Se o arquivo faltar ou estiver incompleto, mostra o aviso com o motivo e o menu manual.
- [ ] Os READMEs documentam o recurso e não citam mais `go run forgeloop.go`.
- [ ] Nenhuma dependência externa nova.

### Estado RED (2026-11-16)

`go test ./...` falha na compilação apenas por implementação ausente: `undefined: detectStack`, `loadProjectStack`, `errProjectStackIncomplete`, `stackEnvValues`, `writeProjectEnv`, `setupProjectEnv` e a assinatura nova de `selectStack`. Os 3 arquivos de teste passam no `gofmt`.
