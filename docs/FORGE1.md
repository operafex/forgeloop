**`forgeloop`** é uma arquitetura aberta (licenciada sob **Apache License 2.0**) de **Loop Engineering** orientada a especificações e testes (*Spec & Test-Driven Loop*), escrita em Go puro (zero dependências externas) e autenticada 100% via OAuth local (sem chaves de API).

Cada etapa do ciclo produtivo é tratada como uma **Forge** (*Forja*), desacoplando **apelidos funcionais** dos modelos e *skills* do momento para que você possa trocar de LLM ou *skill* editando apenas um arquivo `.env`:

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│ FORGE 0 — DISCOVERY [Opcional: -setup para projetos já existentes]          │
│ Agente: ONBOARDER | Mapeia docs atuais -> gera /docs/PROJECT_CONTEXT.md     │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ FORGE 1 — BLUEPRINT                                                         │
│ Agente: ARCHITECT | Skill: Spec-TDD-Planner (superpowers)                   │
│ Quebra em Steps atômicos -> gera .forge/PLAN.md + Testes em estado RED      │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ FORGE 2 — TEMPER                                                            │
│ Agente: SKEPTIC | Skill: Cold-Global-Lens (elephant-goldfish-model)         │
│ Audita clareza a frio + invariantes globais -> .forge/SPEC_REVIEW.md        │
└──────────────────┬───────────────────────────────────┬──────────────────────┘
                   │ (SPEC_STATUS: REVISION_REQUIRED)  │ (SPEC_STATUS: APPROVED)
                   ▼                                   ▼
       Devolve à FORGE 1               ┌──────────────────────────────────────┐
                                       │ FORGE 3 — STRIKE                     │
  ┌───────────────────────────────────>│ Agente: BUILDER (Contexto Limpo)     │
  │                                    │ Implementa src/ até sair de RED->GREEN│
  │                                    └──────────────────┬───────────────────┘
  │                                                       ▼
  │                                    ┌──────────────────────────────────────┐
  │                                    │ FORGE 4 — QUENCH                     │
  │                                    │ 4A: Runner Stack (RSpec/Go/Node)     │
  │                                    │ 4B: SENTINEL | Sec-Scanner (AppSec)  │
  │                                    └──────────────────┬───────────────────┘
  │                                                       ▼
  │                                    ┌──────────────────────────────────────┐
  │                                    │ FORGE 5 — ASSAY                      │
  │  (RETRY_EXECUTION & ATTEMPT < 3)   │ Agente: ARBITER | Skill: Diff-Judge  │
  └────────────────────────────────────┴────┬─────────────────────────────┬───┘
                                            │ (APPROVE_MERGE)             │ (ESCALATE / AMEND)
                                            ▼                             ▼
                                    Libera Merge na Branch       Circuit Breaker (Para)

```

---

### Tabela de Vínculo (Bindings Atuais via OAuth)

| Etapa | Apelido do Agente | Apelido da Skill | Vínculo Atual (Skill) | Vínculo Atual (CLI & Modelo OAuth) |
| --- | --- | --- | --- | --- |
| **Forge 0: Discovery** | **ONBOARDER** | **Repo-Mapper** | Inspeção de código + docs existentes | `claude --model opus` *(Claude Team)* |
| **Forge 1: Blueprint** | **ARCHITECT** | **Spec-TDD-Planner** | `obra/superpowers` (`writing-plans` + `test-driven-development`) | `claude --model opus` *(Claude Team)* |
| **Forge 2: Temper** | **SKEPTIC** | **Cold-Global-Lens** | `elephant-goldfish-model` | `agy --model "GPT-OSS 120B (Medium)"` |
| **Forge 3: Strike** | **BUILDER** | **Strict-Implementer** | Regras idiomáticas por stack (`03_BUILDER.md`) | `agy --model "Gemini 3.8 Flash"` |
| **Forge 4: Quench** | **SENTINEL** | **Sec-Scanner** | **4A:** CLI da Stack<br>

<br>**4B:** `cloudflare/security-audit-skill` | **4A:** `rspec` / `go test` / `npm test`<br>

<br>**4B:** `agy --model "Claude Sonnet 4.6 (Thinking)"` |
| **Forge 5: Assay** | **ARBITER** | **Diff-Judge** | Rubrica semântica + *Circuit Breaker* | `agy --model "Claude Sonnet 4.6 (Thinking)"` |

---

## Passo 1: Criar o Repositório `forgeloop`, `go.mod`, `.gitignore` e `LICENSE`

Execute os comandos abaixo no terminal para estruturar o repositório:

```bash
mkdir -p forgeloop/.forge forgeloop/docs/demands
cd forgeloop
git init

