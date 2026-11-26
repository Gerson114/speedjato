package middleware

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// LimiteCompartilhado usa o relógio do PostgreSQL e um token bucket atômico.
func LimiteCompartilhado(db *sql.DB, namespace string, taxa float64, rajada int) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := permitirCompartilhado(c, db, namespace+":"+ipCliente(c), taxa, rajada)
		if err == sql.ErrNoRows {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Muitas requisições"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Não foi possível verificar o limite de requisições"})
			return
		}
		c.Next()
	}
}

func permitirCompartilhado(c *gin.Context, db *sql.DB, identidade string, taxa float64, rajada int) error {
	chave := fmt.Sprintf("%x", sha256.Sum256([]byte(identidade)))
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	var tokens float64
	return db.QueryRowContext(ctx, `INSERT INTO limite_requisicoes(chave,tokens,atualizado)
 VALUES($1,$2-1,clock_timestamp()) ON CONFLICT(chave) DO UPDATE
 SET tokens=GREATEST(0,LEAST($2,limite_requisicoes.tokens+GREATEST(0,EXTRACT(EPOCH FROM (clock_timestamp()-limite_requisicoes.atualizado)))*$3)-1), atualizado=clock_timestamp()
 WHERE LEAST($2,limite_requisicoes.tokens+GREATEST(0,EXTRACT(EPOCH FROM (clock_timestamp()-limite_requisicoes.atualizado)))*$3)>=1 RETURNING tokens`, chave, float64(rajada), taxa).Scan(&tokens)
}
