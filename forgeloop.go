package main

import (
  "bufio"
  "bytes"
  "encoding/json"
  "flag"
  "fmt"
  "io"
  "os"
  "os/exec"
  "path/filepath"
  "regexp"
  "strconv"
  "strings"
  "time"
  "unicode"
)

type StackConfig struct {
  Name    string
  TestCmd string
  TypeCmd string
  LintCmd string
}

type AgentBinding struct {
  Forge string
  Alias string
  Bin   string
  Model string
  Skill string
}

type TokenStats struct {
  Input     int
  Output    int
  Total     int
  Estimated bool
}

func (t *TokenStats) Add(other TokenStats) {
  t.Input += other.Input
  t.Output += other.Output
  t.Total += other.Total
  if other.Estimated {
    t.Estimated = true
  }
}

func (t TokenStats) Format() string {
  prefix := ""
  if t.Estimated {
    prefix = "~"
  }
  return fmt.Sprintf("%s%d in / %s%d out (%s%d total)", prefix, t.Input, prefix, t.Output, prefix, t.Total)
}

func main() {
  loadEnvFile(".forge/models.env")
  loadEnvFile(".forge/project.env")

  setupMode := flag.Bool("setup", false, "Forge 0 (Discovery): analisa projeto existente e mapeia documentação")
  taskFlag := flag.String("task", "", "Descrição da nova demanda (salva automaticamente em /docs/demands/)")
  startForge := flag.Int("forge", 1, "Forge inicial: 1 (Completa), 2 (Temper/Skeptic), 3 (Strike/Builder)")
  maxAttempts := flag.Int("max-attempts", getEnvInt("FORGE_MAX_ATTEMPTS", 3), "Limite máximo de tentativas no loop (Circuit Breaker)")
  docsDirFlag := flag.String("docs-dir", getEnv("FORGE_DOCS_DIR", getEnv("FORGE_DEFAULT_DOCS_DIR", "docs")), "Diretório de documentação")
  jsonMetrics := flag.Bool("json-metrics", true, "Extrai contagem exata de tokens via --output-format json quando suportado")

  // Bindings desacoplados: troque via .forge/models.env ou flags CLI
  onbBin := flag.String("onboarder-bin", getEnv("FORGE_ONBOARDER_BIN", "claude"), "Binário da Forge 0 (ONBOARDER)")
  onbModel := flag.String("onboarder-model", getEnv("FORGE_ONBOARDER_MODEL", "opus"), "Modelo da Forge 0 (ONBOARDER)")

  archBin := flag.String("architect-bin", getEnv("FORGE_ARCHITECT_BIN", "claude"), "Binário da Forge 1 (ARCHITECT)")
  archModel := flag.String("architect-model", getEnv("FORGE_ARCHITECT_MODEL", "opus"), "Modelo da Forge 1 (ARCHITECT)")
  archSkill := flag.String("architect-skill", getEnv("FORGE_ARCHITECT_SKILL", "superpowers"), "Skill da Forge 1 (ARCHITECT)")

  skepBin := flag.String("skeptic-bin", getEnv("FORGE_SKEPTIC_BIN", "agy"), "Binário da Forge 2 (SKEPTIC)")
  skepModel := flag.String("skeptic-model", getEnv("FORGE_SKEPTIC_MODEL", "GPT-OSS 120B (Medium)"), "Modelo da Forge 2 (SKEPTIC)")
  skepSkill := flag.String("skeptic-skill", getEnv("FORGE_SKEPTIC_SKILL", "elephant-goldfish-model"), "Skill da Forge 2 (SKEPTIC)")

  buildBin := flag.String("builder-bin", getEnv("FORGE_BUILDER_BIN", "agy"), "Binário da Forge 3 (BUILDER)")
  buildModel := flag.String("builder-model", getEnv("FORGE_BUILDER_MODEL", "Gemini 3.8 Flash"), "Modelo da Forge 3 (BUILDER)")

  sentBin := flag.String("sentinel-bin", getEnv("FORGE_SENTINEL_BIN", "agy"), "Binário da Forge 4B (SENTINEL)")
  sentModel := flag.String("sentinel-model", getEnv("FORGE_SENTINEL_MODEL", "Claude Sonnet 4.6 (Thinking)"), "Modelo da Forge 4B (SENTINEL)")
  sentSkill := flag.String("sentinel-skill", getEnv("FORGE_SENTINEL_SKILL", "cloudflare/security-audit-skill"), "Skill da Forge 4B (SENTINEL)")

  arbBin := flag.String("arbiter-bin", getEnv("FORGE_ARBITER_BIN", "agy"), "Binário da Forge 5 (ARBITER)")
  arbModel := flag.String("arbiter-model", getEnv("FORGE_ARBITER_MODEL", "Claude Sonnet 4.6 (Thinking)"), "Modelo da Forge 5 (ARBITER)")

  flag.Parse()

  if err := os.MkdirAll(".forge", 0755); err != nil {
    fatal("Erro ao criar diretório .forge: %v", err)
  }

  onboarder := AgentBinding{Forge: "FORGE 0", Alias: "ONBOARDER", Bin: *onbBin, Model: *onbModel}
  architect := AgentBinding{Forge: "FORGE 1", Alias: "ARCHITECT", Bin: *archBin, Model: *archModel, Skill: *archSkill}
  skeptic := AgentBinding{Forge: "FORGE 2", Alias: "SKEPTIC", Bin: *skepBin, Model: *skepModel, Skill: *skepSkill}
  builder := AgentBinding{Forge: "FORGE 3", Alias: "BUILDER", Bin: *buildBin, Model: *buildModel}
  sentinel := AgentBinding{Forge: "FORGE 4B", Alias: "SENTINEL", Bin: *sentBin, Model: *sentModel, Skill: *sentSkill}
  arbiter := AgentBinding{Forge: "FORGE 5", Alias: "ARBITER", Bin: *arbBin, Model: *arbModel}

  reader := bufio.NewReader(os.Stdin)

  // =========================================================================
  // FORGE 0 — DISCOVERY (-setup): ANÁLISE DE PROJETO EXISTENTE
  // =========================================================================
  if *setupMode {
    runProjectSetup(reader, onboarder, *docsDirFlag, *jsonMetrics)
    return
  }

  stack := selectStack(reader)
  docsDir := strings.Trim(strings.TrimSpace(*docsDirFlag), "/")
  if docsDir == "" {
    docsDir = "docs"
  }

  task := strings.TrimSpace(*taskFlag)
  if *startForge == 1 && task == "" {
    fmt.Print("\nDescreva a nova demanda ou feature: ")
    input, _ := reader.ReadString('\n')
    task = strings.TrimSpace(input)
    if task == "" {
      fatal("A descrição da demanda não pode estar vazia na Forge 1.")
    }
  }

  modelsSummary := fmt.Sprintf("F1:%s | F2:%s | F3:%s | F4B:%s | F5:%s",
    architect.Model, skeptic.Model, builder.Model, sentinel.Model, arbiter.Model)

  demandFile := ""
  if task != "" {
    demandFile = registerDemandInDocs(docsDir, stack.Name, task, modelsSummary)
    fmt.Printf("\n[DOCS] Demanda registrada em: %s\n", demandFile)
  }

  var totalTokens TokenStats
  var breakdown []string

  printBanner("FORGELOOP INICIADO",
    fmt.Sprintf("Stack: %s | Docs: /%s | Max Tentativas: %d\nBindings: %s", stack.Name, docsDir, *maxAttempts, modelsSummary))

  // =========================================================================
  // FORGE 1 — BLUEPRINT: ARCHITECT (Spec-TDD-Planner / Superpowers)
  // =========================================================================
  if *startForge <= 1 {
    printBanner("FORGE 1: BLUEPRINT — ARCHITECT", fmt.Sprintf("Skill: %s | Modelo: %s (%s)", architect.Skill, architect.Model, architect.Bin))
    prompt := fmt.Sprintf(
      "Siga estritamente as instruções de @.forge/01_ARCHITECT.md aplicando a skill %s. "+
        "Diretório de documentação do projeto: %s/ (leia %s/PROJECT_CONTEXT.md se existir). "+
        "Stack selecionada: %s. Demanda: %s (arquivo de registro: %s). "+
        "Gere .forge/PLAN.md (dividido em Tasks atômicas), .forge/TEST_CASES.md, anexe o plano técnico em %s e crie os testes automatizados em estado RED.",
      architect.Skill, docsDir, docsDir, stack.Name, task, demandFile, demandFile,
    )
    stats, err := runAgent(architect, prompt, *jsonMetrics)
    totalTokens.Add(stats)
    breakdown = append(breakdown, fmt.Sprintf("F1(%s): %s", architect.Alias, stats.Format()))
    if err != nil {
      fatal("Falha na Forge 1 (ARCHITECT): %v", err)
    }
    fmt.Printf("[TOKENS %s - %s] %s\n", architect.Forge, architect.Alias, stats.Format())
  }

  // =========================================================================
  // FORGE 2 — TEMPER: SKEPTIC (Cold-Global-Lens / Elephant-Goldfish)
  // =========================================================================
  if *startForge <= 2 {
    printBanner("FORGE 2: TEMPER — SKEPTIC", fmt.Sprintf("Skill: %s | Modelo: %s (%s)", skeptic.Skill, skeptic.Model, skeptic.Bin))
    prompt := fmt.Sprintf(
      "Siga estritamente as instruções de @.forge/02_SKEPTIC.md aplicando a skill %s. "+
        "Diretório de documentação: %s/. Audite @.forge/PLAN.md, @.forge/TEST_CASES.md e os arquivos de teste recém-criados. "+
        "Grave seu veredito em .forge/SPEC_REVIEW.md.",
      skeptic.Skill, docsDir,
    )

    stats, err := runAgent(skeptic, prompt, *jsonMetrics)
    totalTokens.Add(stats)
    breakdown = append(breakdown, fmt.Sprintf("F2(%s): %s", skeptic.Alias, stats.Format()))
    if err != nil {
      fatal("Falha na Forge 2 (SKEPTIC): %v", err)
    }
    fmt.Printf("[TOKENS %s - %s] %s\n", skeptic.Forge, skeptic.Alias, stats.Format())

    reviewContent, err := readFile(".forge/SPEC_REVIEW.md")
    if err != nil {
      fatal("Não foi possível ler .forge/SPEC_REVIEW.md: %v", err)
    }

    if !strings.Contains(reviewContent, "SPEC_STATUS: APPROVED") {
      updateDemandStatus(demandFile, "SPEC_REVISION_REQUIRED", 0, totalTokens, breakdown)
      appendTelemetry(stack.Name, demandFile, modelsSummary, 0, totalTokens, "SPEC_REVISION_REQUIRED", "Bloqueado pelo SKEPTIC na Forge 2")
      printBanner("BLOQUEIO NA FORGE 2 (TEMPER)", "O SKEPTIC exigiu ajustes na especificação. Consulte .forge/SPEC_REVIEW.md e rode novamente a Forge 1.")
      os.Exit(2)
    }
    fmt.Println("\n[OK] Especificação aprovada (SPEC_STATUS: APPROVED). Entrando no ciclo Strike -> Quench -> Assay...")
  }

  // =========================================================================
  // LOOP FECHADO: FORGE 3 (STRIKE) -> FORGE 4 (QUENCH) -> FORGE 5 (ASSAY)
  // =========================================================================
  for attempt := 1; attempt <= *maxAttempts; attempt++ {
    // FORGE 3 — STRIKE: BUILDER (Contexto Limpo)
    printBanner(fmt.Sprintf("FORGE 3: STRIKE — BUILDER (Tentativa %d/%d)", attempt, *maxAttempts),
      fmt.Sprintf("Contexto Higienizado | Modelo: %s (%s)", builder.Model, builder.Bin))

    execPrompt := fmt.Sprintf(
      "Siga estritamente as instruções de @.forge/03_BUILDER.md. Diretório de docs: %s/. "+
        "Leia @.forge/PLAN.md e @.forge/TEST_CASES.md e implemente as Tasks no código de produção.",
      docsDir,
    )
    if attempt > 1 {
      execPrompt += " ATENÇÃO: Rodada de correção. Leia @.forge/DECISION.md, @.forge/VERIFY_REPORT.md e @.forge/SECURITY_REPORT.md e corrija todos os apontamentos."
    }

    bStats, err := runAgent(builder, execPrompt, *jsonMetrics)
    totalTokens.Add(bStats)
    breakdown = append(breakdown, fmt.Sprintf("F3#%d(%s): %s", attempt, builder.Alias, bStats.Format()))
    if err != nil {
      fmt.Printf("[AVISO] BUILDER retornou erro de processo: %v\n", err)
    }
    fmt.Printf("[TOKENS %s - %s] %s\n", builder.Forge, builder.Alias, bStats.Format())

    // FORGE 4A — QUENCH TESTS: Verificação Determinística
    printBanner(fmt.Sprintf("FORGE 4A: QUENCH TESTS (Tentativa %d/%d)", attempt, *maxAttempts),
      fmt.Sprintf("Runner Determinístico (%s)", stack.Name))
    checksPassed := runDeterministicChecks(stack)

    // FORGE 4B — QUENCH APPSEC: SENTINEL
    printBanner(fmt.Sprintf("FORGE 4B: QUENCH APPSEC — SENTINEL (Tentativa %d/%d)", attempt, *maxAttempts),
      fmt.Sprintf("Skill: %s | Modelo: %s", sentinel.Skill, sentinel.Model))

    secPrompt := fmt.Sprintf(
      "Siga estritamente as instruções de @.forge/04_SENTINEL.md aplicando a skill %s. "+
        "Audite o git diff atual e grave o relatório em .forge/SECURITY_REPORT.md.",
      sentinel.Skill,
    )
    sStats, err := runAgent(sentinel, secPrompt, *jsonMetrics)
    totalTokens.Add(sStats)
    breakdown = append(breakdown, fmt.Sprintf("F4B#%d(%s): %s", attempt, sentinel.Alias, sStats.Format()))
    if err != nil {
      fmt.Printf("[AVISO] SENTINEL (AppSec) retornou erro de processo: %v\n", err)
    }
    fmt.Printf("[TOKENS %s - %s] %s\n", sentinel.Forge, sentinel.Alias, sStats.Format())

    // FORGE 5 — ASSAY: ARBITER (Juiz Semântico & Circuit Breaker)
    printBanner(fmt.Sprintf("FORGE 5: ASSAY — ARBITER (Tentativa %d/%d)", attempt, *maxAttempts),
      fmt.Sprintf("Modelo: %s (%s)", arbiter.Model, arbiter.Bin))

    arbPrompt := fmt.Sprintf(
      "Siga estritamente as instruções de @.forge/05_ARBITER.md. "+
        "Esta é a tentativa ATTEMPT: %d de %d. "+
        "Analise @.forge/PLAN.md, @.forge/TEST_CASES.md, @.forge/VERIFY_REPORT.md, @.forge/SECURITY_REPORT.md e execute git diff. "+
        "Grave seu veredito em .forge/DECISION.md.",
      attempt, *maxAttempts,
    )

    aStats, err := runAgent(arbiter, arbPrompt, *jsonMetrics)
    totalTokens.Add(aStats)
    breakdown = append(breakdown, fmt.Sprintf("F5#%d(%s): %s", attempt, arbiter.Alias, aStats.Format()))
    if err != nil {
      fatal("Falha na Forge 5 (ARBITER): %v", err)
    }
    fmt.Printf("[TOKENS %s - %s] %s\n", arbiter.Forge, arbiter.Alias, aStats.Format())
    fmt.Printf("[TOKENS ACUMULADOS NO FORGELOOP] %s\n", totalTokens.Format())

    decisionContent, err := readFile(".forge/DECISION.md")
    if err != nil {
      fatal("Não foi possível ler .forge/DECISION.md: %v", err)
    }

    switch {
    case strings.Contains(decisionContent, "DECISION: APPROVE_MERGE"):
      updateDemandStatus(demandFile, "APPROVED", attempt, totalTokens, breakdown)
      appendTelemetry(stack.Name, demandFile, modelsSummary, attempt, totalTokens, "APPROVED", "Testes, AppSec e Diff-Judge aprovados")
      printBanner("FORGELOOP CONCLUÍDO — APPROVE_MERGE",
        fmt.Sprintf("Aprovado em %d tentativa(s) | Gasto Total: %s", attempt, totalTokens.Format()))
      os.Exit(0)

    case strings.Contains(decisionContent, "DECISION: AMEND_TEST"):
      updateDemandStatus(demandFile, "AMEND_TEST", attempt, totalTokens, breakdown)
      appendTelemetry(stack.Name, demandFile, modelsSummary, attempt, totalTokens, "AMEND_TEST", "Erro detectado no teste da Forge 1")
      printBanner("INTERVENÇÃO NECESSÁRIA — AMEND_TEST", "O ARBITER identificou bug em um teste da Forge 1. Revise .forge/DECISION.md.")
      os.Exit(3)

    case strings.Contains(decisionContent, "DECISION: ESCALATE_ARCHITECT"):
      updateDemandStatus(demandFile, "ESCALATE_ARCHITECT", attempt, totalTokens, breakdown)
      appendTelemetry(stack.Name, demandFile, modelsSummary, attempt, totalTokens, "ESCALATE_ARCHITECT", "Escalonado para revisão arquitetural")
      printBanner("CIRCUIT BREAKER — ESCALATE_ARCHITECT", "Falha estrutural ou limite atingido. Consulte .forge/DECISION.md.")
      os.Exit(4)

    case strings.Contains(decisionContent, "DECISION: RETRY_EXECUTION"):
      if !checksPassed {
        fmt.Println("\n[LOOP] Testes falharam na Forge 4A. Reiniciando Forge 3 (Strike) com contexto limpo...")
      } else {
        fmt.Println("\n[LOOP] Apontamento de segurança ou qualidade semântica. Reiniciando Forge 3 (Strike)...")
      }
      continue

    default:
      updateDemandStatus(demandFile, "UNKNOWN_VERDICT", attempt, totalTokens, breakdown)
      appendTelemetry(stack.Name, demandFile, modelsSummary, attempt, totalTokens, "UNKNOWN_VERDICT", "Veredito não reconhecido")
      fatal("Veredito não reconhecido em .forge/DECISION.md.")
    }
  }

  updateDemandStatus(demandFile, "CIRCUIT_BREAKER_TRIPPED", *maxAttempts, totalTokens, breakdown)
  appendTelemetry(stack.Name, demandFile, modelsSummary, *maxAttempts, totalTokens, "CIRCUIT_BREAKER_TRIPPED", "Limite máximo de tentativas excedido")
  printBanner("CIRCUIT BREAKER ACIONADO",
    fmt.Sprintf("Limite de %d tentativas atingido | Gasto Total: %s", *maxAttempts, totalTokens.Format()))
  os.Exit(1)
}

