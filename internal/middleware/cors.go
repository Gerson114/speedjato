package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// origensPermitidas lê CORS_ORIGINS do ambiente (lista separada por vírgula).
// Em desenvolvimento, se a variável não existir, libera o dev server do Next.
func origensPermitidas() map[string]struct{} {
	bruto := os.Getenv("CORS_ORIGINS")
	if bruto == "" {
		bruto = "http://localhost:3000,http://localhost:3100"
	}

	permitidas := make(map[string]struct{})
	for o := range strings.SplitSeq(bruto, ",") {
		if o = strings.TrimSpace(o); o != "" {
			permitidas[o] = struct{}{}
		}
	}
	return permitidas
}

// CORS libera o frontend (origens configuradas) a chamar a API a partir do
// navegador, respondendo aos preflights (OPTIONS) e ecoando apenas origens
// conhecidas no Access-Control-Allow-Origin.
func CORS() gin.HandlerFunc {
	permitidas := origensPermitidas()

	return func(c *gin.Context) {
		origem := c.GetHeader("Origin")
		// CORS sozinho só bloqueia a leitura da resposta. Rejeita a execução
		// de mutações vindas de origens externas, inclusive subdomínios same-site.
		if origem != "" {
			if _, ok := permitidas[origem]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Origem não permitida"})
				return
			}
		} else if c.GetHeader("Sec-Fetch-Site") == "cross-site" &&
			c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Origem não permitida"})
			return
		}
		if _, ok := permitidas[origem]; ok {
			c.Header("Access-Control-Allow-Origin", origem)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "300")
		}

		// Encerra o preflight antes de chegar nas rotas.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
