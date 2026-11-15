Apliquei os três ajustes. go vet e go build passam, mas go test -v -race ./... não rodou nenhum teste, porque forgeloop_test.go não existe no diretório.

O que mudou
- Bindings e permissões: FORGE_BUILDER_MODEL agora é Gemini 3.8 Flash (High) no .forge/models.env e no default de forgeloop.go. FORGE_SKEPTIC_ARGS, FORGE_BUILDER_ARGS, FORGE_SENTINEL_ARGS e FORGE_ARB-dangerously-skip-permissions, nos
