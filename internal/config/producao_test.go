package config

import (
	"strings"
	"testing"
)

func TestValidarProxies(t *testing.T) {
	ambiente := map[string]string{"APP_ENV": "production", "GIN_MODE": "release", "GATEWAY_CLIENT_IP_REQUIRED": "true", "GATEWAY_TOKEN": strings.Repeat("a", 32), "CORS_ORIGINS": "https://cliente.example"}
	for _, teste := range []struct {
		valor  string
		valido bool
	}{
		{"", true}, {"10.1.0.2, 192.0.2.0/24", true}, {"2001:db8::1/64", true}, {"192.0.2.1/0", false}, {"2001:db8::1/0", false}, {"invalido", false},
	} {
		ambiente["TRUSTED_PROXIES"] = teste.valor
		err := ValidarProducao(func(chave string) string { return ambiente[chave] })
		if (err == nil) != teste.valido {
			t.Errorf("TRUSTED_PROXIES=%q: %v", teste.valor, err)
		}
	}
}
