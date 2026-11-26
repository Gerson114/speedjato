package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// PrazoContexto também cancela consultas SQL e HTTP que usam o contexto.
// WriteTimeout limita a escrita da resposta, mas não cancela essas operações.
func PrazoContexto(prazo time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), prazo)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