```

### 1.1 Arquivo `go.mod`

```go
module forgeloop

go 1.22

```

### 1.2 Arquivo `.gitignore`

```gitignore
# Artefatos transitórios de cada execução da Forge
.forge/PLAN.md
.forge/TEST_CASES.md
.forge/SPEC_REVIEW.md
.forge/VERIFY_REPORT.md
.forge/SECURITY_REPORT.md
.forge/DECISION.md
.forge/project.env

```

### 1.3 Arquivo `LICENSE` (Apache License 2.0)

```text
                                 Apache License
                           Version 2.0, January 2004
                        http://www.apache.org/licenses/

   TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

   1. Definitions.

      "License" shall mean the terms and conditions for use, reproduction,
      and distribution as defined by Sections 1 through 9 of this document.

      "Licensor" shall mean the copyright owner or entity authorized by
      the copyright owner that is granting the License.

      "Legal Entity" shall mean the union of the acting entity and all
      other entities that control, are controlled by, or are under common
      control with that entity. For the purposes of this definition,
      "control" means (i) the power, direct or indirect, to cause the
      direction or management of such entity, whether by contract or
      otherwise, or (ii) ownership of fifty percent (50%) or more of the
      outstanding shares, or (iii) beneficial ownership of such entity.

      "You" (or "Your") shall mean an individual or Legal Entity
      exercising permissions granted by this License.

      "Source" form shall mean the preferred form for making modifications,
      including but not limited to software source code, documentation
      source, and configuration files.

      "Object" form shall mean any form resulting from mechanical
      transformation or translation of a Source form, including but
      not limited to compiled object code, generated documentation,
      and conversions to other media types.

      "Work" shall mean the work of authorship, whether in Source or
      Object form, made available under the License, as indicated by a
      copyright notice that is included in or attached to the work
      (an example is provided in the Appendix below).

      "Derivative Works" shall mean any work, whether in Source or Object
      form, that is based on (or derived from) the Work and for which the
      editorial revisions, annotations, elaborations, or other modifications
      represent, as a whole, an original work of authorship. For the purposes
      of this License, Derivative Works shall not include works that remain
      separable from, or merely link (or bind by name) to the interfaces of,
      the Work and Derivative Works thereof.

      "Contribution" shall mean any work of authorship, including
      the original version of the Work and any modifications or additions
      to that Work or Derivative Works thereof, that is intentionally
      submitted to Licensor for inclusion in the Work by the copyright owner
      or by an individual or Legal Entity authorized to submit on behalf of
      the copyright owner. For the purposes of this definition, "submitted"
      means any form of electronic, verbal, or written communication sent
      to the Licensor or its representatives, including but not limited to
      communication on electronic mailing lists, source code control systems,
      and issue tracking systems that are managed by, or on behalf of, the
      Licensor for the purpose of discussing and improving the Work, but
      excluding communication that is conspicuously marked or otherwise
      designated in writing by the copyright owner as "Not a Contribution."

      "Contributor" shall mean Licensor and any individual or Legal Entity
      on behalf of whom a Contribution has been received by Licensor and
      subsequently incorporated within the Work.

   2. Grant of Copyright License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      copyright license to reproduce, prepare Derivative Works of,
      publicly display, publicly perform, sublicense, and distribute the
      Work and such Derivative Works in Source or Object form.

   3. Grant of Patent License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      (except as stated in this section) patent license to make, have made,
      use, offer to sell, sell, import, and otherwise transfer the Work,
      where such license applies only to those patent claims licensable
      by such Contributor that are necessarily infringed by their
      Contribution(s) alone or by combination of their Contribution(s)
      with the Work to which such Contribution(s) was submitted. If You
      institute patent litigation against any entity (including a
      cross-claim or counterclaim in a lawsuit) alleging that the Work
      or a Contribution incorporated within the Work constitutes direct
      or contributory patent infringement, then any patent licenses
      granted to You under this License for that Work shall terminate
      as of the date such litigation is filed.

   4. Redistribution. You may reproduce and distribute copies of the
      Work or Derivative Works thereof in any medium, with or without
      modifications, and in Source or Object form, provided that You
      meet the following conditions:

      (a) You must give any other recipients of the Work or
          Derivative Works a copy of this License; and

      (b) You must cause any modified files to carry prominent notices
          stating that You changed the files; and

      (c) You must retain, in the Source form of any Derivative Works
          that You distribute, all copyright, patent, trademark, and
          attribution notices from the Source form of the Work,
          excluding those notices that do not pertain to any part of
          the Derivative Works; and

      (d) If the Work includes a "NOTICE" text file as part of its
          distribution, then any Derivative Works that You distribute must
          include a readable copy of the attribution notices contained
          within such NOTICE file, excluding those notices that do not
          pertain to any part of the Derivative Works, in at least one
          of the following places: within a NOTICE text file distributed
          as part of the Derivative Works; within the Source form or
          documentation, if provided along with the Derivative Works; or,
          within a display generated by the Derivative Works, if and
          wherever such third-party notices normally appear. The contents
          of the NOTICE file are for informational purposes only and
          do not modify the License. You may add Your own attribution
          notices within Derivative Works that You distribute, alongside
          or as an addendum to the NOTICE text file from the Work, provided
          that such additional attribution notices cannot be construed
          as modifying the License.

      You may add Your own copyright statement to Your modifications and
      may provide additional or different license terms and conditions
      for use, reproduction, or distribution of Your modifications, or
      for any such Derivative Works as a whole, provided Your use,
      reproduction, and distribution of the Work otherwise complies with
      the conditions stated in this License.

   5. Submission of Contributions. Unless You explicitly state otherwise,
      any Contribution intentionally submitted for inclusion in the Work
      by You to the Licensor shall be under the terms and conditions of
      this License, without any additional terms or conditions.
      Notwithstanding the above, nothing herein shall supersede or modify
      the terms of any separate license agreement you may have executed
      with Licensor regarding such Contributions.

   6. Trademarks. This License does not grant permission to use the trade
      names, trademarks, service marks, or product names of the Licensor,
      except as required for reasonable and customary use in describing the
      origin of the Work and reproducing the content of the NOTICE file.

   7. Disclaimer of Warranty. Unless required by applicable law or
      agreed to in writing, Licensor provides the Work (and each
      Contributor provides its Contributions) on an "AS IS" BASIS,
      WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
      implied, including, without limitation, any warranties or conditions
      of TITLE, NON-INFRINGEMENT, MERCHANTABILITY, or FITNESS FOR A
      PARTICULAR PURPOSE. You are solely responsible for determining the
      appropriateness of using or redistributing the Work and assume any
      risks associated with Your exercise of permissions under this License.

   8. Limitation of Liability. In no event and under no theory of
      liability, whether in contract, tort (including negligence), or
      otherwise, unless required by applicable law (such as deliberate
      and grossly negligent acts) or agreed to in writing, shall any
      Contributor be liable to You for damages, including any direct,
      indirect, special, incidental, or consequential damages of any
      character arising as a result of this License or out of the use or
      inability to use the Work (including but not limited to damages for
      loss of goodwill, work stoppage, computer failure or malfunction, or
      any and all other commercial damages or losses), even if such
      Contributor has been advised of the possibility of such damages.

   9. Accepting Warranty or Additional Liability. While redistributing
      the Work or Derivative Works thereof, You may choose to offer,
      and charge a fee for, acceptance of support, warranty, indemnity,
      or other liability obligations and/or rights consistent with this
      License. However, in accepting such obligations, You may act only
      on Your own behalf and on Your sole responsibility, not on behalf
      of any other Contributor, and only if You agree to indemnify,
      defend, and hold each Contributor harmless for any liability
      incurred by, or claims asserted against, such Contributor by reason
      of your accepting any such warranty or additional liability.

   END OF TERMS AND CONDITIONS

