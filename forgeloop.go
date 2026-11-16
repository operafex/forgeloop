package main

import (
  "bufio"
  "bytes"
  "encoding/csv"
  "encoding/json"
  "flag"
  "fmt"
  "io"
  "os"
  "os/exec"
  "path/filepath"
  "regexp"
  "sort"
  "strconv"
  "strings"
  "sync"
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
  Forge string // rótulo da Etapa (ex.: "ETAPA 1"); nome do campo mantido por compatibilidade
  Alias string
  Bin   string
  Model string
  Skill string
  Args  string
  Stage string // nome da etapa (ex.: BLUEPRINT), usado no spinner e nos logs
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
  runID := newRunID()

  loadEnvFile(".forge/config/models.env")
  loadEnvFile(".forge/config/project.env")

  setupMode := flag.Bool("setup", false, "Etapa 0 (Discovery): analisa projeto existente e mapeia documentação")
  taskFlag := flag.String("task", "", "Descrição da nova demanda (salva automaticamente em /docs/demands/)")
  initMode := flag.Bool("init", false, "Verifica binários, instala skills ausentes (npx skills add) e cria a estrutura de pastas")
  forceSkills := flag.Bool("force-skills", false, "Com -init: reinstala/atualiza todas as skills configuradas, mesmo as já presentes")
  stageFlag := flag.Int("stage", 1, "Etapa inicial da Forja: 1 (ciclo completo), 2 (Temper/Skeptic), 3 (Strike/Builder)")
  forgeAlias := flag.Int("forge", 1, "Alias de compatibilidade para -stage")
  maxAttempts := flag.Int("max-attempts", getEnvInt("FORGE_MAX_ATTEMPTS", 3), "Limite máximo de tentativas no loop (Circuit Breaker)")
  docsDirFlag := flag.String("docs-dir", getEnv("FORGE_DOCS_DIR", getEnv("FORGE_DEFAULT_DOCS_DIR", "docs")), "Diretório de documentação")
  var refFlags stringList
  flag.Var(&refFlags, "ref", "Arquivo de referência para enriquecer a demanda (repetível; caminho direto ou nome dentro de <DOCS_DIR>/references/)")
  jsonMetrics := flag.Bool("json-metrics", true, "Extrai contagem exata de tokens via --output-format json quando suportado")

  // Bindings desacoplados: troque via .forge/config/models.env ou flags CLI
  onbBin := flag.String("onboarder-bin", getEnv("FORGE_ONBOARDER_BIN", "claude"), "Binário da Etapa 0 (ONBOARDER)")
  onbModel := flag.String("onboarder-model", getEnv("FORGE_ONBOARDER_MODEL", "opus"), "Modelo da Etapa 0 (ONBOARDER)")
  onbArgs := flag.String("onboarder-args", getEnv("FORGE_ONBOARDER_ARGS", ""), "Argumentos extras do CLI da Etapa 0 (ONBOARDER)")

  archBin := flag.String("architect-bin", getEnv("FORGE_ARCHITECT_BIN", "claude"), "Binário da Etapa 1 (ARCHITECT)")
  archModel := flag.String("architect-model", getEnv("FORGE_ARCHITECT_MODEL", "opus"), "Modelo da Etapa 1 (ARCHITECT)")
  archSkill := flag.String("architect-skill", getEnv("FORGE_ARCHITECT_SKILL", "obra/superpowers"), "Skill da Etapa 1 (ARCHITECT)")
  archArgs := flag.String("architect-args", getEnv("FORGE_ARCHITECT_ARGS", ""), "Argumentos extras do CLI da Etapa 1 (ARCHITECT)")

  skepBin := flag.String("skeptic-bin", getEnv("FORGE_SKEPTIC_BIN", "agy"), "Binário da Etapa 2 (SKEPTIC)")
  skepModel := flag.String("skeptic-model", getEnv("FORGE_SKEPTIC_MODEL", "GPT-OSS 120B (Medium)"), "Modelo da Etapa 2 (SKEPTIC)")
  skepSkill := flag.String("skeptic-skill", getEnv("FORGE_SKEPTIC_SKILL", "google/elephant-goldfish-model"), "Skill da Etapa 2 (SKEPTIC)")
  skepArgs := flag.String("skeptic-args", getEnv("FORGE_SKEPTIC_ARGS", "--dangerously-skip-permissions"), "Argumentos extras do CLI da Etapa 2 (SKEPTIC)")

  buildBin := flag.String("builder-bin", getEnv("FORGE_BUILDER_BIN", "agy"), "Binário da Etapa 3 (BUILDER)")
  buildModel := flag.String("builder-model", getEnv("FORGE_BUILDER_MODEL", "Gemini 3.8 Flash (High)"), "Modelo da Etapa 3 (BUILDER)")
  buildArgs := flag.String("builder-args", getEnv("FORGE_BUILDER_ARGS", "--dangerously-skip-permissions"), "Argumentos extras do CLI da Etapa 3 (BUILDER)")

  sentBin := flag.String("sentinel-bin", getEnv("FORGE_SENTINEL_BIN", "agy"), "Binário da Etapa 4B (SENTINEL)")
  sentModel := flag.String("sentinel-model", getEnv("FORGE_SENTINEL_MODEL", "Claude Sonnet 4.6 (Thinking)"), "Modelo da Etapa 4B (SENTINEL)")
  sentSkill := flag.String("sentinel-skill", getEnv("FORGE_SENTINEL_SKILL", "cloudflare/security-audit-skill"), "Skill da Etapa 4B (SENTINEL)")
  sentArgs := flag.String("sentinel-args", getEnv("FORGE_SENTINEL_ARGS", "--dangerously-skip-permissions"), "Argumentos extras do CLI da Etapa 4B (SENTINEL)")

  arbBin := flag.String("arbiter-bin", getEnv("FORGE_ARBITER_BIN", "agy"), "Binário da Etapa 5 (ARBITER)")
  arbModel := flag.String("arbiter-model", getEnv("FORGE_ARBITER_MODEL", "Claude Sonnet 4.6 (Thinking)"), "Modelo da Etapa 5 (ARBITER)")
  arbArgs := flag.String("arbiter-args", getEnv("FORGE_ARBITER_ARGS", "--dangerously-skip-permissions"), "Argumentos extras do CLI da Etapa 5 (ARBITER)")

  flag.Parse()

  setFlags := map[string]bool{}
  flag.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })
  startStage, stageErr := resolveStage(*stageFlag, *forgeAlias, setFlags["stage"], setFlags["forge"])
  if stageErr != nil {
    fatal("%v", stageErr)
  }

  if err := ensureForgeDirs(); err != nil {
    fatal("Erro ao criar estrutura .forge/: %v", err)
  }

  onboarder := AgentBinding{Forge: "ETAPA 0", Stage: "DISCOVERY", Alias: "ONBOARDER", Bin: *onbBin, Model: *onbModel, Args: *onbArgs}
  architect := AgentBinding{Forge: "ETAPA 1", Stage: "BLUEPRINT", Alias: "ARCHITECT", Bin: *archBin, Model: *archModel, Skill: *archSkill, Args: *archArgs}
  skeptic := AgentBinding{Forge: "ETAPA 2", Stage: "TEMPER", Alias: "SKEPTIC", Bin: *skepBin, Model: *skepModel, Skill: *skepSkill, Args: *skepArgs}
  builder := AgentBinding{Forge: "ETAPA 3", Stage: "STRIKE", Alias: "BUILDER", Bin: *buildBin, Model: *buildModel, Args: *buildArgs}
  sentinel := AgentBinding{Forge: "ETAPA 4B", Stage: "QUENCH APPSEC", Alias: "SENTINEL", Bin: *sentBin, Model: *sentModel, Skill: *sentSkill, Args: *sentArgs}
  arbiter := AgentBinding{Forge: "ETAPA 5", Stage: "ASSAY", Alias: "ARBITER", Bin: *arbBin, Model: *arbModel, Args: *arbArgs}

  reader := bufio.NewReader(os.Stdin)

  // =========================================================================
  // -init: verificação de binários, instalação de skills e estrutura de pastas
  // =========================================================================
  if *initMode {
    initDocs := strings.Trim(strings.TrimSpace(*docsDirFlag), "/")
    if initDocs == "" {
      initDocs = "docs"
    }
    runInit(initDocs, []AgentBinding{onboarder, architect, skeptic, builder, sentinel, arbiter}, *forceSkills)
    return
  }

  // =========================================================================
  // ETAPA 0 — DISCOVERY (-setup): ANÁLISE DE PROJETO EXISTENTE
  // =========================================================================
  if *setupMode {
    runProjectSetup(reader, onboarder, *docsDirFlag, *jsonMetrics)
    finish(0)
    return
  }

  stack, err := selectStack(reader, os.Stdout, projectEnvPath)
  if err != nil {
    fatal("%v", err)
  }
  docsDir := strings.Trim(strings.TrimSpace(*docsDirFlag), "/")
  if docsDir == "" {
    docsDir = "docs"
  }

  refs, refErr := resolveReferences(refFlags, docsDir)
  if refErr != nil {
    fatal("%v", refErr)
  }
  if len(refs) > 0 && startStage > 1 {
    fmt.Println("[AVISO] -ref só é usado pelo ARCHITECT (Etapa 1); ignorado ao iniciar a partir da Etapa", startStage)
    refs = nil
  }

  task := strings.TrimSpace(*taskFlag)
  if startStage == 1 && task == "" {
    fmt.Print("\nDescreva a nova demanda ou feature: ")
    input, _ := reader.ReadString('\n')
    task = strings.TrimSpace(input)
    if task == "" {
      fatal("A descrição da demanda não pode estar vazia na Etapa 1.")
    }
  }

  modelsSummary := fmt.Sprintf("F1:%s | F2:%s | F3:%s | F4B:%s | F5:%s",
    architect.Model, skeptic.Model, builder.Model, sentinel.Model, arbiter.Model)

  // Log completo da execução em <DOCS_DIR>/logs/ (um arquivo por execução).
  logSuffix := fmt.Sprintf("forge%d", startStage)
  if task != "" {
    logSuffix = slugify(task)
  }
  startRunLog(docsDir, logSuffix)
  logName := ""
  if runLog != nil {
    logName = runLog.Name
  }

  demandFile := ""
  if task != "" {
    demandFile = registerDemandInDocs(docsDir, stack.Name, task, modelsSummary, logName, refs)
    fmt.Printf("\n[DOCS] Demanda registrada em: %s\n", demandFile)
  }

  logBlock("CABEÇALHO DA EXECUÇÃO", buildRunHeader(stack, []AgentBinding{architect, skeptic, builder, sentinel, arbiter},
    task, demandFile, startStage, *maxAttempts, docsDir))
  if len(refs) > 0 {
    fmt.Printf("[DOCS] Referências anexadas à demanda: %s\n", strings.Join(refs, ", "))
    logEvent("Referências anexadas à demanda: %s", strings.Join(refs, ", "))
  }

  var totalTokens TokenStats

  preflightSkills(stageSkills(startStage, architect, skeptic, sentinel))

  printBanner("A FORJA (FORGELOOP) INICIADA",
    fmt.Sprintf("Stack: %s | Docs: /%s | Max Tentativas: %d\nBindings: %s", stack.Name, docsDir, *maxAttempts, modelsSummary))

  // =========================================================================
  // ETAPA 1 — BLUEPRINT: ARCHITECT (Spec-TDD-Planner / Superpowers)
  // =========================================================================
  if startStage <= 1 {
    printBanner("ETAPA 1: BLUEPRINT — ARCHITECT", fmt.Sprintf("Skill: %s | Modelo: %s (%s)", architect.Skill, architect.Model, architect.Bin))
    prompt := fmt.Sprintf(
      "Siga estritamente as instruções de @.forge/agents/01_ARCHITECT.md aplicando a skill %s. "+
        "Diretório de documentação do projeto: %s/ (leia %s/PROJECT_CONTEXT.md se existir). "+
        "Stack selecionada: %s. Demanda: %s (arquivo de registro: %s). "+
        "Gere .forge/state/PLAN.md (dividido em Tasks atômicas), .forge/state/TEST_CASES.md, anexe o plano técnico em %s e crie os testes automatizados em estado RED.",
      architect.Skill, docsDir, docsDir, stack.Name, task, demandFile, demandFile,
    )
    prompt += referencesPrompt(refs)
    start := time.Now()
    stats, err := runAgent(architect, prompt, *jsonMetrics)
    recordEventTelemetry(runID, "F1", architect.Alias, architect.Model, 1, stats, time.Since(start).Seconds())
    totalTokens.Add(stats)
    if err != nil {
      fatal("Falha na Etapa 1 (ARCHITECT): %v", err)
    }
    reportTokens(architect, stats)
  }

  // =========================================================================
  // ETAPA 2 — TEMPER: SKEPTIC (Cold-Global-Lens / Elephant-Goldfish)
  // =========================================================================
  if startStage <= 2 {
    printBanner("ETAPA 2: TEMPER — SKEPTIC", fmt.Sprintf("Skill: %s | Modelo: %s (%s)", skeptic.Skill, skeptic.Model, skeptic.Bin))
    prompt := fmt.Sprintf(
      "Siga estritamente as instruções de @.forge/agents/02_SKEPTIC.md aplicando a skill %s. "+
        "Diretório de documentação: %s/. Audite @.forge/state/PLAN.md, @.forge/state/TEST_CASES.md e os arquivos de teste recém-criados. "+
        "Grave seu veredito em .forge/state/SPEC_REVIEW.md.",
      skeptic.Skill, docsDir,
    )

    start := time.Now()
    stats, err := runAgent(skeptic, prompt, *jsonMetrics)
    recordEventTelemetry(runID, "F2", skeptic.Alias, skeptic.Model, 1, stats, time.Since(start).Seconds())
    totalTokens.Add(stats)
    if err != nil {
      fatal("Falha na Etapa 2 (SKEPTIC): %v", err)
    }
    reportTokens(skeptic, stats)

    reviewContent, err := readFile(".forge/state/SPEC_REVIEW.md")
    if err != nil {
      fatal("Não foi possível ler .forge/state/SPEC_REVIEW.md: %v", err)
    }
    logBlock(".forge/state/SPEC_REVIEW.md", reviewContent)

    if !strings.Contains(reviewContent, "SPEC_STATUS: APPROVED") {
      updateDemandStatus(demandFile, "SPEC_REVISION_REQUIRED", 0, totalTokens)
      recordRunTelemetry(runID, stack.Name, demandFile, *maxAttempts, 0, totalTokens, "SPEC_REVISION_REQUIRED", "Bloqueado pelo SKEPTIC na Etapa 2")
      printBanner("BLOQUEIO NA ETAPA 2 (TEMPER)", "O SKEPTIC exigiu ajustes na especificação. Consulte .forge/state/SPEC_REVIEW.md e rode novamente a Etapa 1.")
      finish(2)
    }
    fmt.Println("\n[OK] Especificação aprovada (SPEC_STATUS: APPROVED). Entrando no ciclo Strike -> Quench -> Assay...")
  }

  // =========================================================================
  // LOOP FECHADO: ETAPA 3 (STRIKE) -> ETAPA 4 (QUENCH) -> ETAPA 5 (ASSAY)
  // =========================================================================
  for attempt := 1; attempt <= *maxAttempts; attempt++ {
    // ETAPA 3 — STRIKE: BUILDER (Contexto Limpo)
    printBanner(fmt.Sprintf("ETAPA 3: STRIKE — BUILDER (Tentativa %d/%d)", attempt, *maxAttempts),
      fmt.Sprintf("Contexto Higienizado | Modelo: %s (%s)", builder.Model, builder.Bin))

    execPrompt := fmt.Sprintf(
      "Siga estritamente as instruções de @.forge/agents/03_BUILDER.md. Diretório de docs: %s/. "+
        "Leia @.forge/state/PLAN.md e @.forge/state/TEST_CASES.md e implemente as Tasks no código de produção.",
      docsDir,
    )
    if attempt > 1 {
      execPrompt += " ATENÇÃO: Rodada de correção. Leia @.forge/state/DECISION.md, @.forge/state/VERIFY_REPORT.md e @.forge/state/SECURITY_REPORT.md e corrija todos os apontamentos."
    }

    bStart := time.Now()
    bStats, err := runAgent(builder, execPrompt, *jsonMetrics)
    recordEventTelemetry(runID, "F3", builder.Alias, builder.Model, attempt, bStats, time.Since(bStart).Seconds())
    totalTokens.Add(bStats)
    if err != nil {
      fatal("Falha na Etapa 3 (BUILDER): %v (verifique FORGE_BUILDER_MODEL/FORGE_BUILDER_BIN em .forge/config/models.env)", err)
    }
    reportTokens(builder, bStats)

    // ETAPA 4A — QUENCH TESTS: Verificação Determinística
    printBanner(fmt.Sprintf("ETAPA 4A: QUENCH TESTS (Tentativa %d/%d)", attempt, *maxAttempts),
      fmt.Sprintf("Runner Determinístico (%s)", stack.Name))
    checksPassed := runDeterministicChecks(stack)

    // ETAPA 4B — QUENCH APPSEC: SENTINEL
    printBanner(fmt.Sprintf("ETAPA 4B: QUENCH APPSEC — SENTINEL (Tentativa %d/%d)", attempt, *maxAttempts),
      fmt.Sprintf("Skill: %s | Modelo: %s", sentinel.Skill, sentinel.Model))

    secPrompt := fmt.Sprintf(
      "Siga estritamente as instruções de @.forge/agents/04_SENTINEL.md aplicando a skill %s. "+
        "Audite o git diff atual e grave o relatório em .forge/state/SECURITY_REPORT.md.",
      sentinel.Skill,
    )
    secReportPath := ".forge/state/SECURITY_REPORT.md"
    var secModBefore time.Time
    if fi, statErr := os.Stat(secReportPath); statErr == nil {
      secModBefore = fi.ModTime()
    }
    sStart := time.Now()
    sStats, err := runAgent(sentinel, secPrompt, *jsonMetrics)
    recordEventTelemetry(runID, "F4B", sentinel.Alias, sentinel.Model, attempt, sStats, time.Since(sStart).Seconds())
    totalTokens.Add(sStats)
    if err != nil {
      fmt.Printf("[AVISO] SENTINEL (AppSec) retornou erro de processo: %v\n", err)
      logEvent("AVISO: SENTINEL retornou erro de processo: %v", err)
    }
    if secContent, readErr := readFile(secReportPath); readErr == nil {
      logBlock(".forge/state/SECURITY_REPORT.md (tentativa "+strconv.Itoa(attempt)+")", secContent)
    }
    if fi, statErr := os.Stat(secReportPath); statErr != nil {
      fmt.Printf("[AVISO] SENTINEL não gerou %s. A auditoria de segurança NÃO foi registrada (verifique permissões em FORGE_SENTINEL_ARGS).\n", secReportPath)
    } else if !secModBefore.IsZero() && !fi.ModTime().After(secModBefore) {
      fmt.Printf("[AVISO] SENTINEL não atualizou %s nesta tentativa; o relatório é de uma execução anterior.\n", secReportPath)
    }
    reportTokens(sentinel, sStats)

    // ETAPA 5 — ASSAY: ARBITER (Juiz Semântico & Circuit Breaker)
    printBanner(fmt.Sprintf("ETAPA 5: ASSAY — ARBITER (Tentativa %d/%d)", attempt, *maxAttempts),
      fmt.Sprintf("Modelo: %s (%s)", arbiter.Model, arbiter.Bin))

    arbPrompt := fmt.Sprintf(
      "Siga estritamente as instruções de @.forge/agents/05_ARBITER.md. "+
        "Esta é a tentativa ATTEMPT: %d de %d. "+
        "Analise @.forge/state/PLAN.md, @.forge/state/TEST_CASES.md, @.forge/state/VERIFY_REPORT.md, @.forge/state/SECURITY_REPORT.md e execute git diff. "+
        "Grave seu veredito em .forge/state/DECISION.md.",
      attempt, *maxAttempts,
    )

    aStart := time.Now()
    aStats, err := runAgent(arbiter, arbPrompt, *jsonMetrics)
    recordEventTelemetry(runID, "F5", arbiter.Alias, arbiter.Model, attempt, aStats, time.Since(aStart).Seconds())
    totalTokens.Add(aStats)
    if err != nil {
      fatal("Falha na Etapa 5 (ARBITER): %v", err)
    }
    reportTokens(arbiter, aStats)
    fmt.Printf("[TOKENS ACUMULADOS NA FORJA] %s\n", totalTokens.Format())
    logEvent("TOKENS ACUMULADOS NA FORJA: %s", totalTokens.Format())

    decisionContent, err := readFile(".forge/state/DECISION.md")
    if err != nil {
      fatal("Não foi possível ler .forge/state/DECISION.md: %v", err)
    }
    logBlock(".forge/state/DECISION.md (tentativa "+strconv.Itoa(attempt)+")", decisionContent)

    switch {
    case strings.Contains(decisionContent, "DECISION: APPROVE_MERGE"):
      updateDemandStatus(demandFile, "APPROVED", attempt, totalTokens)
      recordRunTelemetry(runID, stack.Name, demandFile, *maxAttempts, attempt, totalTokens, "APPROVED", "Testes, AppSec e Diff-Judge aprovados")
      printBanner("A FORJA CONCLUÍDA — APPROVE_MERGE",
        fmt.Sprintf("Aprovado em %d tentativa(s) | Gasto Total: %s", attempt, totalTokens.Format()))
      finish(0)

    case strings.Contains(decisionContent, "DECISION: AMEND_TEST"):
      updateDemandStatus(demandFile, "AMEND_TEST", attempt, totalTokens)
      recordRunTelemetry(runID, stack.Name, demandFile, *maxAttempts, attempt, totalTokens, "AMEND_TEST", "Erro detectado no teste da Etapa 1")
      printBanner("INTERVENÇÃO NECESSÁRIA — AMEND_TEST", "O ARBITER identificou bug em um teste da Etapa 1. Revise .forge/state/DECISION.md.")
      finish(3)

    case strings.Contains(decisionContent, "DECISION: ESCALATE_ARCHITECT"):
      updateDemandStatus(demandFile, "ESCALATE_ARCHITECT", attempt, totalTokens)
      recordRunTelemetry(runID, stack.Name, demandFile, *maxAttempts, attempt, totalTokens, "ESCALATE_ARCHITECT", "Escalonado para revisão arquitetural")
      printBanner("CIRCUIT BREAKER — ESCALATE_ARCHITECT", "Falha estrutural ou limite atingido. Consulte .forge/state/DECISION.md.")
      finish(4)

    case strings.Contains(decisionContent, "DECISION: RETRY_EXECUTION"):
      if !checksPassed {
        fmt.Println("\n[LOOP] Testes falharam na Etapa 4A. Reiniciando Etapa 3 (Strike) com contexto limpo...")
      } else {
        fmt.Println("\n[LOOP] Apontamento de segurança ou qualidade semântica. Reiniciando Etapa 3 (Strike)...")
      }
      continue

    default:
      updateDemandStatus(demandFile, "UNKNOWN_VERDICT", attempt, totalTokens)
      recordRunTelemetry(runID, stack.Name, demandFile, *maxAttempts, attempt, totalTokens, "UNKNOWN_VERDICT", "Veredito não reconhecido")
      fatal("Veredito não reconhecido em .forge/state/DECISION.md.")
    }
  }

  updateDemandStatus(demandFile, "CIRCUIT_BREAKER_TRIPPED", *maxAttempts, totalTokens)
  recordRunTelemetry(runID, stack.Name, demandFile, *maxAttempts, *maxAttempts, totalTokens, "CIRCUIT_BREAKER_TRIPPED", "Limite máximo de tentativas excedido")
  printBanner("CIRCUIT BREAKER ACIONADO",
    fmt.Sprintf("Limite de %d tentativas atingido | Gasto Total: %s", *maxAttempts, totalTokens.Format()))
  finish(1)
}