func runProjectSetup(reader *bufio.Reader, onboarder AgentBinding, defaultDocs string, jsonMetrics bool) {
  printBanner("FORGE 0: DISCOVERY — SETUP DE PROJETO EXISTENTE",
    fmt.Sprintf("Agente: %s (%s / %s)", onboarder.Alias, onboarder.Bin, onboarder.Model))

  fmt.Print("\nOnde está localizada a documentação atual do projeto? (ex: docs, wiki, README.md ou Enter para criar /docs): ")
  existingDocs, _ := reader.ReadString('\n')
  existingDocs = strings.TrimSpace(existingDocs)

  targetDocsDir := defaultDocs
  if existingDocs != "" && !strings.HasSuffix(existingDocs, ".md") {
    targetDocsDir = strings.Trim(existingDocs, "/")
  }
  if targetDocsDir == "" {
    targetDocsDir = "docs"
  }

  _ = os.MkdirAll(filepath.Join(targetDocsDir, "demands"), 0755)

  projectEnv := fmt.Sprintf("FORGE_DOCS_DIR=%s\nFORGE_EXISTING_DOCS=%s\n", targetDocsDir, existingDocs)
  _ = os.WriteFile(".forge/project.env", []byte(projectEnv), 0644)

  sourceHint := existingDocs
  if sourceHint == "" {
    sourceHint = "Nenhuma documentação prévia informada (criar estrutura inicial em " + targetDocsDir + "/)"
  }

  prompt := fmt.Sprintf(
    "Siga estritamente as instruções de @.forge/00_ONBOARDER.md. "+
      "Documentação existente informada pelo usuário: '%s'. "+
      "Diretório oficial de saída para documentação: '%s/'. "+
      "Analise a estrutura do código existente, leia a documentação informada e gere %s/PROJECT_CONTEXT.md e %s/DEMANDS_INDEX.md.",
    sourceHint, targetDocsDir, targetDocsDir, targetDocsDir,
  )

  stats, err := runAgent(onboarder, prompt, jsonMetrics)
  if err != nil {
    fatal("Falha na Forge 0 (ONBOARDER): %v", err)
  }

  fmt.Printf("\n[TOKENS %s - %s] %s\n", onboarder.Forge, onboarder.Alias, stats.Format())
  printBanner("FORGE 0 CONCLUÍDA",
    fmt.Sprintf("Contexto salvo em /%s/PROJECT_CONTEXT.md.\nDispare novas demandas com: go run forgeloop.go -task \"...\"", targetDocsDir))
}

