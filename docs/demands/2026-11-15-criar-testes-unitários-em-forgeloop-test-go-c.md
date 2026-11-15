# Demanda: Criar testes unitários em forgeloop_test.go cobrindo slugify, parseJSONUsage, extractOrEstimateTokens e updateIndexStatus

- **Data:** 2026-11-15 17:12
- **Stack:** Go
- **Bindings:** `F1:opus | F2:GPT-OSS 120B (Medium) | F3:Gemini 3.8 Flash (High) | F4B:Claude Sonnet 4.6 (Thinking) | F5:Claude Sonnet 4.6 (Thinking)`
- **Status:** `APPROVED`
- **Tentativas:** `1`
- **Tokens Totais:** `402262 in / 57205 out (459467 total)`
- **Detalhamento por Etapa:** `F1(ARCHITECT): 24 in / 23576 out (23600 total) | F2(SKEPTIC): 210500 in / 5539 out (216039 total) | F3#1(BUILDER): 113155 in / 19027 out (132182 total) | F4B#1(SENTINEL): 37656 in / 4240 out (41896 total) | F5#1(ARBITER): 40927 in / 4823 out (45750 total)`
- **Log de Execução:** [`2026-11-15_171203-criar-testes-unitários-em-forgeloop-test-go-c.log`](../logs/2026-11-15_171203-criar-testes-unitários-em-forgeloop-test-go-c.log)

## Descrição Original

Criar testes unitários em forgeloop_test.go cobrindo slugify, parseJSONUsage, extractOrEstimateTokens e updateIndexStatus

## Blueprint & Critérios de Aceite

**Plano completo:** `.forge/PLAN.md` · **Matriz:** `.forge/TEST_CASES.md` · **Testes:** `forgeloop_test.go`

### Resumo técnico
As 4 funções já existem em `forgeloop.go`. Os testes antigos (4 funções, ~12 asserções) foram substituídos por uma suíte table-driven com ~60 cenários (happy path, edge cases, erros). A maior parte **caracteriza** o comportamento atual (GREEN). Durante a especificação, 4 defeitos reais viraram regras novas (RED):

| Task | Função | Regra nova (RED) | Defeito corrigido |
| :- | :- | :- | :- |
| 1 | `slugify` | aceitar `unicode.IsMark` | texto NFD (macOS) quebrava palavras: `decisão` → `decisa-o` |
| 2 | `parseJSONUsage` | varrer cada `{` com `json.Decoder`; preferir objeto com chaves conhecidas | ruído com chaves no stderr mesclado invalidava o JSON → caía na estimativa |
| 3 | `extractOrEstimateTokens` | captura `\d{1,3}(?:,\d{3})+\|\d+` e remover vírgulas | `12,345` era lido como `12` |
| 4 | `updateIndexStatus` | `strings.TrimSpace(status) == ""` → no-op | status vazio gravava ``` `` ``` no índice |

Assinaturas inalteradas. Sem dependências novas. Apenas `forgeloop.go` deve ser alterado na Etapa 3.

### Critérios de Aceite
- [ ] `go vet ./...` limpo.
- [ ] `go test -v -race ./...` → 0 FAIL (suíte inteira).
- [ ] `forgeloop_test.go` inalterado pelo BUILDER.

### Estado RED registrado (Etapa 1)
`go test -race ./...` → 92 PASS, 4 testes FAIL: `TestSlugifyKeepsCombiningMarks`, `TestParseJSONUsageToleratesBraceNoise` (4 subtestes), `TestExtractOrEstimateTokensThousandsSeparator`, `TestUpdateIndexStatusRejectsBlankStatus`. Todos falham por asserção (comportamento ausente), sem erros de compilação ou import.
