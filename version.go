package main

// Version define a versão atual do forgeloop seguindo SemVer.
const Version = "0.1.0"

// GetVersion retorna o nome da aplicação concatenado com o prefixo 'v' e a versão.
func GetVersion() string {
	return "forgeloop v" + Version
}
