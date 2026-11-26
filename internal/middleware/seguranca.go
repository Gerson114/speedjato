package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SegurancaHeaders adiciona cabeçalhos de segurança em TODAS as respostas. Como a
// API só devolve JSON (não serve HTML), a CSP é a mais restritiva possível e o
// embed em iframe é proibido (anti-clickjacking). O nosniff impede o navegador de
// "adivinhar" o content-type. HSTS só é honrado sob HTTPS (atrás do Traefik); em
// HTTP o navegador o ignora, então é seguro deixar sempre.
func SegurancaHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}

// LimiteCorpo limita o tamanho do corpo da requisição (proteção contra payloads
// gigantes que estouram a memória do servidor). Acima de `maxBytes` a leitura do
// corpo falha e o handler responde 413.
func LimiteCorpo(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Corpo muito grande"})
			return
		}
		if c.Request.Body != nil && c.Request.Body != http.NoBody {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