```

---

## Passo 2: Autenticação OAuth e Instalação das Skills Leves (Zero Docker)

1. **Autentique os CLIs oficiais via navegador (sem chaves de API):**
* `claude` → Faça login com sua conta **Claude Team**.
* `agy` → Faça login com sua conta Google no **Antigravity CLI**.


2. **Instale as 3 skills leves em Markdown (sem containers ou bancos externos):**
* **Spec-TDD-Planner (`obra/superpowers`):**
```bash
npx skills add obra/superpowers -g

```


*(Instala os workflows `writing-plans` e `test-driven-development` diretamente na pasta global de skills, sem Docker nem Postgres).*
* **Cold-Global-Lens (`elephant-goldfish-model`):** Instale no diretório global de skills (`~/.antigravity/skills/` ou via `npx skills add google/elephant-goldfish-model -g`).
* **Sec-Scanner (`cloudflare/security-audit-skill`):**
```bash
npx skills add cloudflare/security-audit-skill -g

```





---

## Passo 3: Criar a Matriz de Modelos (`.forge/models.env`)

Salve este arquivo em `.forge/models.env`. Sempre que quiser testar novos modelos ou trocar uma *skill* que ficou obsoleta, basta editar estas variáveis:

```env
# ==============================================================================
# FORGELOOP — VARIÁVEIS DE CONTROLE DE MODELOS E SKILLS (ZERO API KEY)
# Edite este arquivo para trocar ou testar modelos sem alterar o código Go.
# ==============================================================================