func runProjectSetup(reader *bufio.Reader, onboarder AgentBinding, defaultDocs string, jsonMetrics bool) {
  printBanner("ETAPA 0: DISCOVERY — SETUP DE PROJETO EXISTENTE",
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

  if err := ensureForgeDirs(); err != nil {
    fatal("Erro ao criar estrutura .forge/: %v", err)
  }
  if err := ensureDocsDirs(targetDocsDir); err != nil {
    fatal("Erro ao criar estrutura de pastas em %s: %v", targetDocsDir, err)
  }
  startRunLog(targetDocsDir, "forge0")
  logBlock("CABEÇALHO DA EXECUÇÃO", buildRunHeader(StackConfig{Name: "(n/a — Etapa 0 Discovery)"}, []AgentBinding{onboarder},
    "", "", 0, 0, targetDocsDir))

  detected, ok, envErr := setupProjectEnv(projectEnvPath, ".", targetDocsDir, existingDocs)
  if envErr != nil {
    fatal("Erro ao gravar %s: %v", projectEnvPath, envErr)
  }
  if ok {
    fmt.Printf("[STACK] Detectada: %s (testes: %s)\n", detected.Name, detected.TestCmd)
    logEvent("STACK DETECTADA: %s (testes: %s)", detected.Name, detected.TestCmd)
  } else {
    fmt.Println("[AVISO] Stack não detectada automaticamente. O ONBOARDER tentará identificá-la.")
    logEvent("AVISO: stack não detectada automaticamente")
  }

  sourceHint := existingDocs
  if sourceHint == "" {
    sourceHint = "Nenhuma documentação prévia informada (criar estrutura inicial em " + targetDocsDir + "/)"
  }

  detectedHint := "(nenhuma)"
  if ok {
    detectedHint = detected.Name
  }
  prompt := fmt.Sprintf(
    "Siga estritamente as instruções de @.forge/agents/00_ONBOARDER.md. "+
      "Documentação existente informada pelo usuário: '%s'. "+
      "Diretório oficial de saída para documentação: '%s/'. "+
      "Stack pré-detectada pelo CLI: '%s'. Valide ou corrija as chaves FORGE_STACK_NAME, FORGE_TEST_CMD, FORGE_TYPE_CMD e FORGE_LINT_CMD em .forge/config/project.env, preservando as demais chaves. "+
      "Analise a estrutura do código existente, leia a documentação informada e gere %s/PROJECT_CONTEXT.md e %s/demands/INDEX.md.",
    sourceHint, targetDocsDir, detectedHint, targetDocsDir, targetDocsDir,
  )

  stats, err := runAgent(onboarder, prompt, jsonMetrics)
  if err != nil {
    fatal("Falha na Etapa 0 (ONBOARDER): %v", err)
  }

  reportTokens(onboarder, stats)

  contextPath := filepath.Join(targetDocsDir, "PROJECT_CONTEXT.md")
  if info, statErr := os.Stat(contextPath); statErr != nil || info.IsDir() {
    fatal("Etapa 0 terminou sem gerar %s. O ONBOARDER pode ter sido bloqueado por permissões (veja FORGE_ONBOARDER_ARGS em .forge/config/models.env) ou falhou ao gravar o arquivo.", contextPath)
  }

  if err := ensureDocsStructureSection(contextPath, targetDocsDir); err != nil {
    fmt.Printf("[AVISO] Não foi possível documentar a estrutura de pastas em %s: %v\n", contextPath, err)
    logEvent("AVISO: falha ao documentar estrutura em %s: %v", contextPath, err)
  }

  savedStack, loadErr := loadProjectStack(projectEnvPath)
  bannerMsg := fmt.Sprintf("Contexto salvo em /%s/PROJECT_CONTEXT.md.", targetDocsDir)
  if loadErr == nil {
    bannerMsg += fmt.Sprintf("\nStack salva: %s", savedStack.Name)
  } else {
    fmt.Printf("[AVISO] Stack não configurada em project.env (motivo: %v). O menu manual será exibido no -task.\n", loadErr)
    logEvent("AVISO: stack não configurada em project.env: %v", loadErr)
  }
  bannerMsg += "\nDispare novas demandas com: go run . -task \"...\""
  printBanner("ETAPA 0 CONCLUÍDA", bannerMsg)
}

// referencesLine devolve a linha "- **Referências:**" da demanda (vazia quando não há referências).
func referencesLine(refs []string) string {
  if len(refs) == 0 {
    return ""
  }
  quoted := make([]string, len(refs))
  for i, r := range refs {
    quoted[i] = "`" + r + "`"
  }
  return "- **Referências:** " + strings.Join(quoted, ", ") + "\n"
}

func registerDemandInDocs(docsDir, stackName, task, modelsSummary, logName string, refs []string) string {
  demandsDir := filepath.Join(docsDir, "demands")
  _ = os.MkdirAll(demandsDir, 0755)

  datePrefix := time.Now().Format("2006-01-02")
  slug := slugify(task)
  fileName := fmt.Sprintf("%s-%s.md", datePrefix, slug)
  fullPath := filepath.Join(demandsDir, fileName)

  logLine := ""
  if logName != "" {
    logLine = fmt.Sprintf("- **Log de Execução:** [`%s`](../logs/%s)\n", logName, logName)
  }

  if _, err := os.Stat(fullPath); os.IsNotExist(err) {
    content := fmt.Sprintf(
      "# Demanda: %s\n\n"+
        "- **Data:** %s\n"+
        "- **Stack:** %s\n"+
        "- **Bindings:** `%s`\n"+
        "- **Status:** `IN_PROGRESS`\n"+
        "- **Tentativas:** `0`\n"+
        "- **Tokens Totais:** `0`\n"+
        "%s%s\n"+
        "## Descrição Original\n\n%s\n\n"+
        "## Blueprint & Critérios de Aceite\n\n*(Será preenchido automaticamente pelo ARCHITECT na Etapa 1)*\n",
      task, time.Now().Format("2006-01-02 15:04"), stackName, modelsSummary, logLine, referencesLine(refs), task,
    )
    _ = os.WriteFile(fullPath, []byte(content), 0644)
  } else if b, readErr := os.ReadFile(fullPath); readErr == nil {
    // Demanda já registrada (nova execução no mesmo dia): vincula também o log e as referências desta execução.
    existing := string(b)
    extra := ""
    if logLine != "" && !strings.Contains(existing, "(../logs/"+logName+")") {
      extra += logLine
    }
    var missing []string
    for _, r := range refs {
      if !strings.Contains(existing, "`"+r+"`") {
        missing = append(missing, r)
      }
    }
    extra += referencesLine(missing)
    if extra != "" {
      sep := ""
      if !strings.HasSuffix(existing, "\n") {
        sep = "\n"
      }
      _ = os.WriteFile(fullPath, append(b, []byte(sep+"\n"+extra)...), 0644)
    }
  }

  indexPath := filepath.Join(demandsDir, "INDEX.md")
  if _, err := os.Stat(indexPath); os.IsNotExist(err) {
    header := "# Índice de Demandas — forgeloop\n\n| Data | Stack | Demanda | Arquivo | Status |\n| :--- | :--- | :--- | :--- | :--- |\n"
    _ = os.WriteFile(indexPath, []byte(header), 0644)
  }

  cleanTask := strings.ReplaceAll(task, "|", "-")
  row := fmt.Sprintf("| %s | %s | %s | [`%s`](%s) | `IN_PROGRESS` |\n",
    datePrefix, stackName, cleanTask, fileName, fileName)

  if f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_WRONLY, 0644); err == nil {
    defer f.Close()
    _, _ = f.WriteString(row)
  }

  return filepath.ToSlash(fullPath)
}

