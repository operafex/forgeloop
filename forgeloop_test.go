package main

import (
  "bytes"
  "os"
  "path/filepath"
  "regexp"
  "strings"
  "testing"
  "time"
)

func TestSlugify(t *testing.T) {
  cases := map[string]string{
    "Criar API de Usuários": "criar-api-de-usuários",
    "  --Olá,   Mundo!!  ":  "olá-mundo",
    "":                      "demanda",
    "!!!":                   "demanda",
  }
  for in, want := range cases {
    if got := slugify(in); got != want {
      t.Errorf("slugify(%q) = %q; want %q", in, got, want)
    }
  }
  if got := []rune(slugify(strings.Repeat("ã", 100))); len(got) > 45 {
    t.Errorf("slugify não cortou em 45 runes: %d", len(got))
  }
}

func TestParseJSONUsage(t *testing.T) {
  _, s, has, ok := parseJSONUsage(`{"result":"oi","usage":{"input_tokens":10,"output_tokens":5}}`)
  if !ok || !has || s.Total != 15 {
    t.Errorf("usage: %+v has=%v ok=%v", s, has, ok)
  }
  _, s, has, ok = parseJSONUsage(`{"response":"oi","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`)
  if !ok || !has || s.Input != 10 || s.Output != 5 || s.Total != 15 {
    t.Errorf("usageMetadata: %+v has=%v ok=%v", s, has, ok)
  }
  _, s, _, _ = parseJSONUsage(`{"usage":{"input_tokens":1,"output_tokens":2},"usageMetadata":{"totalTokenCount":99}}`)
  if s.Total != 3 {
    t.Errorf("usage deve ter precedência: %+v", s)
  }
  if _, _, _, ok = parseJSONUsage("texto sem json"); ok {
    t.Error("texto sem JSON não deveria ser ok")
  }
}

func TestExtractOrEstimateTokens(t *testing.T) {
  s := extractOrEstimateTokens("p", `{"cache_creation_input_tokens": 9000, "input_tokens": 12, "output_tokens": 34}`)
  if s.Input != 12 || s.Output != 34 || s.Total != 46 || s.Estimated {
    t.Errorf("cache_creation_input_tokens capturado indevidamente: %+v", s)
  }
  s = extractOrEstimateTokens("p", "input_tokens: 99999999999999999999999 output_tokens: 1")
  if !s.Estimated || s.Total < 0 {
    t.Errorf("overflow deveria cair na estimativa: %+v", s)
  }
  s = extractOrEstimateTokens("p", "sem métricas")
  if !s.Estimated || s.Total <= 0 {
    t.Errorf("estimativa esperada: %+v", s)
  }
}

const indexHeader = "# Índice\n\n| Data | Stack | Demanda | Arquivo | Status |\n| :--- | :--- | :--- | :--- | :--- |\n"

func TestUpdateIndexStatus(t *testing.T) {
  dir := t.TempDir()
  if err := ensureDocsDirs(dir); err != nil {
    t.Fatal(err)
  }
  demand := filepath.Join(dir, "demands", "2026-01-01-x.md")
  index := filepath.Join(dir, "DEMANDS_INDEX.md")
  rows := "| 2026-01-01 | Go | x | [`2026-01-01-x.md`](demands/2026-01-01-x.md) | `IN_PROGRESS` |\n" +
    "| 2026-01-01 | Go | y | [`2026-01-01-y.md`](demands/2026-01-01-y.md) | `IN_PROGRESS` |\n"
  _ = os.WriteFile(index, []byte(indexHeader+rows), 0644)

  updateIndexStatus(demand, "APPROVED")
  b, _ := os.ReadFile(index)
  got := string(b)
  if !strings.Contains(got, "(demands/2026-01-01-x.md) | `APPROVED` |") {
    t.Errorf("linha da demanda x não atualizada:\n%s", got)
  }
  if !strings.Contains(got, "(demands/2026-01-01-y.md) | `IN_PROGRESS` |") {
    t.Errorf("linha da demanda y não deveria mudar:\n%s", got)
  }
}