# FORGE 0 — DISCOVERY (Onboarding de projetos já existentes)
FORGE_ONBOARDER_BIN=claude
FORGE_ONBOARDER_MODEL=opus

# FORGE 1 — BLUEPRINT (Planejamento incremental + criação de testes RED)
FORGE_ARCHITECT_BIN=claude
FORGE_ARCHITECT_MODEL=opus
FORGE_ARCHITECT_SKILL=superpowers

# FORGE 2 — TEMPER (Revisão crítica a frio + invariantes globais)
FORGE_SKEPTIC_BIN=agy
FORGE_SKEPTIC_MODEL=GPT-OSS 120B (Medium)
FORGE_SKEPTIC_SKILL=elephant-goldfish-model

# FORGE 3 — STRIKE (Execução rápida em contexto higienizado)
FORGE_BUILDER_BIN=agy
FORGE_BUILDER_MODEL=Gemini 3.8 Flash

# FORGE 4B — QUENCH APPSEC (Auditoria de segurança no git diff)
FORGE_SENTINEL_BIN=agy
FORGE_SENTINEL_MODEL=Claude Sonnet 4.6 (Thinking)
FORGE_SENTINEL_SKILL=cloudflare/security-audit-skill

# FORGE 5 — ASSAY (Juiz semântico final + Circuit Breaker)
FORGE_ARBITER_BIN=agy
FORGE_ARBITER_MODEL=Claude Sonnet 4.6 (Thinking)

# CONTROLE DO LOOP & DOCUMENTAÇÃO
FORGE_MAX_ATTEMPTS=3
FORGE_DEFAULT_DOCS_DIR=docs

```

---

## Passo 4: Criar os Contratos das 6 Forges em `.forge/`

### 4.1 `.forge/00_ONBOARDER.md` (Forge 0 — Discovery)

```markdown
# Forge 0: DISCOVERY — Agente ONBOARDER
# Capacidade Ativa: Repo-Mapper (Claude Code Setup)

Você está analisando um **repositório já existente** para preparar o ambiente do `forgeloop` antes que qualquer nova demanda seja planejada ou codificada.

## Regras Estritas de Isolamento
1. **Somente Leitura no Código:** Nunca altere código de produção ou testes existentes.
2. **Saída Restrita:** Crie ou atualize apenas `<DOCS_DIR>/PROJECT_CONTEXT.md` e `<DOCS_DIR>/DEMANDS_INDEX.md`.

## Fluxo Obrigatório de Discovery
1. **Inspeção da Documentação Existente:**
   - Leia os arquivos indicados pelo usuário no caminho de documentação informado (`<EXISTING_DOCS_PATH>`), além de `README.md`, arquivos de configuração e schemas de banco/API.