func registerDemandInDocs(docsDir, stackName, task, modelsSummary string) string {
  demandsDir := filepath.Join(docsDir, "demands")
  _ = os.MkdirAll(demandsDir, 0755)

  datePrefix := time.Now().Format("2006-01-02")
  slug := slugify(task)
  fileName := fmt.Sprintf("%s-%s.md", datePrefix, slug)
  fullPath := filepath.Join(demandsDir, fileName)

  if _, err := os.Stat(fullPath); os.IsNotExist(err) {
    content := fmt.Sprintf(
      "# Demanda: %s\n\n"+
        "- **Data:** %s\n"+
        "- **Stack:** %s\n"+
        "- **Bindings:** `%s`\n"+
        "- **Status:** `IN_PROGRESS`\n"+
        "- **Tentativas:** `0`\n"+
        "- **Tokens Totais:** `0`\n"+
        "- **Detalhamento por Forge:** `pendente`\n\n"+
        "## Descrição Original\n\n%s\n\n"+
        "## Blueprint & Critérios de Aceite\n\n*(Será preenchido automaticamente pelo ARCHITECT na Forge 1)*\n",
      task, time.Now().Format("2006-01-02 15:04"), stackName, modelsSummary, task,
    )
    _ = os.WriteFile(fullPath, []byte(content), 0644)
  }

  indexPath := filepath.Join(docsDir, "DEMANDS_INDEX.md")
  if _, err := os.Stat(indexPath); os.IsNotExist(err) {
    header := "# Índice de Demandas — forgeloop\n\n| Data | Stack | Demanda | Arquivo | Status |\n| :--- | :--- | :--- | :--- | :--- |\n"
    _ = os.WriteFile(indexPath, []byte(header), 0644)
  }

  cleanTask := strings.ReplaceAll(task, "|", "-")
  row := fmt.Sprintf("| %s | %s | %s | [`%s`](demands/%s) | `IN_PROGRESS` |\n",
    datePrefix, stackName, cleanTask, fileName, fileName)

  if f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_WRONLY, 0644); err == nil {
    defer f.Close()
    _, _ = f.WriteString(row)
  }

  return filepath.ToSlash(fullPath)
}

