# Etapa 0: DISCOVERY — Agente ONBOARDER
# A Forja (`forgeloop`) é o ciclo completo; este arquivo descreve a ETAPA 0.
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
   - Consolide em um guia objetivo para as próximas Etapas:
     - **Visão Geral & Stack:** Linguagem, frameworks, versão e comandos de build/teste.
     - **Mapa de Diretórios:** Onde ficam models, regras de negócio, controllers/handlers e testes.
     - **Convenções & Regras de Ouro:** Padrões obrigatórios do projeto e armadilhas a evitar.
     - **Referências de Documentação:** Onde está a documentação original do sistema.
4. **Inicialização do Índice (`<DOCS_DIR>/DEMANDS_INDEX.md`):**
   - Se ainda não existir, crie a tabela inicial para registrar as próximas demandas.
5. **Handoff:** Encerre declarando: `FORGE_0_DONE — Contexto mapeado em <DOCS_DIR>/PROJECT_CONTEXT.md.`