func updateDemandStatus(demandPath, status string, attempts int, tokens TokenStats) {
  if demandPath == "" {
    return
  }
  updateIndexStatus(demandPath, status)
  b, err := os.ReadFile(demandPath)
  if err != nil {
    return
  }
  content := strings.Replace(string(b), "- **Status:** `IN_PROGRESS`", fmt.Sprintf("- **Status:** `%s`", status), 1)
  content = strings.Replace(content, "- **Tentativas:** `0`", fmt.Sprintf("- **Tentativas:** `%d`", attempts), 1)
  content = strings.Replace(content, "- **Tokens Totais:** `0`", fmt.Sprintf("- **Tokens Totais:** `%s`", tokens.Format()), 1)
  _ = os.WriteFile(demandPath, []byte(content), 0644)
}

// updateIndexStatus localiza no INDEX.md (mesma pasta demands/ da demanda) as linhas da demanda
// ainda marcadas como IN_PROGRESS e troca pelo novo status. Aceita links relativos ao índice
// ("(arquivo.md)") e o formato legado "(demands/arquivo.md)".
func updateIndexStatus(demandPath, status string) {
  if strings.TrimSpace(status) == "" {
    return
  }
  fileName := filepath.Base(demandPath)
  indexPath := filepath.Join(filepath.Dir(demandPath), "INDEX.md")
  b, err := os.ReadFile(indexPath)
  if err != nil {
    return
  }
  lines := strings.Split(string(b), "\n")
  changed := false
  for i, line := range lines {
    linked := strings.Contains(line, "("+fileName+")") || strings.Contains(line, "(demands/"+fileName+")")
    if linked && strings.Contains(line, "`IN_PROGRESS`") {
      lines[i] = strings.Replace(line, "`IN_PROGRESS`", fmt.Sprintf("`%s`", status), 1)
      changed = true
    }
  }
  if changed {
    _ = os.WriteFile(indexPath, []byte(strings.Join(lines, "\n")), 0644)
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
    logEvent("ETAPA 4A: executando %s: %s", c.label, c.cmd)
    var output string
    var err error
    t0 := time.Now()
    withSpinner(fmt.Sprintf("ETAPA 4A: QUENCH TESTS | %s", c.cmd), func() {
      output, err = runShellCommand(c.cmd)
    })
    fmt.Print(output)
    status := "PASSED"
    if err != nil {
      status = "FAILED"
      allPassed = false
    }
    logBlock(fmt.Sprintf("ETAPA 4A — %s (%s) — %s [%s]", c.label, c.cmd, status, formatElapsed(time.Since(t0))), output)
    report.WriteString(fmt.Sprintf("## %s (`%s`) — %s\n\n```text\n%s\n```\n\n", c.label, c.cmd, status, output))
  }

  header := fmt.Sprintf("# Relatório de Verificação Determinística (Etapa 4A)\n\nALL_CHECKS_PASSED: %v\nStack: %s\nTimestamp: %s\n\n",
    allPassed, stack.Name, time.Now().Format(time.RFC3339))

  _ = os.WriteFile(".forge/state/VERIFY_REPORT.md", []byte(header+report.String()), 0644)
  logEvent("ETAPA 4A: ALL_CHECKS_PASSED=%v", allPassed)
  return allPassed
}