func updateDemandStatus(demandPath, status string, attempts int, tokens TokenStats, breakdown []string) {
  if demandPath == "" {
    return
  }
  b, err := os.ReadFile(demandPath)
  if err != nil {
    return
  }
  content := string(b)
  content = strings.Replace(content, "- **Status:** `IN_PROGRESS`", fmt.Sprintf("- **Status:** `%s`", status), 1)
  content = strings.Replace(content, "- **Tentativas:** `0`", fmt.Sprintf("- **Tentativas:** `%d`", attempts), 1)
  content = strings.Replace(content, "- **Tokens Totais:** `0`", fmt.Sprintf("- **Tokens Totais:** `%s`", tokens.Format()), 1)
  if len(breakdown) > 0 {
    content = strings.Replace(content, "- **Detalhamento por Forge:** `pendente`",
      fmt.Sprintf("- **Detalhamento por Forge:** `%s`", strings.Join(breakdown, " | ")), 1)
  }
  _ = os.WriteFile(demandPath, []byte(content), 0644)
}

func selectStack(reader *bufio.Reader) StackConfig {
  fmt.Println("\nQual stack deve ser utilizada nesta execução?")
  fmt.Println("  [1] Ruby on Rails")
  fmt.Println("  [2] Go")
  fmt.Println("  [3] TypeScript + Node.js")
  fmt.Println("  [4] Personalizada")
  fmt.Print("Escolha uma opção (1-4): ")

  choice, _ := reader.ReadString('\n')
  switch strings.TrimSpace(choice) {
  case "1":
    testCmd := "bundle exec rspec"
    if _, err := os.Stat("spec"); os.IsNotExist(err) {
      testCmd = "bin/rails test"
    }
    return StackConfig{
      Name:    "Ruby on Rails",
      TestCmd: testCmd,
      LintCmd: "bundle exec rubocop",
    }
  case "2":
    lintCmd := ""
    if _, err := os.Stat(".golangci.yml"); err == nil {
      lintCmd = "golangci-lint run"
    }
    return StackConfig{
      Name:    "Go",
      TestCmd: "go test -v -race ./...",
      TypeCmd: "go vet ./...",
      LintCmd: lintCmd,
    }
  case "3":
    lintCmd := ""
    if pkg, err := readFile("package.json"); err == nil && strings.Contains(pkg, `"lint"`) {
      lintCmd = "npm run lint"
    }
    return StackConfig{
      Name:    "TypeScript + Node.js",
      TestCmd: "npm test",
      TypeCmd: "npx tsc --noEmit",
      LintCmd: lintCmd,
    }
  case "4":
    fmt.Print("Comando de Testes (ex: pytest): ")
    tCmd, _ := reader.ReadString('\n')
    fmt.Print("Comando de Tipos/Compilação (opcional, Enter para pular): ")
    typeCmd, _ := reader.ReadString('\n')
    fmt.Print("Comando de Linter (opcional, Enter para pular): ")
    lCmd, _ := reader.ReadString('\n')
    return StackConfig{
      Name:    "Personalizada",
      TestCmd: strings.TrimSpace(tCmd),
      TypeCmd: strings.TrimSpace(typeCmd),
      LintCmd: strings.TrimSpace(lCmd),
    }
  default:
    fatal("Opção inválida. Escolha entre 1 e 4.")
    return StackConfig{}
  }
}

