package middleware

import (
	"crypto/subtle"
	"net/http"
	"net/netip"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// GatewayToken exige que TODA requisição traga o cabeçalho `X-Gateway-Token`
// igual ao segredo `GATEWAY_TOKEN` — compartilhado só entre o front e o back
// pelo .env. Sem o token correto → 403. O front injeta o header no servidor
// (no proxy do Next), então ele nunca chega ao navegador: na prática, só o seu
// front consegue falar com a API, mesmo que o backend esteja acessível.
//
// Se `GATEWAY_TOKEN` não estiver definido (ex.: desenvolvimento local), a
// checagem fica DESLIGADA — não atrapalha testes com curl.
func GatewayToken() gin.HandlerFunc {
	esperado := strings.TrimSpace(os.Getenv("GATEWAY_TOKEN"))
	return func(c *gin.Context) {
		if esperado == "" {
			c.Next()
			return
		}
		recebido := strings.TrimSpace(c.GetHeader("X-Gateway-Token"))
		// Comparação em tempo constante (evita vazar o token por timing).
		if subtle.ConstantTimeCompare([]byte(recebido), []byte(esperado)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Acesso negado"})
			return
		}
		// Somente o gateway autenticado pode afirmar o IP do cliente. Nunca use
		// X-Forwarded-For recebido diretamente para decidir cotas de acesso.
		ip, err := netip.ParseAddr(c.GetHeader("X-Client-IP"))
		if err == nil && ip.Zone() == "" && !ip.IsUnspecified() && !ip.IsMulticast() {
			c.Set("gateway_client_ip", ip.Unmap().String())
		} else if os.Getenv("GATEWAY_CLIENT_IP_REQUIRED") == "true" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "IP do cliente ausente ou inválido"})
			return
		}
		c.Next()
	}
}

func ipCliente(c *gin.Context) string {
	if ip := c.GetString("gateway_client_ip"); ip != "" {
		return ip
	}
	return c.ClientIP()
}