func agentLabel(a AgentBinding) string {
  stage := a.Forge
  if a.Stage != "" {
    stage = a.Forge + ": " + a.Stage
  }
  return fmt.Sprintf("%s — %s | %s", stage, a.Alias, a.Model)
}

// runAgent executa o agente (com spinner no console) e registra prompt, resposta completa e duração no log.
func runAgent(agent AgentBinding, prompt string, tryJSON bool) (TokenStats, error) {
  label := agentLabel(agent)
  start := time.Now()
  logBlock("PROMPT ENVIADO — "+label, prompt)

  stats, text, err := runAgentCore(agent, label, prompt, tryJSON)

  logBlock("RESPOSTA RECEBIDA — "+label, text)
  logEvent("%s concluído em %s (erro: %v)", label, formatElapsed(time.Since(start)), err)
  return stats, err
}

func runAgentCore(agent AgentBinding, label, prompt string, tryJSON bool) (TokenStats, string, error) {
  baseArgs := []string{}
  if strings.TrimSpace(agent.Model) != "" {
    baseArgs = append(baseArgs, "--model", agent.Model)
  }
  baseArgs = append(baseArgs, strings.Fields(agent.Args)...)
  baseArgs = append(baseArgs, "-p", prompt)

  // A saída do agente é sempre capturada (nunca transmitida ao vivo) enquanto o spinner redesenha a linha;
  // ela só é impressa depois que o spinner limpa a linha.
  run := func(args []string) (out string, err error) {
    withSpinner(label, func() { out, err = execCapture(agent.Bin, args) })
    return out, err
  }

  if tryJSON {
    jsonArgs := append(append([]string{}, baseArgs...), "--output-format", "json")
    rawOut, err := run(jsonArgs)
    if err == nil {
      // Comando executou com sucesso: NUNCA reexecutar (evita trabalho duplicado e efeitos colaterais).
      text, stats, hasUsage, ok := parseJSONUsage(rawOut)
      if !ok || text == "" {
        text = rawOut
      }
      fmt.Println(text)
      if !ok || !hasUsage {
        stats = extractOrEstimateTokens(prompt, text)
      }
      return stats, text, nil
    }
    if !isUnsupportedFlagError(rawOut) {
      // Falha real do agente (não é incompatibilidade de flag): não reexecutar.
      fmt.Print(rawOut)
      return extractOrEstimateTokens(prompt, rawOut), rawOut, err
    }
    // Flag --output-format não suportada pelo binário: fallback único para modo texto.
    logEvent("%s: --output-format não suportado; fallback único para modo texto", label)
  }

  rawOut, err := run(baseArgs)
  fmt.Print(rawOut)
  return extractOrEstimateTokens(prompt, rawOut), rawOut, err
}