func runDeterministicChecks(stack StackConfig) bool {
  var report strings.Builder
  allPassed := true

  commands := []struct {
    label string
    cmd   string
  }{
    {"Suíte de Testes", stack.TestCmd},
    {"Verificação de Tipos / Vet", stack.TypeCmd},
    {"Linter", stack.LintCmd},
  }

  for _, c := range commands {
    if strings.TrimSpace(c.cmd) == "" {
      continue
    }
    fmt.Printf("\n> Executando %s: %s\n", c.label, c.cmd)
    output, err := runShellCommand(c.cmd)
    status := "PASSED"
    if err != nil {
      status = "FAILED"
      allPassed = false
    }
    report.WriteString(fmt.Sprintf("## %s (`%s`) — %s\n\n```text\n%s\n```\n\n", c.label, c.cmd, status, output))
  }

  header := fmt.Sprintf("# Relatório de Verificação Determinística (Forge 4A)\n\nALL_CHECKS_PASSED: %v\nStack: %s\nTimestamp: %s\n\n",
    allPassed, stack.Name, time.Now().Format(time.RFC3339))

  _ = os.WriteFile(".forge/VERIFY_REPORT.md", []byte(header+report.String()), 0644)
  return allPassed
}

func runAgent(agent AgentBinding, prompt string, tryJSON bool) (TokenStats, error) {
  baseArgs := []string{}
  if strings.TrimSpace(agent.Model) != "" {
    baseArgs = append(baseArgs, "--model", agent.Model)
  }
  baseArgs = append(baseArgs, "-p", prompt)

  if tryJSON {
    jsonArgs := append(append([]string{}, baseArgs...), "--output-format", "json")
    rawOut, err := execCapture(agent.Bin, jsonArgs, false)
    if err == nil {
      if text, stats, ok := parseJSONUsage(rawOut); ok {
        fmt.Println(text)
        return stats, nil
      }
    }
  }

  rawOut, err := execCapture(agent.Bin, baseArgs, true)
  stats := extractOrEstimateTokens(prompt, rawOut)
  return stats, err
}

