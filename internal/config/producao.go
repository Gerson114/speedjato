package config

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"
)

// ValidarProducao verifica as configurações HTTP e do gateway em produção.
func ValidarProducao(getenv func(string) string) error {
	if getenv("APP_ENV") != "production" {
		return nil
	}
	if getenv("GIN_MODE") != "release" {
		return fmt.Errorf("produção exige GIN_MODE=release")
	}
	if getenv("GATEWAY_CLIENT_IP_REQUIRED") != "true" {
		return fmt.Errorf("produção exige GATEWAY_CLIENT_IP_REQUIRED=true")
	}
	token := getenv("GATEWAY_TOKEN")
	if len(token) < 32 || strings.TrimSpace(token) != token {
		return fmt.Errorf("produção exige GATEWAY_TOKEN com pelo menos 32 bytes, sem espaços nas extremidades")
	}
	origins := strings.Split(getenv("CORS_ORIGINS"), ",")
	for _, origin := range origins {
		u, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.Contains(u.Host, "*") {
			return fmt.Errorf("CORS_ORIGINS exige origens HTTPS exatas sem caminhos ou curingas")
		}
	}
	for _, proxy := range strings.Split(getenv("TRUSTED_PROXIES"), ",") {
		proxy = strings.TrimSpace(proxy)
		if proxy == "" {
			continue
		}
		if _, err := netip.ParseAddr(proxy); err == nil {
			continue
		}
		bloco, err := netip.ParsePrefix(proxy)
		if err != nil {
			return fmt.Errorf("TRUSTED_PROXIES contém um IP ou CIDR inválido")
		}
		if bloco.Bits() == 0 {
			return fmt.Errorf("TRUSTED_PROXIES não pode confiar em toda a internet")
		}
	}
	return nil
}

// ValidarSessao evita iniciar o login com uma chave de assinatura inválida.
func ValidarSessao(getenv func(string) string) error {
	secret := getenv("JWT_SECRET")
	if len(secret) < 32 || strings.TrimSpace(secret) != secret {
		return fmt.Errorf("JWT_SECRET exige pelo menos 32 bytes, sem espaços nas extremidades")
	}
	if secret == getenv("GATEWAY_TOKEN") {
		return fmt.Errorf("JWT_SECRET e GATEWAY_TOKEN devem ser diferentes")
	}
	if getenv("APP_ENV") == "production" && getenv("COOKIE_SECURE") != "true" {
		return fmt.Errorf("produção exige COOKIE_SECURE=true")
	}
	return nil
}