var unsupportedFlagRe = regexp.MustCompile(`(?i)(unknown|unrecognized|invalid|unexpected)\s+(option|flag|argument)|flag provided but not defined|no such option`)

func isUnsupportedFlagError(output string) bool {
  return unsupportedFlagRe.MatchString(output)
}

// execCapture executa o comando capturando stdout+stderr (sem escrever no console).
func execCapture(bin string, args []string) (string, error) {
  cmd := exec.Command(bin, args...)
  var buf bytes.Buffer
  cmd.Stdout = &buf
  cmd.Stderr = &buf
  err := cmd.Run()
  return buf.String(), err
}

func parseJSONUsage(raw string) (text string, stats TokenStats, hasUsage bool, ok bool) {
  var candidatePayload map[string]any
  var firstValidPayload map[string]any

  targetKeys := []string{"result", "response", "content", "text", "message", "usage", "usageMetadata"}

  for i := 0; i < len(raw); i++ {
    if raw[i] != '{' {
      continue
    }
    dec := json.NewDecoder(strings.NewReader(raw[i:]))
    var obj map[string]any
    if err := dec.Decode(&obj); err != nil {
      continue
    }
    hasKey := false
    for _, k := range targetKeys {
      if _, exists := obj[k]; exists {
        hasKey = true
        break
      }
    }
    if hasKey {
      candidatePayload = obj
      break
    }
    if firstValidPayload == nil {
      firstValidPayload = obj
    }
    if offset := dec.InputOffset(); offset > 0 {
      i += int(offset) - 1
    }
  }

  payload := candidatePayload
  if payload == nil {
    payload = firstValidPayload
  }
  if payload == nil {
    return "", TokenStats{}, false, false
  }

  for _, key := range []string{"result", "response", "content", "text", "message"} {
    if v, isStr := payload[key].(string); isStr && v != "" {
      text = v
      break
    }
  }

  usage, isMap := payload["usage"].(map[string]any)
  if !isMap {
    usage, isMap = payload["usageMetadata"].(map[string]any)
  }
  if isMap {
    stats.Input = intFromMap(usage, "input_tokens", "prompt_tokens", "promptTokenCount")
    stats.Output = intFromMap(usage, "output_tokens", "completion_tokens", "candidatesTokenCount")
    // Claude CLI reporta o prompt em cache à parte; somá-lo reflete o consumo total de entrada.
    stats.Input += intFromMap(usage, "cache_creation_input_tokens") + intFromMap(usage, "cache_read_input_tokens")
    stats.Total = intFromMap(usage, "total_tokens", "totalTokenCount")
    if stats.Total == 0 {
      stats.Total = stats.Input + stats.Output
    }
    hasUsage = stats.Total > 0
  }

  return text, stats, hasUsage, true
}

func intFromMap(m map[string]any, keys ...string) int {
  for _, k := range keys {
    if val, ok := m[k].(float64); ok {
      return int(val)
    }
  }
  return 0
}

var (
  tokenInRe  = regexp.MustCompile(`(?i)(?:^|[^a-z0-9_])(?:input|prompt)[_\s-]*tokens["'\s:=]+(\d{1,3}(?:,\d{3})+|\d+)`)
  tokenOutRe = regexp.MustCompile(`(?i)(?:^|[^a-z0-9_])(?:output|completion|candidates)[_\s-]*tokens["'\s:=]+(\d{1,3}(?:,\d{3})+|\d+)`)
)