func execCapture(bin string, args []string, streamToConsole bool) (string, error) {
  cmd := exec.Command(bin, args...)
  var buf bytes.Buffer
  if streamToConsole {
    mw := io.MultiWriter(os.Stdout, &buf)
    cmd.Stdout = mw
    cmd.Stderr = mw
  } else {
    cmd.Stdout = &buf
    cmd.Stderr = &buf
  }
  err := cmd.Run()
  return buf.String(), err
}

func parseJSONUsage(raw string) (string, TokenStats, bool) {
  start := strings.Index(raw, "{")
  end := strings.LastIndex(raw, "}")
  if start == -1 || end <= start {
    return "", TokenStats{}, false
  }

  var payload map[string]any
  if err := json.Unmarshal([]byte(raw[start:end+1]), &payload); err != nil {
    return "", TokenStats{}, false
  }

  text := ""
  for _, key := range []string{"result", "response", "content", "text", "message"} {
    if v, ok := payload[key].(string); ok && v != "" {
      text = v
      break
    }
  }

  var stats TokenStats
  if usage, ok := payload["usage"].(map[string]any); ok {
    stats.Input = intFromMap(usage, "input_tokens", "prompt_tokens", "promptTokenCount")
    stats.Output = intFromMap(usage, "output_tokens", "completion_tokens", "candidatesTokenCount")
    stats.Total = intFromMap(usage, "total_tokens", "totalTokenCount")
    if stats.Total == 0 {
      stats.Total = stats.Input + stats.Output
    }
    if stats.Total > 0 {
      return text, stats, true
    }
  }

  return "", TokenStats{}, false
}

