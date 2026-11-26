package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// LimiteLogin allows a burst of 100 different users on one office IP, while
// limiting password attempts for each account even when source IPs rotate.
func LimiteLogin(ctx context.Context, db *sql.DB) gin.HandlerFunc {
	porIP := NewRateLimiter(ctx, 10, 100)
	porConta := NewRateLimiter(ctx, 0.1, 5)
	return func(c *gin.Context) {
		if !porIP.limiterPara(ipCliente(c)).Allow() {
			negarLogin(c)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10))
		if err != nil {
			c.AbortWithStatusJSON(413, gin.H{"error": "Corpo muito grande ou inválido"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		var req struct {
			Email string `json:"email"`
		}
		if json.Unmarshal(body, &req) == nil && req.Email != "" {
			hash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(req.Email))))
			chave := hex.EncodeToString(hash[:])
			if !porConta.limiterPara(chave).Allow() {
				negarLogin(c)
				return
			}
			err := permitirCompartilhado(c, db, "login-conta:"+chave, 0.1, 5)
			if err == sql.ErrNoRows {
				negarLogin(c)
				return
			}
			if err != nil {
				c.AbortWithStatusJSON(503, gin.H{"error": "Não foi possível verificar o limite de login"})
				return
			}
		}
		c.Next()
	}
}
func negarLogin(c *gin.Context) {
	c.Header("Retry-After", "10")
	c.AbortWithStatusJSON(429, gin.H{"error": "Muitas tentativas de login. Tente novamente em instantes"})
}