func extractOrEstimateTokens(prompt, output string) TokenStats {
  mIn := tokenInRe.FindStringSubmatch(output)
  mOut := tokenOutRe.FindStringSubmatch(output)
  if mIn != nil && mOut != nil {
    inVal, errIn := strconv.Atoi(strings.ReplaceAll(mIn[1], ",", ""))
    outVal, errOut := strconv.Atoi(strings.ReplaceAll(mOut[1], ",", ""))
    if errIn == nil && errOut == nil {
      return TokenStats{Input: inVal, Output: outVal, Total: inVal + outVal, Estimated: false}
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
  cmd.Stdout = &buf
  cmd.Stderr = &buf
  err := cmd.Run()
  return buf.String(), err
}

const (
  telemetryDir   = ".forge/telemetry"
  runsCSVPath    = telemetryDir + "/runs.csv"
  eventsCSVPath  = telemetryDir + "/events.csv"
  noDemandMarker = "(via .forge/state/PLAN.md)"
)

var (
  runsCSVHeader = []string{"RunID", "Timestamp", "Stack", "DemandFile", "MaxAttempts", "TotalAttempts",
    "TotalTokensIn", "TotalTokensOut", "TotalTokens", "FinalStatus", "Note"}
  eventsCSVHeader = []string{"RunID", "Timestamp", "Stage", "Agent", "Model", "Attempt",
    "TokensIn", "TokensOut", "TotalTokens", "DurationSeconds"}
)

// newRunID gera o identificador único da execução corrente (timestamp em nanossegundos).
func newRunID() string {
  return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// appendCSVRecord acrescenta uma linha ao CSV em path (usando encoding/csv para o escape correto),
// criando o diretório e escrevendo o cabeçalho quando o arquivo ainda não existe ou está vazio.
func appendCSVRecord(path string, header, record []string) error {
  if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
    return err
  }
  f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil {
    return err
  }
  w := csv.NewWriter(f)
  if fi, statErr := f.Stat(); statErr == nil && fi.Size() == 0 {
    _ = w.Write(header)
  }
  _ = w.Write(record)
  w.Flush()
  if err := w.Error(); err != nil {
    _ = f.Close()
    return err
  }
  return f.Close()
}

// recordRunTelemetry registra em .forge/telemetry/runs.csv o consolidado de uma execução da Forja
// (uma linha por demanda completa). Falhas de escrita não interrompem a Forja.
func recordRunTelemetry(runID, stack, demandFile string, maxAttempts, attempts int, tokens TokenStats, status, note string) {
  if demandFile == "" {
    demandFile = noDemandMarker
  }
  record := []string{
    runID, time.Now().Format(time.RFC3339), stack, demandFile,
    strconv.Itoa(maxAttempts), strconv.Itoa(attempts),
    strconv.Itoa(tokens.Input), strconv.Itoa(tokens.Output), strconv.Itoa(tokens.Total),
    status, note,
  }
  if err := appendCSVRecord(runsCSVPath, runsCSVHeader, record); err != nil {
    fmt.Printf("[AVISO] Falha ao gravar %s: %v\n", runsCSVPath, err)
  }
}

// recordEventTelemetry registra em .forge/telemetry/events.csv uma linha por etapa/agente executado.
// duration é o tempo de execução da etapa, em segundos.
func recordEventTelemetry(runID, stage, agent, model string, attempt int, tokens TokenStats, duration float64) {
  record := []string{
    runID, time.Now().Format(time.RFC3339), stage, agent, model, strconv.Itoa(attempt),
    strconv.Itoa(tokens.Input), strconv.Itoa(tokens.Output), strconv.Itoa(tokens.Total),
    strconv.FormatFloat(duration, 'f', 3, 64),
  }
  if err := appendCSVRecord(eventsCSVPath, eventsCSVHeader, record); err != nil {
    fmt.Printf("[AVISO] Falha ao gravar %s: %v\n", eventsCSVPath, err)
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
    if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) {
      b.WriteRune(r)
      lastDash = false
    } else if !lastDash {
      b.WriteRune('-')
      lastDash = true
    }
  }
  res := strings.Trim(b.String(), "-")
  if runes := []rune(res); len(runes) > 45 {
    res = strings.Trim(string(runes[:45]), "-")
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
  logEvent("BANNER: %s | %s", title, strings.ReplaceAll(subtitle, "\n", " / "))
}

func fatal(format string, args ...any) {
  msg := fmt.Sprintf(format, args...)
  fmt.Fprintf(os.Stderr, "\n[ERRO FATAL] %s\n", msg)
  logEvent("ERRO FATAL: %s", msg)
  finish(1)
}

// finish registra o código de saída, informa o caminho do log e encerra o processo.
func finish(code int) {
  if runLog != nil {
    logEvent("Código de saída final: %d", code)
    fmt.Printf("\n[LOG] Registro completo salvo em: %s\n", runLog.Path)
    runLog.Close()
    runLog = nil
  }
  os.Exit(code)
}

func reportTokens(a AgentBinding, s TokenStats) {
  fmt.Printf("[TOKENS %s - %s] %s\n", a.Forge, a.Alias, s.Format())
  logEvent("TOKENS %s - %s: %s", a.Forge, a.Alias, s.Format())
}

// ----------------------------------------------------------------------------
// Estrutura de pastas e log de execução
// ----------------------------------------------------------------------------

// stringList é um flag repetível (-ref a -ref b).
type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(v string) error {
  *l = append(*l, v)
  return nil
}

// resolveReferences valida os arquivos passados em -ref. Cada item é aceito como caminho direto
// (relativo ao diretório atual ou absoluto) ou, se não existir, como nome dentro de
// <docsDir>/references/. Retorna os caminhos normalizados (separador "/"), sem duplicatas.
func resolveReferences(refs []string, docsDir string) ([]string, error) {
  var out []string
  seen := map[string]bool{}
  for _, raw := range refs {
    ref := strings.TrimSpace(raw)
    if ref == "" {
      return nil, fmt.Errorf("-ref não pode ser vazio")
    }
    candidates := []string{filepath.Clean(ref)}
    if !filepath.IsAbs(ref) {
      candidates = append(candidates, filepath.Join(docsDir, "references", ref))
    }
    found := ""
    for _, c := range candidates {
      if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
        found = filepath.ToSlash(c)
        break
      }
    }
    if found == "" {
      return nil, fmt.Errorf("arquivo de referência não encontrado: %s (procurado em: %s)", ref, strings.Join(candidates, ", "))
    }
    if !seen[found] {
      seen[found] = true
      out = append(out, found)
    }
  }
  return out, nil
}

// referencesPrompt monta o trecho do prompt do ARCHITECT que aponta para os arquivos de referência.
func referencesPrompt(refs []string) string {
  if len(refs) == 0 {
    return ""
  }
  at := make([]string, len(refs))
  for i, r := range refs {
    at[i] = "@" + r
  }
  return " Material de referência (leia integralmente antes de planejar e respeite-o no PLAN.md e nos testes): " + strings.Join(at, ", ") + "."
}

// ensureForgeDirs cria .forge/{agents,config,state,telemetry}.
func ensureForgeDirs() error {
  for _, sub := range []string{"agents", "config", "state", "telemetry"} {
    if err := os.MkdirAll(filepath.Join(".forge", sub), 0755); err != nil {
      return err
    }
  }
  return nil
}

// ensureDocsDirs cria <docs>/demands, <docs>/logs e <docs>/references (com .gitkeep).
func ensureDocsDirs(docsDir string) error {
  for _, sub := range []string{"demands", "logs", "references"} {
    if err := os.MkdirAll(filepath.Join(docsDir, sub), 0755); err != nil {
      return err
    }
  }
  keep := filepath.Join(docsDir, "references", ".gitkeep")
  if _, err := os.Stat(keep); os.IsNotExist(err) {
    return os.WriteFile(keep, nil, 0644)
  }
  return nil
}

const docsStructureMarker = "## Estrutura de Documentação do forgeloop"

// ensureDocsStructureSection acrescenta ao PROJECT_CONTEXT.md a descrição da estrutura de pastas (idempotente).
func ensureDocsStructureSection(contextPath, docsDir string) error {
  b, err := os.ReadFile(contextPath)
  if err != nil {
    return err
  }
  if strings.Contains(string(b), docsStructureMarker) {
    return nil
  }
  section := fmt.Sprintf("\n%s\n\n"+
    "- `%s/PROJECT_CONTEXT.md` — contexto do projeto (este arquivo).\n"+
    "- `%s/demands/INDEX.md` — índice das demandas (`| Data | Stack | Demanda | Arquivo | Status |`).\n"+
    "- `%s/demands/` — um arquivo `AAAA-MM-DD-<slug>.md` por demanda (descrição, blueprint, status, tentativas e tokens).\n"+
    "- `%s/references/` — material de referência do projeto (specs, links, artigos).\n"+
    "- `%s/logs/` — um log completo por execução: `AAAA-MM-DD_HHMMSS-<slug>.log` (com demanda) ou `AAAA-MM-DD_HHMMSS-forge<N>.log` (sem demanda). "+
    "Cada demanda aponta para o seu log no campo **Log de Execução**.\n",
    docsStructureMarker, docsDir, docsDir, docsDir, docsDir, docsDir)
  sep := ""
  if !strings.HasSuffix(string(b), "\n") {
    sep = "\n"
  }
  return os.WriteFile(contextPath, append(b, []byte(sep+section)...), 0644)
}

type RunLog struct {
  mu   sync.Mutex
  f    *os.File
  Path string // caminho de exibição (barras normais)
  Name string // nome do arquivo
}

var runLog *RunLog

// initRunLog cria <docs>/logs/ e o arquivo AAAA-MM-DD_HHMMSS-<suffix>.log desta execução.
func initRunLog(docsDir, suffix string) (*RunLog, error) {
  logsDir := filepath.Join(docsDir, "logs")
  if err := os.MkdirAll(logsDir, 0755); err != nil {
    return nil, err
  }
  name := fmt.Sprintf("%s-%s.log", time.Now().Format("2006-01-02_150405"), suffix)
  path := filepath.Join(logsDir, name)
  f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
  if err != nil {
    return nil, err
  }
  return &RunLog{f: f, Path: filepath.ToSlash(path), Name: name}, nil
}

func startRunLog(docsDir, suffix string) {
  l, err := initRunLog(docsDir, suffix)
  if err != nil {
    fmt.Fprintf(os.Stderr, "[AVISO] Não foi possível criar o log de execução: %v\n", err)
    return
  }
  runLog = l
  fmt.Printf("[LOG] Registro completo salvo em: %s\n", l.Path)
}

// ansiRe casa sequências de escape ANSI (cores, cursor, limpeza de linha) emitidas por subprocessos como o npx.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func (l *RunLog) write(s string) {
  if l == nil {
    return
  }
  s = ansiRe.ReplaceAllString(s, "")
  l.mu.Lock()
  defer l.mu.Unlock()
  _, _ = l.f.WriteString(s)
}