func TestEnsureDocsDirs(t *testing.T) {
  dir := filepath.Join(t.TempDir(), "docs")
  if err := ensureDocsDirs(dir); err != nil {
    t.Fatal(err)
  }
  for _, sub := range []string{"demands", "logs"} {
    if fi, err := os.Stat(filepath.Join(dir, sub)); err != nil || !fi.IsDir() {
      t.Errorf("pasta %s não criada: %v", sub, err)
    }
  }
}

func TestRunLogFileCreationAndContent(t *testing.T) {
  dir := filepath.Join(t.TempDir(), "docs")
  l, err := initRunLog(dir, "minha-demanda")
  if err != nil {
    t.Fatal(err)
  }
  if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}_\d{6}-minha-demanda\.log$`).MatchString(l.Name) {
    t.Errorf("nome de log inesperado: %s", l.Name)
  }
  if want := filepath.ToSlash(filepath.Join(dir, "logs", l.Name)); want != l.Path {
    t.Errorf("Path = %s; want %s", l.Path, want)
  }

  prev := runLog
  runLog = l
  defer func() { runLog = prev }()

  full := strings.Repeat("linha longa de resposta\n", 500)
  logEvent("evento %d", 42)
  logBlock("RESPOSTA", full)
  logBlock("SEM-NEWLINE", "abc")
  l.Close()

  b, err := os.ReadFile(filepath.Join(dir, "logs", l.Name))
  if err != nil {
    t.Fatal(err)
  }
  got := string(b)
  if !regexp.MustCompile(`(?m)^\[\d{2}:\d{2}:\d{2}\] evento 42$`).MatchString(got) {
    t.Errorf("evento sem timestamp:\n%s", got)
  }
  if !strings.Contains(got, full) {
    t.Error("resposta completa deveria estar no log sem cortes")
  }
  if !strings.Contains(got, "INÍCIO: SEM-NEWLINE") || !strings.Contains(got, "FIM: SEM-NEWLINE") {
    t.Errorf("delimitadores ausentes:\n%s", got)
  }
  if strings.Contains(got, "\033") {
    t.Error("log não deve conter códigos ANSI")
  }
}

func TestLogFunctionsNoopWithoutActiveLog(t *testing.T) {
  prev := runLog
  runLog = nil
  defer func() { runLog = prev }()
  logEvent("nada")
  logBlock("nada", "nada")
}

func TestRegisterDemandLinksLog(t *testing.T) {
  dir := filepath.Join(t.TempDir(), "docs")
  logName := "2026-11-15_163000-criar-x.log"
  demandPath := registerDemandInDocs(dir, "Go", "Criar x", "F1:opus", logName)

  b, err := os.ReadFile(demandPath)
  if err != nil {
    t.Fatal(err)
  }
  want := "- **Log de Execução:** [`" + logName + "`](../logs/" + logName + ")"
  if !strings.Contains(string(b), want) {
    t.Errorf("vínculo do log ausente em %s:\n%s", demandPath, b)
  }
  if !strings.Contains(string(b), "- **Detalhamento por Etapa:** `pendente`") || !strings.Contains(string(b), "## Descrição Original") {
    t.Errorf("estrutura da demanda alterada:\n%s", b)
  }

  idx, _ := os.ReadFile(filepath.Join(dir, "DEMANDS_INDEX.md"))
  lines := strings.Split(strings.TrimSpace(string(idx)), "\n")
  if !strings.HasPrefix(lines[2], "| Data | Stack | Demanda | Arquivo | Status |") {
    t.Errorf("cabeçalho do índice alterado: %q", lines[2])
  }
  row := lines[len(lines)-1]
  if n := strings.Count(row, "|"); n != 6 { // 5 colunas
    t.Errorf("linha do índice deve ter 5 colunas, tem %d separadores: %q", n, row)
  }
  if strings.Contains(row, "logs/") {
    t.Errorf("índice não deve citar o log: %q", row)
  }

  // Status da demanda continua sincronizado com o índice.
  updateDemandStatus(demandPath, "APPROVED", 1, TokenStats{Total: 1}, nil)
  idx, _ = os.ReadFile(filepath.Join(dir, "DEMANDS_INDEX.md"))
  if !strings.Contains(string(idx), "`APPROVED`") {
    t.Errorf("índice não sincronizado:\n%s", idx)
  }
}

func TestRegisterDemandExistingFileGetsNewLogLink(t *testing.T) {
  dir := filepath.Join(t.TempDir(), "docs")
  p := registerDemandInDocs(dir, "Go", "Mesma demanda", "m", "log-a.log")
  registerDemandInDocs(dir, "Go", "Mesma demanda", "m", "log-b.log")
  registerDemandInDocs(dir, "Go", "Mesma demanda", "m", "log-b.log") // idempotente
  b, _ := os.ReadFile(p)
  if !strings.Contains(string(b), "(../logs/log-a.log)") || strings.Count(string(b), "(../logs/log-b.log)") != 1 {
    t.Errorf("vínculos de log incorretos:\n%s", b)
  }
}

func TestRegisterDemandWithoutLogOmitsField(t *testing.T) {
  dir := filepath.Join(t.TempDir(), "docs")
  p := registerDemandInDocs(dir, "Go", "Sem log", "m", "")
  b, _ := os.ReadFile(p)
  if strings.Contains(string(b), "Log de Execução") {
    t.Errorf("campo de log não deveria existir:\n%s", b)
  }
}

func TestEnsureDocsStructureSection(t *testing.T) {
  dir := t.TempDir()
  ctx := filepath.Join(dir, "PROJECT_CONTEXT.md")
  _ = os.WriteFile(ctx, []byte("# Contexto"), 0644)
  for i := 0; i < 2; i++ { // idempotente
    if err := ensureDocsStructureSection(ctx, "docs"); err != nil {
      t.Fatal(err)
    }
  }
  b, _ := os.ReadFile(ctx)
  got := string(b)
  if strings.Count(got, docsStructureMarker) != 1 {
    t.Errorf("seção deveria aparecer uma vez:\n%s", got)
  }
  for _, want := range []string{"docs/demands/", "docs/logs/", "DEMANDS_INDEX.md", "Log de Execução"} {
    if !strings.Contains(got, want) {
      t.Errorf("seção sem %q:\n%s", want, got)
    }
  }
  if err := ensureDocsStructureSection(filepath.Join(dir, "nao-existe.md"), "docs"); err == nil {
    t.Error("esperava erro para arquivo inexistente")
  }
}

func TestWithSpinnerTTY(t *testing.T) {
  var buf bytes.Buffer
  ran := false
  withSpinnerTo(&buf, true, "ETAPA 1: BLUEPRINT — ARCHITECT | opus", func() {
    ran = true
    time.Sleep(250 * time.Millisecond)
  })
  out := buf.String()
  if !ran {
    t.Fatal("fn não executou")
  }
  if !strings.Contains(out, "[ETAPA 1: BLUEPRINT — ARCHITECT | opus] Processando... (00:0") {
    t.Errorf("linha do spinner ausente: %q", out)
  }
  if !strings.Contains(out, "\r\033[K") || !strings.HasSuffix(out, "\r\033[K") {
    t.Errorf("deveria redesenhar na mesma linha e limpar ao final: %q", out)
  }
  if !strings.Contains(out, spinnerFrames[0]) || !strings.Contains(out, spinnerFrames[1]) {
    t.Errorf("frames Braille ausentes: %q", out)
  }
}

func TestWithSpinnerNonTTY(t *testing.T) {
  var buf bytes.Buffer
  withSpinnerTo(&buf, false, "ETAPA 3", func() { time.Sleep(150 * time.Millisecond) })
  out := buf.String()
  if out != "[AGUARDE] [ETAPA 3] Processando...\n" {
    t.Errorf("saída inesperada: %q", out)
  }
  if strings.Contains(out, "\033") || strings.Contains(out, "\r") {
    t.Error("fora de TTY não pode haver ANSI/CR")
  }
}

func TestSpinnerDoesNotInterfereWithCapture(t *testing.T) {
  var buf bytes.Buffer
  var out string
  withSpinnerTo(&buf, true, "x", func() {
    out, _ = runShellCommand("echo olá")
  })
  if strings.TrimSpace(out) != "olá" || strings.Contains(out, "\033") {
    t.Errorf("captura contaminada: %q", out)
  }
}

func TestFormatElapsed(t *testing.T) {
  if got := formatElapsed(74 * time.Second); got != "01:14" {
    t.Errorf("got %s", got)
  }
  if got := formatElapsed(0); got != "00:00" {
    t.Errorf("got %s", got)
  }
}

func TestAgentLabelAndHeader(t *testing.T) {
  a := AgentBinding{Forge: "ETAPA 1", Stage: "BLUEPRINT", Alias: "ARCHITECT", Bin: "claude", Model: "opus", Skill: "superpowers"}
  if got := agentLabel(a); got != "ETAPA 1: BLUEPRINT — ARCHITECT | opus" {
    t.Errorf("label = %q", got)
  }
  h := buildRunHeader(StackConfig{Name: "Go", TestCmd: "go test ./..."}, []AgentBinding{a}, "", "", 1, 3, "docs")
  for _, want := range []string{"Stack: Go", "go test ./...", `modelo="opus"`, `skill="superpowers"`, "Demanda: (sem demanda)"} {
    if !strings.Contains(h, want) {
      t.Errorf("cabeçalho sem %q:\n%s", want, h)
    }
  }
}

func TestRunAgentLogsPromptAndFullResponse(t *testing.T) {
  dir := filepath.Join(t.TempDir(), "docs")
  l, err := initRunLog(dir, "forge3")
  if err != nil {
    t.Fatal(err)
  }
  prev := runLog
  runLog = l
  defer func() { runLog = prev }()

  // "echo" faz o papel do binário do agente: devolve os argumentos (inclui o prompt) como resposta.
  a := AgentBinding{Forge: "ETAPA 3", Stage: "STRIKE", Alias: "BUILDER", Bin: "echo", Model: "m1"}
  stats, err := runAgent(a, "PROMPT-UNICO-123", false)
  l.Close()
  if err != nil || stats.Total <= 0 {
    t.Fatalf("stats=%+v err=%v", stats, err)
  }
  b, _ := os.ReadFile(filepath.Join(dir, "logs", l.Name))
  got := string(b)
  for _, want := range []string{"PROMPT ENVIADO — ETAPA 3: STRIKE — BUILDER | m1", "RESPOSTA RECEBIDA", "--model m1 -p PROMPT-UNICO-123", "concluído em"} {
    if !strings.Contains(got, want) {
      t.Errorf("log sem %q:\n%s", want, got)
    }
  }
  if strings.Contains(got, "\033") {
    t.Error("log com ANSI")
  }
}

func TestCollectConfiguredSkills(t *testing.T) {
  env := []string{
    "FORGE_ARCHITECT_SKILL=obra/superpowers",
    "FORGE_SKEPTIC_SKILL=google/elephant-goldfish-model",
    "FORGE_SENTINEL_SKILL=cloudflare/security-audit-skill",
    "FORGE_OTHER_SKILL=obra/superpowers", // duplicada
    "FORGE_BUILDER_SKILL=  ",             // vazia
    "FORGE_BUILDER_MODEL=Gemini",         // não é skill
    "SKILL=x",
    "FORGE_SKILL_EXTRA=y",
    "PATH=/usr/bin",
  }
  got := collectConfiguredSkills(env)
  want := []string{"cloudflare/security-audit-skill", "google/elephant-goldfish-model", "obra/superpowers"}
  if strings.Join(got, ",") != strings.Join(want, ",") {
    t.Errorf("got %v; want %v", got, want)
  }
  if len(collectConfiguredSkills(nil)) != 0 {
    t.Error("sem env deveria retornar vazio")
  }
}

func TestSkillShortName(t *testing.T) {
  cases := map[string]string{
    "obra/superpowers":        "superpowers",
    "superpowers":             "superpowers",
    "a/b/c.git":               "c",
    "owner/repo@1.2.3":        "repo",
    "  cloudflare/sec-skill ": "sec-skill",
    "":                        "",
  }
  for in, want := range cases {
    if got := skillShortName(in); got != want {
      t.Errorf("skillShortName(%q) = %q; want %q", in, got, want)
    }
  }
}

func TestSkillInstalledInRoots(t *testing.T) {
  base, home := t.TempDir(), t.TempDir()
  roots := skillSearchRoots(base, home)
  if len(roots) != 4 {
    t.Fatalf("esperava 4 raízes, veio %v", roots)
  }
  for _, want := range []string{
    filepath.Join(base, "skills"), filepath.Join(base, ".agents", "skills"),
    filepath.Join(home, ".claude", "skills"), filepath.Join(home, ".antigravity", "skills"),
  } {
    found := false
    for _, r := range roots {
      found = found || r == want
    }
    if !found {
      t.Errorf("raiz ausente: %s", want)
    }
  }

  if skillInstalled("obra/superpowers", roots, "") {
    t.Error("nada instalado ainda")
  }
  for i, r := range roots {
    name := []string{"superpowers", "elephant-goldfish-model", "security-audit-skill", "outra"}[i]
    if err := os.MkdirAll(filepath.Join(r, name), 0755); err != nil {
      t.Fatal(err)
    }
  }
  for _, sk := range []string{"obra/superpowers", "google/elephant-goldfish-model", "cloudflare/security-audit-skill"} {
    if !skillInstalled(sk, roots, "") {
      t.Errorf("%s deveria estar instalada", sk)
    }
  }
  // arquivo com o mesmo nome (não pasta) não conta
  _ = os.WriteFile(filepath.Join(base, "skills", "so-arquivo"), []byte("x"), 0644)
  if skillInstalled("a/so-arquivo", roots, "") {
    t.Error("arquivo não deveria contar como skill")
  }
}

func TestSkillInstalledViaLockFile(t *testing.T) {
  dir := t.TempDir()
  lock := filepath.Join(dir, "skills-lock.json")
  _ = os.WriteFile(lock, []byte(`{"version":2,"skills":{"ponytail":{"source":"DietrichGebert/ponytail"}}}`), 0644)
  if !skillInstalled("dietrichgebert/ponytail", nil, lock) {
    t.Error("source do lock deveria casar (case-insensitive)")
  }
  if skillInstalled("obra/superpowers", nil, lock) {
    t.Error("skill fora do lock não deveria casar")
  }
  _ = os.WriteFile(lock, []byte("{quebrado"), 0644)
  if skillInstalled("obra/superpowers", nil, lock) {
    t.Error("lock inválido deve ser ignorado")
  }
}

func TestMissingSkills(t *testing.T) {
  base := t.TempDir()
  roots := skillSearchRoots(base, "")
  _ = os.MkdirAll(filepath.Join(base, "skills", "superpowers"), 0755)
  got := missingSkills([]string{"obra/superpowers", "google/elephant-goldfish-model"}, roots, "")
  if len(got) != 1 || got[0] != "google/elephant-goldfish-model" {
    t.Errorf("missing = %v", got)
  }
}

func TestEnsureSkillsInstallsOnlyMissing(t *testing.T) {
  base := t.TempDir()
  roots := skillSearchRoots(base, "")
  _ = os.MkdirAll(filepath.Join(base, "skills", "superpowers"), 0755)

  var called []string
  install := func(s string) (string, error) {
    called = append(called, s)
    if s == "x/falha" {
      return "boom", os.ErrInvalid
    }
    return "ok", nil
  }
  skills := []string{"obra/superpowers", "google/elephant-goldfish-model", "x/falha"}
  res := ensureSkills(skills, roots, "", false, install)
  if strings.Join(called, ",") != "google/elephant-goldfish-model,x/falha" {
    t.Errorf("instalou: %v", called)
  }
  actions := []string{res[0].Action, res[1].Action, res[2].Action}
  if strings.Join(actions, ",") != "presente,instalada,falha" || res[2].Err == nil {
    t.Errorf("ações: %v", actions)
  }

  called = nil
  ensureSkills(skills, roots, "", true, install) // -force-skills
  if len(called) != 3 {
    t.Errorf("force deveria reinstalar todas: %v", called)
  }
}

func TestSkillInstallArgs(t *testing.T) {
  if got := strings.Join(skillInstallArgs("obra/superpowers"), " "); got != "skills add obra/superpowers -g -y" {
    t.Errorf("args = %q", got)
  }
}

func TestCheckBinaries(t *testing.T) {
  lp := func(n string) (string, error) {
    if n == "agy" {
      return "", os.ErrNotExist
    }
    return "/bin/" + n, nil
  }
  got := checkBinaries([]string{"claude", "agy", "claude", "", "npx", "git"}, lp)
  if len(got) != 4 {
    t.Fatalf("deveria deduplicar: %+v", got)
  }
  if !got[0].Found || got[0].Path != "/bin/claude" || got[1].Found || got[1].Name != "agy" {
    t.Errorf("status incorreto: %+v", got)
  }
}

func TestStageSkills(t *testing.T) {
  a := AgentBinding{Skill: "obra/superpowers"}
  s := AgentBinding{Skill: "google/elephant-goldfish-model"}
  se := AgentBinding{Skill: "cloudflare/security-audit-skill"}
  if got := stageSkills(1, a, s, se); len(got) != 3 {
    t.Errorf("etapa 1: %v", got)
  }
  if got := stageSkills(2, a, s, se); len(got) != 2 || got[0] != "google/elephant-goldfish-model" {
    t.Errorf("etapa 2: %v", got)
  }
  if got := stageSkills(3, a, s, se); len(got) != 1 || got[0] != "cloudflare/security-audit-skill" {
    t.Errorf("etapa 3: %v", got)
  }
}

func TestResolveStage(t *testing.T) {
  cases := []struct {
    stage, forge     int
    stageSet, forgeSet bool
    want             int
    wantErr          bool
  }{
    {1, 1, false, false, 1, false},
    {3, 1, true, false, 3, false},
    {1, 2, false, true, 2, false}, // alias -forge
    {2, 2, true, true, 2, false},
    {2, 3, true, true, 0, true}, // divergentes
    {4, 1, true, false, 0, true},
    {1, 0, false, true, 0, true},
  }
  for _, c := range cases {
    got, err := resolveStage(c.stage, c.forge, c.stageSet, c.forgeSet)
    if (err != nil) != c.wantErr || (!c.wantErr && got != c.want) {
      t.Errorf("resolveStage(%+v) = %d, %v", c, got, err)
    }
  }
}

func TestPreflightSkillsWarnsAndLogs(t *testing.T) {
  dir := filepath.Join(t.TempDir(), "docs")
  l, err := initRunLog(dir, "pre")
  if err != nil {
    t.Fatal(err)
  }
  prev := runLog
  runLog = l
  defer func() { runLog = prev }()
  preflightSkills([]string{"zzz/skill-que-nao-existe-xyz"})
  l.Close()
  b, _ := os.ReadFile(filepath.Join(dir, "logs", l.Name))
  if !strings.Contains(string(b), "./forgeloop -init") || !strings.Contains(string(b), "zzz/skill-que-nao-existe-xyz") {
    t.Errorf("aviso não registrado:\n%s", b)
  }
}

func TestAgentLabelUsesEtapa(t *testing.T) {
  a := AgentBinding{Forge: "ETAPA 3", Stage: "STRIKE", Alias: "BUILDER", Model: "m"}
  if got := agentLabel(a); got != "ETAPA 3: STRIKE — BUILDER | m" {
    t.Errorf("label = %q", got)
  }
}
