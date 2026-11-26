// Package middleware reúne middlewares HTTP reutilizáveis do Gin.
package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// visitante guarda o limiter de um IP e quando ele foi usado pela última vez
// (para limpar IPs inativos e não vazar memória).
type visitante struct {
	limiter   *rate.Limiter
	ultimoUso time.Time
}

// RateLimiter limita requisições por IP. Útil em rotas sensíveis como o login,
// para conter ataques de força bruta.
type RateLimiter struct {
	mu            sync.Mutex
	visitantes    map[string]*visitante
	taxa          rate.Limit
	burst         int
	maxVisitantes int
	excesso       *rate.Limiter
}

// NewRateLimiter cria um limiter que permite `porSegundo` requisições por
// segundo por IP, com uma rajada (burst) de até `burst` requisições.
func NewRateLimiter(ctx context.Context, porSegundo float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitantes:    make(map[string]*visitante),
		taxa:          rate.Limit(porSegundo),
		burst:         burst,
		maxVisitantes: 10000,
		excesso:       rate.NewLimiter(rate.Limit(porSegundo), burst),
	}
	go rl.limparInativos(ctx)
	return rl
}

func (rl *RateLimiter) limiterPara(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitantes[ip]
	if !ok {
		if len(rl.visitantes) >= rl.maxVisitantes {
			return rl.excesso
		}
		l := rate.NewLimiter(rl.taxa, rl.burst)
		rl.visitantes[ip] = &visitante{limiter: l, ultimoUso: time.Now()}
		return l
	}
	v.ultimoUso = time.Now()
	return v.limiter
}

// limparInativos remove periodicamente IPs sem atividade recente.
func (rl *RateLimiter) limparInativos(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		rl.mu.Lock()
		for ip, v := range rl.visitantes {
			if time.Since(v.ultimoUso) > 3*time.Minute {
				delete(rl.visitantes, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Handler é o middleware do Gin que aplica o limite por IP.
func (rl *RateLimiter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.limiterPara(ipCliente(c)).Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Muitas requisições, tente novamente em instantes",
			})
			return
		}
		c.Next()
	}
}