func intFromMap(m map[string]any, keys ...string) int {
  for _, k := range keys {
    if val, ok := m[k].(float64); ok {
      return int(val)
    }
  }
  return 0
}

func extractOrEstimateTokens(prompt, output string) TokenStats {
  reIn := regexp.MustCompile(`(?i)(?:input|prompt)[_\s-]*tokens["'\s:=]+(\d+)`)
  reOut := regexp.MustCompile(`(?i)(?:output|completion|candidates)[_\s-]*tokens["'\s:=]+(\d+)`)

  inMatch := reIn.FindStringSubmatch(output)
  outMatch := reOut.FindStringSubmatch(output)

  if len(inMatch) > 1 && len(outMatch) > 1 {
    inVal, _ := strconv.Atoi(inMatch[1])
    outVal, _ := strconv.Atoi(outMatch[1])
    return TokenStats{
      Input:     inVal,
      Output:    outVal,
      Total:     inVal + outVal,
      Estimated: false,
    }
  }

  estIn := (len(prompt) / 4) + 1200
  estOut := len(output) / 4
  return TokenStats{
    Input:     estIn,
    Output:    estOut,
    Total:     estIn + estOut,
    Estimated: true,
  }
}

func runShellCommand(command string) (string, error) {
  cmd := exec.Command("sh", "-c", command)
  var buf bytes.Buffer
  mw := io.MultiWriter(os.Stdout, &buf)
  cmd.Stdout = mw
  cmd.Stderr = mw
  err := cmd.Run()
  return buf.String(), err
}