func (l *RunLog) Close() {
  if l == nil {
    return
  }
  l.mu.Lock()
  defer l.mu.Unlock()
  _ = l.f.Sync()
  _ = l.f.Close()
}

func stamp() string { return time.Now().Format("15:04:05") }

// logEvent grava uma linha com timestamp [HH:MM:SS] (no-op se não houver log ativo).
func logEvent(format string, args ...any) {
  runLog.write(fmt.Sprintf("[%s] %s\n", stamp(), fmt.Sprintf(format, args...)))
}

// logBlock grava um bloco delimitado com o conteúdo integral (sem cortes).
func logBlock(title, content string) {
  if !strings.HasSuffix(content, "\n") {
    content += "\n"
  }
  runLog.write(fmt.Sprintf("[%s] ----- INÍCIO: %s -----\n%s[%s] ----- FIM: %s -----\n", stamp(), title, content, stamp(), title))
}

func describeBinding(a AgentBinding) string {
  return fmt.Sprintf("%s (%s) bin=%q modelo=%q skill=%q args=%q", a.Forge, a.Alias, a.Bin, a.Model, a.Skill, a.Args)
}

func buildRunHeader(stack StackConfig, agents []AgentBinding, task, demandFile string, startStage, maxAttempts int, docsDir string) string {
  var b strings.Builder
  fmt.Fprintf(&b, "Data/Hora: %s\n", time.Now().Format(time.RFC3339))
  fmt.Fprintf(&b, "Stack: %s\n", stack.Name)
  fmt.Fprintf(&b, "Comando de testes: %s\nComando de tipos/vet: %s\nComando de linter: %s\n", stack.TestCmd, stack.TypeCmd, stack.LintCmd)
  fmt.Fprintf(&b, "Etapa inicial: %d | Max tentativas: %d | Docs: %s/\n", startStage, maxAttempts, docsDir)
  b.WriteString("Bindings:\n")
  for _, a := range agents {
    fmt.Fprintf(&b, "  - %s\n", describeBinding(a))
  }
  if task == "" {
    task = "(sem demanda)"
  }
  fmt.Fprintf(&b, "Demanda: %s\n", task)
  if demandFile != "" {
    fmt.Fprintf(&b, "Arquivo da demanda: %s\n", demandFile)
  }
  return b.String()
}