2. **Mapeamento Arquitetural do Código:**
   - Identifique estrutura de pastas, padrões de arquitetura, dependências principais, autenticação, banco de dados e suíte de testes atual.
3. **Geração do Contexto Base (`<DOCS_DIR>/PROJECT_CONTEXT.md`):**
   - Consolide em um guia objetivo para as próximas Forges:
     - **Visão Geral & Stack:** Linguagem, frameworks, versão e comandos de build/teste.
     - **Mapa de Diretórios:** Onde ficam models, regras de negócio, controllers/handlers e testes.
     - **Convenções & Regras de Ouro:** Padrões obrigatórios do projeto e armadilhas a evitar.
     - **Referências de Documentação:** Onde está a documentação original do sistema.
4. **Inicialização do Índice (`<DOCS_DIR>/DEMANDS_INDEX.md`):**
   - Se ainda não existir, crie a tabela inicial para registrar as próximas demandas.
5. **Handoff:** Encerre declarando: `FORGE_0_DONE — Contexto mapeado em <DOCS_DIR>/PROJECT_CONTEXT.md.`

```

### 4.2 `.forge/01_ARCHITECT.md` (Forge 1 — Blueprint; copiar também para `CLAUDE.md`)

```markdown
# Forge 1: BLUEPRINT — Agente ARCHITECT
# Capacidade Ativa: Spec-TDD-Planner (superpowers: writing-plans + test-driven-development)

Você é o arquiteto e especificador TDD. Sua missão é transformar a demanda registrada em um plano de passos atômicos e criar os testes em estado RED antes que qualquer código de produção exista.

## Leis de Ferro (Superpowers TDD & Planejamento)
1. **NENHUM CÓDIGO DE PRODUÇÃO SEM TESTE FALHANDO ANTES:** Nunca crie ou altere arquivos em `src/`, `app/`, `lib/`, `pkg/`, `internal/` ou `cmd/`.
2. **Escrita Restrita:** Escreva apenas em `.forge/PLAN.md`, `.forge/TEST_CASES.md`, no arquivo de demanda em `<DOCS_DIR>/demands/` e nos arquivos de teste (`spec/`, `tests/`, `__tests__/`, `*_test.go`, `*.test.*`, `*.spec.*`).
3. **Mentalidade TDD (Estado RED):** Todo teste novo deve falhar exclusivamente pela ausência da implementação, nunca por erro de sintaxe ou import quebrado.

## Fluxo Obrigatório (Metodologia Superpowers)
1. **Leitura de Contexto:** Se existir `<DOCS_DIR>/PROJECT_CONTEXT.md`, leia-o integralmente para respeitar as convenções do projeto existente. Se existir `.forge/SPEC_REVIEW.md` com `SPEC_STATUS: REVISION_REQUIRED`, corrija todos os pontos apontados.
2. **Plano em Passos Atômicos (`.forge/PLAN.md`):** Usando o padrão `superpowers:writing-plans`, divida a entrega em **Tasks com checkboxes (`- [ ]`)**, onde cada passo contém:
   - Arquivos exatos a criar/modificar.
   - Assinaturas exatas (nomes de funções, parâmetros, retornos, structs/interfaces, rotas e status HTTP) — sem ambiguidade.
   - Comportamento esperado e tratamento de erros.
3. **Matriz de Cenários (`.forge/TEST_CASES.md`):** Mapeie *Happy Path*, *Edge Cases* e *Error Handling* vinculados a cada Task.
4. **Implementação dos Testes & Registro:** Escreva os testes automatizados cobrindo 100% dos cenários, rode o runner para confirmar o estado `RED` e anexe o resumo técnico do plano no arquivo da demanda em `<DOCS_DIR>/demands/`.
5. **Handoff:** Encerre declarando: `FORGE_1_DONE — Blueprint e testes RED gerados.`

```

### 4.3 `.forge/02_SKEPTIC.md` (Forge 2 — Temper)

```markdown
# Forge 2: TEMPER — Agente SKEPTIC
# Capacidade Ativa: Cold-Global-Lens (elephant-goldfish-model)

Você é o revisor crítico pré-execução. Sua função é temperar a especificação antes da execução, impedindo que o BUILDER desperdice tokens implementando um plano ambíguo ou testes mal estruturados.