func appendTelemetry(stackName, demandFile, modelsSummary string, attempts int, tokens TokenStats, status, note string) {
  path := ".forge/TELEMETRY.md"
  if _, err := os.Stat(path); os.IsNotExist(err) {
    header := "# Telemetria forgeloop\n\n| Data | Stack | Demanda | Bindings Testados | Tentativas | Tokens (In / Out / Total) | Status | Observação |\n| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n"
    _ = os.WriteFile(path, []byte(header), 0644)
  }

  if demandFile == "" {
    demandFile = "(via PLAN.md)"
  }
  row := fmt.Sprintf("| %s | %s | `%s` | %s | %d | %s | **%s** | %s |\n",
    time.Now().Format("2006-01-02 15:04"), stackName, demandFile, modelsSummary, attempts, tokens.Format(), status, note)

  if f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644); err == nil {
    defer f.Close()
    _, _ = f.WriteString(row)
  }
}

func loadEnvFile(path string) {
  b, err := os.ReadFile(path)
  if err != nil {
    return
  }
  for _, line := range strings.Split(string(b), "\n") {
    line = strings.TrimSpace(line)
    if line == "" || strings.HasPrefix(line, "#") {
      continue
    }
    parts := strings.SplitN(line, "=", 2)
    if len(parts) != 2 {
      continue
    }
    key := strings.TrimSpace(parts[0])
    val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
    if os.Getenv(key) == "" {
      _ = os.Setenv(key, val)
    }
  }
}

func getEnv(key, fallback string) string {
  if v := strings.TrimSpace(os.Getenv(key)); v != "" {
    return v
  }
  return fallback
}

func getEnvInt(key string, fallback int) int {
  if v := strings.TrimSpace(os.Getenv(key)); v != "" {
    if i, err := strconv.Atoi(v); err == nil {
      return i
    }
  }
  return fallback
}

func slugify(s string) string {
  var b strings.Builder
  lastDash := false
  for _, r := range strings.ToLower(s) {
    if unicode.IsLetter(r) || unicode.IsDigit(r) {
      b.WriteRune(r)
      lastDash = false
    } else if !lastDash {
      b.WriteRune('-')
      lastDash = true
    }
  }
  res := strings.Trim(b.String(), "-")
  if len(res) > 45 {
    res = res[:45]
  }
  if res == "" {
    return "demanda"
  }
  return res
}

func readFile(path string) (string, error) {
  b, err := os.ReadFile(path)
  if err != nil {
    return "", err
  }
  return string(b), nil
}

func printBanner(title, subtitle string) {
  fmt.Printf("\n==============================================================================\n")
  fmt.Printf(" %s\n %s\n", title, subtitle)
  fmt.Printf("==============================================================================\n")
}

func fatal(format string, args ...any) {
  fmt.Fprintf(os.Stderr, "\n[ERRO FATAL] "+format+"\n", args...)
  os.Exit(1)
}