// ----------------------------------------------------------------------------
// Spinner de progresso
// ----------------------------------------------------------------------------

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func isTerminal(f *os.File) bool {
  fi, err := f.Stat()
  return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func formatElapsed(d time.Duration) string {
  secs := int(d.Seconds())
  return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}

// withSpinner executa fn exibindo um spinner no stdout (apenas em TTY; fora dele imprime uma linha estática).
// O spinner escreve somente no console: nunca toca no arquivo de log nem na captura do comando.
func withSpinner(label string, fn func()) {
  withSpinnerTo(os.Stdout, isTerminal(os.Stdout), label, fn)
}

func withSpinnerTo(w io.Writer, tty bool, label string, fn func()) {
  if !tty {
    fmt.Fprintf(w, "[AGUARDE] [%s] Processando...\n", label)
    fn()
    return
  }

  start := time.Now()
  stop := make(chan struct{})
  done := make(chan struct{})
  go func() {
    defer close(done)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    i := 0
    draw := func() {
      fmt.Fprintf(w, "\r\033[K%s [%s] Processando... (%s)", spinnerFrames[i%len(spinnerFrames)], label, formatElapsed(time.Since(start)))
      i++
    }
    draw()
    for {
      select {
      case <-stop:
        fmt.Fprint(w, "\r\033[K")
        return
      case <-ticker.C:
        draw()
      }
    }
  }()
  defer func() {
    close(stop)
    <-done
  }()
  fn()
}

// ----------------------------------------------------------------------------
// Etapa inicial (-stage / -forge)
// ----------------------------------------------------------------------------

// resolveStage consolida -stage (principal) e -forge (alias de compatibilidade).
func resolveStage(stage, forge int, stageSet, forgeSet bool) (int, error) {
  if stageSet && forgeSet && stage != forge {
    return 0, fmt.Errorf("-stage (%d) e -forge (%d) divergem; use apenas -stage", stage, forge)
  }
  v := stage
  if !stageSet && forgeSet {
    v = forge
  }
  if v < 1 || v > 3 {
    return 0, fmt.Errorf("etapa inicial inválida: %d (use 1, 2 ou 3)", v)
  }
  return v, nil
}

// ----------------------------------------------------------------------------
// -init: binários, skills e estrutura de pastas
// ----------------------------------------------------------------------------

type BinStatus struct {
  Name  string
  Path  string
  Found bool
}

// checkBinaries verifica a disponibilidade de cada binário (sem duplicatas, na ordem informada).
func checkBinaries(names []string, lookPath func(string) (string, error)) []BinStatus {
  seen := map[string]bool{}
  var res []BinStatus
  for _, n := range names {
    n = strings.TrimSpace(n)
    if n == "" || seen[n] {
      continue
    }
    seen[n] = true
    path, err := lookPath(n)
    res = append(res, BinStatus{Name: n, Path: path, Found: err == nil})
  }
  return res
}

var skillEnvRe = regexp.MustCompile(`^FORGE_[A-Z0-9]+_SKILL$`)

// collectConfiguredSkills retorna as skills únicas (ordenadas) definidas em variáveis FORGE_*_SKILL.
func collectConfiguredSkills(environ []string) []string {
  seen := map[string]bool{}
  var skills []string
  for _, kv := range environ {
    parts := strings.SplitN(kv, "=", 2)
    if len(parts) != 2 || !skillEnvRe.MatchString(parts[0]) {
      continue
    }
    v := strings.TrimSpace(parts[1])
    if v == "" || seen[v] {
      continue
    }
    seen[v] = true
    skills = append(skills, v)
  }
  sort.Strings(skills)
  return skills
}

// skillShortName extrai o nome da skill de um identificador "dono/repositório[@versão]".
func skillShortName(id string) string {
  id = strings.TrimSpace(id)
  if i := strings.LastIndex(id, "/"); i >= 0 {
    id = id[i+1:]
  }
  if i := strings.Index(id, "@"); i >= 0 {
    id = id[:i]
  }
  return strings.TrimSuffix(id, ".git")
}

// skillSearchRoots lista as pastas onde uma skill pode estar instalada (locais e globais).
func skillSearchRoots(baseDir, home string) []string {
  roots := []string{filepath.Join(baseDir, "skills"), filepath.Join(baseDir, ".agents", "skills")}
  if home != "" {
    roots = append(roots, filepath.Join(home, ".claude", "skills"), filepath.Join(home, ".antigravity", "skills"))
  }
  return roots
}

// skillAliases mapeia o nome curto de uma skill configurada para o nome com que o
// `npx skills add` a registra/instala (a pasta ou a chave do lock diferem do nome do repositório).
var skillAliases = map[string]string{
  "security-audit-skill":    "security-audit",
  "superpowers":             "writing-plans",
  "elephant-goldfish-model": "egm-review",
}

// skillInstalled considera a skill presente se existir uma pasta com o seu nome (ou alias) em alguma
// raiz, ou se algum lock registrar o identificador como origem ou o nome (ou alias) como chave.
// lockPaths aceita vários arquivos separados por os.PathListSeparator
// (ex.: skills-lock.json local e ~/.agents/.skill-lock.json global).
func skillInstalled(skill string, roots []string, lockPaths string) bool {
  name := skillShortName(skill)
  if name == "" {
    return false
  }
  names := []string{name}
  if alias, ok := skillAliases[strings.ToLower(name)]; ok {
    names = append(names, alias)
  }
  for _, r := range roots {
    for _, n := range names {
      if fi, err := os.Stat(filepath.Join(r, n)); err == nil && fi.IsDir() {
        return true
      }
    }
  }
  for _, lockPath := range filepath.SplitList(lockPaths) {
    b, err := os.ReadFile(lockPath)
    if err != nil {
      continue
    }
    var lock struct {
      Skills map[string]struct {
        Source string `json:"source"`
      } `json:"skills"`
    }
    if json.Unmarshal(b, &lock) != nil {
      continue
    }
    for k, v := range lock.Skills {
      if strings.EqualFold(v.Source, skill) {
        return true
      }
      for _, n := range names {
        if strings.EqualFold(k, n) {
          return true
        }
      }
    }
  }
  return false
}

func missingSkills(skills, roots []string, lockPath string) []string {
  var missing []string
  for _, sk := range skills {
    if !skillInstalled(sk, roots, lockPath) {
      missing = append(missing, sk)
    }
  }
  return missing
}

type SkillResult struct {
  Skill  string
  Action string // "presente", "instalada" ou "falha"
  Output string
  Err    error
}

// ensureSkills instala (via install) as skills ausentes, ou todas quando force=true.
func ensureSkills(skills, roots []string, lockPath string, force bool, install func(string) (string, error)) []SkillResult {
  var res []SkillResult
  for _, sk := range skills {
    if !force && skillInstalled(sk, roots, lockPath) {
      res = append(res, SkillResult{Skill: sk, Action: "presente"})
      continue
    }
    out, err := install(sk)
    r := SkillResult{Skill: sk, Action: "instalada", Output: out, Err: err}
    if err != nil {
      r.Action = "falha"
    }
    res = append(res, r)
  }
  return res
}

func skillInstallArgs(skill string) []string {
  return []string{"skills", "add", skill, "-g", "-y"}
}

// stageSkills devolve as skills realmente necessárias a partir da etapa inicial.
func stageSkills(startStage int, architect, skeptic, sentinel AgentBinding) []string {
  var list []string
  if startStage <= 1 {
    list = append(list, architect.Skill)
  }
  if startStage <= 2 {
    list = append(list, skeptic.Skill)
  }
  list = append(list, sentinel.Skill)
  seen := map[string]bool{}
  var out []string
  for _, sk := range list {
    sk = strings.TrimSpace(sk)
    if sk != "" && !seen[sk] {
      seen[sk] = true
      out = append(out, sk)
    }
  }
  return out
}

func currentSkillRoots() ([]string, string) {
  cwd, _ := os.Getwd()
  home, _ := os.UserHomeDir()
  locks := "skills-lock.json"
  if home != "" {
    locks += string(os.PathListSeparator) + filepath.Join(home, ".agents", ".skill-lock.json")
  }
  return skillSearchRoots(cwd, home), locks
}

// preflightSkills avisa (sem bloquear) se alguma skill configurada não está instalada.
func preflightSkills(skills []string) {
  roots, lock := currentSkillRoots()
  missing := missingSkills(skills, roots, lock)
  if len(missing) == 0 {
    return
  }
  msg := fmt.Sprintf("Skills configuradas e não instaladas: %s. Rode `./forgeloop -init` para instalá-las.", strings.Join(missing, ", "))
  fmt.Printf("\n[AVISO] %s\n", msg)
  logEvent("AVISO (pre-flight): %s", msg)
}

func runInit(docsDir string, agents []AgentBinding, forceSkills bool) {
  printBanner("FORGELOOP -init: AMBIENTE E SKILLS", "Binários, skills (npx skills add) e estrutura de pastas")
  code := 0

  // (d) Estrutura de pastas e log primeiro, para que tudo abaixo fique registrado.
  if err := ensureForgeDirs(); err != nil {
    fatal("Erro ao criar .forge/: %v", err)
  }
  if err := ensureDocsDirs(docsDir); err != nil {
    fatal("Erro ao criar estrutura de pastas em %s: %v", docsDir, err)
  }
  startRunLog(docsDir, "init")
  logBlock("CABEÇALHO DA EXECUÇÃO (-init)", fmt.Sprintf("Data/Hora: %s\nDocs: %s/\nForce skills: %v\n", time.Now().Format(time.RFC3339), docsDir, forceSkills))
  fmt.Printf("[OK] Pastas prontas: .forge/{agents,config,state}/, %s/demands/, %s/logs/, %s/references/\n", docsDir, docsDir, docsDir)
  logEvent("Pastas prontas: .forge/{agents,config,state}/, %s/demands/, %s/logs/, %s/references/", docsDir, docsDir, docsDir)

  // (a) Binários
  names := []string{}
  for _, a := range agents {
    names = append(names, a.Bin)
  }
  names = append(names, "npx", "git")
  fmt.Println("\nBinários:")
  npxOK := false
  for _, b := range checkBinaries(names, exec.LookPath) {
    if b.Found {
      fmt.Printf("  [OK]     %s (%s)\n", b.Name, b.Path)
      logEvent("[OK] binário %s (%s)", b.Name, b.Path)
      if b.Name == "npx" {
        npxOK = true
      }
    } else {
      fmt.Printf("  [AUSENTE] %s\n", b.Name)
      logEvent("[AUSENTE] binário %s", b.Name)
      code = 1
    }
  }

  // (b) Skills configuradas
  skills := collectConfiguredSkills(os.Environ())
  if len(skills) == 0 {
    for _, a := range agents {
      if sk := strings.TrimSpace(a.Skill); sk != "" {
        skills = append(skills, sk)
      }
    }
    skills = collectConfiguredSkills(skillEnvFrom(skills))
  }
  fmt.Printf("\nSkills configuradas (%d): %s\n", len(skills), strings.Join(skills, ", "))
  logEvent("Skills configuradas: %s", strings.Join(skills, ", "))

  // (c) Instalação das ausentes (ou de todas, com -force-skills)
  roots, lock := currentSkillRoots()
  install := func(skill string) (out string, err error) {
    if !npxOK {
      return "", fmt.Errorf("npx ausente no PATH")
    }
    args := skillInstallArgs(skill)
    fmt.Printf("  -> npx %s\n", strings.Join(args, " "))
    logEvent("Executando: npx %s", strings.Join(args, " "))
    withSpinner("npx "+strings.Join(args, " "), func() { out, err = execCapture("npx", args) })
    fmt.Print(out)
    logBlock("npx "+strings.Join(args, " "), out)
    return out, err
  }
  for _, r := range ensureSkills(skills, roots, lock, forceSkills, install) {
    switch r.Action {
    case "presente":
      fmt.Printf("  [OK]     %s já instalada\n", r.Skill)
    case "instalada":
      fmt.Printf("  [OK]     %s instalada/atualizada\n", r.Skill)
    default:
      fmt.Printf("  [FALHA]  %s: %v\n", r.Skill, r.Err)
      code = 1
    }
    logEvent("Skill %s: %s (erro: %v)", r.Skill, r.Action, r.Err)
  }

  if code == 0 {
    printBanner("-init CONCLUÍDO", "Ambiente pronto. Use ./forgeloop -setup (projeto existente) ou ./forgeloop -task \"...\".")
  } else {
    printBanner("-init CONCLUÍDO COM PENDÊNCIAS", "Há binários ausentes ou skills com falha. Corrija e rode ./forgeloop -init novamente.")
  }
  finish(code)
}

// skillEnvFrom monta pares FORGE_n_SKILL=valor para reaproveitar collectConfiguredSkills (deduplicação/ordem).
func skillEnvFrom(skills []string) []string {
  env := make([]string, 0, len(skills))
  for i, sk := range skills {
    env = append(env, fmt.Sprintf("FORGE_%d_SKILL=%s", i, sk))
  }
  return env
}
