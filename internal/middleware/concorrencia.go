package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// LimiteWebSocket impede que conexões persistentes ocupem toda a capacidade HTTP.
func LimiteWebSocket(max int) gin.HandlerFunc {
	limitar := LimiteConcorrencia(max)
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet && strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
			limitar(c)
			return
		}
		c.Next()
	}
}

// LimiteConcorrencia refuses excess work rather than queueing unbounded goroutines.
func LimiteConcorrencia(max int) gin.HandlerFunc {
	sem := make(chan struct{}, max)
	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		default:
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(503, gin.H{"error": "Servidor ocupado, tente novamente"})
		}
	}
}