## Regras Estritas de Isolamento
1. **Somente Leitura no Código:** Nunca modifique código de produção nem arquivos de teste.
2. **Saída Única:** Grave seu veredito exclusivamente em `.forge/SPEC_REVIEW.md`.

## Fluxo Obrigatório (Dupla Perspectiva)
1. **Lente Goldfish (Leitura a Frio / Sem Suposições):**
   - Leia `.forge/PLAN.md` e `.forge/TEST_CASES.md` sem assumir nenhum contexto implícito.
   - Um executor literal consegue implementar cada Task sem adivinhar nomes de funções, tipos, rotas ou regras de negócio?
   - Os testes criados na Forge 1 possuem imports, mocks e fixtures válidos, falhando apenas pela falta da implementação?
2. **Lente Elephant (Invariantes Globais do Repositório):**
   - Cruzar com `<DOCS_DIR>/PROJECT_CONTEXT.md` (se existir) e com a stack ativa:
     - **Ruby on Rails:** Conflitos com migrations existentes, callbacks do ActiveRecord, N+1, rotas e validações.
     - **Go:** Quebra de interfaces, *data races*, propagação de `context.Context` e dependências circulares.
     - **TypeScript + Node.js:** Quebra de contratos exportados, schemas (Zod/etc.) e tipagem estrita.

## Veredito (Topo de `.forge/SPEC_REVIEW.md`)
Emita **exatamente um** status seguido da justificativa técnica:
- `SPEC_STATUS: APPROVED`
- `SPEC_STATUS: REVISION_REQUIRED`

```

### 4.4 `.forge/03_BUILDER.md` (Forge 3 — Strike)

```markdown
# Forge 3: STRIKE — Agente BUILDER
# Capacidade Ativa: Strict-Implementer

Você é o engenheiro de implementação operando em **contexto higienizado (stateless)**. Transforme `.forge/PLAN.md` em código funcional que passe nos testes e na auditoria de segurança.

## Regras Estritas de Isolamento
1. **Gate Obrigatório:** Só execute se `.forge/SPEC_REVIEW.md` contiver `SPEC_STATUS: APPROVED`.
2. **PROIBIDO modificar testes:** Nunca altere, comente, pule (`skip`) ou apague arquivos de teste nem `.forge/TEST_CASES.md`.

## Fluxo Obrigatório
1. **Leitura de Estado:** Leia `<DOCS_DIR>/PROJECT_CONTEXT.md` (se existir), `.forge/PLAN.md` e `.forge/TEST_CASES.md`.
2. **Correção Prioritária (Se for rodada > 1):** Leia `.forge/DECISION.md`, `.forge/VERIFY_REPORT.md` e `.forge/SECURITY_REPORT.md`. Resolva todas as falhas apontadas antes de qualquer outra mudança.
3. **Execução Incremental:** Implemente uma Task por vez conforme o `.forge/PLAN.md`, validando os testes até sair de `RED` para `GREEN`.
4. **Padrões de Segurança e Idioma:**
   - Nunca insira segredos/tokens hardcoded, SQL concatenado ou inputs sem sanitização.
   - **Rails:** Respeite *strong parameters* e evite N+1.
   - **Go:** Trate todos os erros (`if err != nil`), propague `context.Context` e evite *race conditions*.
   - **TypeScript + Node.js:** Proibido usar `any` ou `@ts-ignore`.
5. **Handoff:** Encerre declarando: `FORGE_3_DONE — Implementação concluída.`

```

### 4.5 `.forge/04_SENTINEL.md` (Forge 4B — Quench AppSec)

```markdown
# Forge 4B: QUENCH APPSEC — Agente SENTINEL
# Capacidade Ativa: Sec-Scanner (cloudflare/security-audit-skill)

Você é o auditor de segurança automatizado (AppSec). Sua missão é inspecionar o código recém-implementado na Forge 3 antes da avaliação final.

## Regras Estritas de Isolamento
1. **Somente Leitura no Código:** Nunca edite código de produção nem testes.
2. **Saída Única:** Grave seu relatório exclusivamente em `.forge/SECURITY_REPORT.md`.

