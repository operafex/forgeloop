package main

import (
	"errors"
	"fmt"
	"strings"
)

const (
	authModeAPIKey  = "api_key"
	authModeAnother = "another"
)

// resolveAuthMode normaliza e valida o modo de autenticação informado.
// Retorna "api_key" como padrão se raw estiver vazio.
func resolveAuthMode(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return authModeAPIKey, nil
	}
	normalized := strings.ToLower(trimmed)
	switch normalized {
	case authModeAPIKey:
		return authModeAPIKey, nil
	case authModeAnother:
		return authModeAnother, nil
	default:
		return "", fmt.Errorf("FORGE_AUTH_MODE inválido: %q (use %q)", trimmed, authModeAPIKey)
	}
}

// validateAuthConfig valida as configurações de autenticação para o modo selecionado.
func validateAuthConfig(authMode string, getenv func(string) string) error {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	switch authMode {
	case authModeAnother:
		return nil
	case authModeAPIKey:
		if strings.EqualFold(strings.TrimSpace(getenv("FORGE_OMNIROUTE_ENABLED")), "true") {
			return nil
		}
		if strings.TrimSpace(getenv("LLM_API_KEY")) != "" || strings.TrimSpace(getenv("FORGE_API_KEY")) != "" {
			return nil
		}
		return errors.New("API key ausente: defina LLM_API_KEY em .forge/config/models.env ou exporte LLM_API_KEY (FORGE_API_KEY também é aceita)")
	default:
		return fmt.Errorf("modo de autenticação inválido: %q", authMode)
	}
}

// OmniRouteConfig armazena a configuração de fallback via OmniRoute.
type OmniRouteConfig struct {
	Enabled bool
	URL     string
	APIKey  string
	Model   string
}

// loadOmniRouteConfig lê as variáveis de ambiente FORGE_OMNIROUTE_* aplicando defaults.
func loadOmniRouteConfig(getenv func(string) string) OmniRouteConfig {
	cfg := OmniRouteConfig{
		Enabled: false,
		URL:     "http://localhost:20128",
		APIKey:  "omniroute-local",
		Model:   "claude-sonnet-4-6",
	}
	if getenv == nil {
		return cfg
	}
	if strings.EqualFold(strings.TrimSpace(getenv("FORGE_OMNIROUTE_ENABLED")), "true") {
		cfg.Enabled = true
	}
	if v := strings.TrimSpace(getenv("FORGE_OMNIROUTE_URL")); v != "" {
		cfg.URL = v
	}
	if v := strings.TrimSpace(getenv("FORGE_OMNIROUTE_API_KEY")); v != "" {
		cfg.APIKey = v
	}
	if v := strings.TrimSpace(getenv("FORGE_OMNIROUTE_MODEL")); v != "" {
		cfg.Model = v
	}
	return cfg
}

// cleanEnvValue remove espaços e aspas externas (simples ou duplas) de um valor de variável de ambiente.
func cleanEnvValue(v string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(v), `"'`))
}

// buildAgentEnv cria uma nova slice de variáveis de ambiente para a execução do agente.
// Nunca muta base.
func buildAgentEnv(base []string, authMode string, omni OmniRouteConfig, useOmni bool) []string {
	if useOmni {
		var res []string
		for _, kv := range base {
			if strings.HasPrefix(kv, "ANTHROPIC_BASE_URL=") || strings.HasPrefix(kv, "ANTHROPIC_AUTH_TOKEN=") {
				continue
			}
			res = append(res, kv)
		}
		res = append(res, "ANTHROPIC_BASE_URL="+omni.URL)
		res = append(res, "ANTHROPIC_AUTH_TOKEN="+omni.APIKey)
		return res
	}

	if authMode == authModeAnother {
		var res []string
		for _, kv := range base {
			if strings.HasPrefix(kv, "LLM_API_KEY=") {
				continue
			}
			res = append(res, kv)
		}
		return res
	}

	if authMode == authModeAPIKey {
		// Primeiro valor não vazio de cada chave (já sem aspas externas/espaços).
		vals := map[string]string{}
		for _, kv := range base {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				continue
			}
			if k != "ANTHROPIC_API_KEY" && k != "LLM_API_KEY" && k != "FORGE_API_KEY" {
				continue
			}
			if cleaned := cleanEnvValue(v); cleaned != "" && vals[k] == "" {
				vals[k] = cleaned
			}
		}

		// O CLI `claude` lê ANTHROPIC_API_KEY; LLM_API_KEY (models.env) e FORGE_API_KEY são fontes alternativas.
		key := vals["ANTHROPIC_API_KEY"]
		if key == "" {
			key = vals["LLM_API_KEY"]
		}
		if key == "" {
			key = vals["FORGE_API_KEY"]
		}

		res := make([]string, 0, len(base)+2)
		for _, kv := range base {
			if key != "" && strings.HasPrefix(kv, "ANTHROPIC_API_KEY=") {
				continue
			}
			if vals["LLM_API_KEY"] == "" && vals["FORGE_API_KEY"] != "" && strings.HasPrefix(kv, "LLM_API_KEY=") {
				continue
			}
			res = append(res, kv)
		}
		if vals["LLM_API_KEY"] == "" && vals["FORGE_API_KEY"] != "" {
			res = append(res, "LLM_API_KEY="+vals["FORGE_API_KEY"])
		}
		if key != "" {
			res = append(res, "ANTHROPIC_API_KEY="+key)
		}
		return res
	}

	res := make([]string, len(base))
	copy(res, base)
	return res
}