## Fluxo Obrigatório
1. Execute `git diff` e `git status` para mapear todas as linhas adicionadas ou modificadas nesta tarefa.
2. Aplique a skill de auditoria de segurança configurada focando em:
   - **Injeção & Validação:** SQL Injection, Command Injection, XSS, Path Traversal, SSRF e desserialização insegura.
   - **Autenticação & Controle de Acesso:** Broken Access Control, IDOR, ausência de verificação de escopo/tenant, tokens fracos.
   - **Exposição de Dados & Segredos:** Chaves, senhas, tokens hardcoded, logs vazando PII ou stack traces sensíveis.
   - **Concorrência & Recursos:** *Race conditions*, ausência de *rate limit* ou *timeouts*, DoS por payload ilimitado.
3. **Veredito (Topo de `.forge/SECURITY_REPORT.md`):**
   - `SEC_STATUS: PASS` — Nenhuma vulnerabilidade crítica, alta ou média detectada no `git diff`.
   - `SEC_STATUS: VULNERABILITY_FOUND` — Liste cada vulnerabilidade com severidade (`CRITICAL`/`HIGH`/`MEDIUM`), arquivo, linha e correção exigida.

```

### 4.6 `.forge/05_ARBITER.md` (Forge 5 — Assay)

```markdown
# Forge 5: ASSAY — Agente ARBITER
# Capacidade Ativa: Diff-Judge

Você é o juiz final e *Circuit Breaker* do `forgeloop`. Avalie o resultado técnico, a auditoria de segurança e a qualidade semântica do código antes de liberar o merge.

## Regras de Autoridade
1. **Somente Leitura no Código:** Nunca modifique código de produção nem testes.
2. **Saída Única:** Grave o veredito em `.forge/DECISION.md`.

## Fluxo Obrigatório
1. Leia `.forge/PLAN.md`, `.forge/TEST_CASES.md`, `.forge/VERIFY_REPORT.md`, `.forge/SECURITY_REPORT.md` e execute `git diff`.
2. Confirme que nenhum arquivo de teste foi adulterado na Forge 3.
3. Verifique se há *hardcoding* artificial para enganar testes ou desvio dos contratos do `.forge/PLAN.md`.

## Veredito (Topo de `.forge/DECISION.md`)
Mantenha `ATTEMPT: <número>` e emita **exatamente uma** decisão:
- `DECISION: APPROVE_MERGE` — Testes passaram (`ALL_CHECKS_PASSED: true`), segurança aprovada (`SEC_STATUS: PASS`) e `git diff` limpo.
- `DECISION: RETRY_EXECUTION` — Falha nos testes, falha de segurança (`VULNERABILITY_FOUND`) ou código de baixa qualidade (apenas se `ATTEMPT < 3`). Forneça instruções cirúrgicas de correção.
- `DECISION: AMEND_TEST` — Código correto, mas há erro/contradição no teste criado na Forge 1.
- `DECISION: ESCALATE_ARCHITECT` — `ATTEMPT >= 3` sem convergência ou falha arquitetural grave no plano.

```

---

## Passo 5: Orquestrador em Go (`forgeloop.go`)

Salve o código abaixo como `forgeloop.go` na raiz do repositório. Ele carrega `.forge/models.env`, organiza cada etapa como **Forge 0** a **Forge 5**, contabiliza tokens por Forge e no total, registra as demandas em `/docs/demands/` e alimenta `.forge/TELEMETRY.md`:

```go
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

```

---

## Passo 6: Arquivo `README.md` Oficial do Repositório `forgeloop`

```markdown
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

```

---

## Passo 7: Validar a Compilação e Realizar o Primeiro Commit

Finalize executando os comandos abaixo na raiz de `forgeloop` para compilar o binário local e registrar a base inicial no Git:

```bash
cp .forge/01_ARCHITECT.md CLAUDE.md
go build -o forgeloop forgeloop.go
git add .
git commit -m "chore: initial forgeloop architecture under Apache 2.0"

```

Com o binário `./forgeloop` compilado, você pode simplesmente copiar a pasta `.forge/` e o executável para qualquer repositório em **Ruby on Rails**, **Go** ou **TypeScript + Node.js** e rodar `./forgeloop -setup` (em projetos existentes) ou `./forgeloop -task "..."` (para novas demandas).
