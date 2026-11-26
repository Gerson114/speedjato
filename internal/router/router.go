package router

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"lava/internal/agenda"
	"lava/internal/config"
	"lava/internal/middleware"
	cadastrouser "lava/internal/registro/cadastro_user"
	loginuser "lava/internal/registro/login_user"

	"github.com/gin-gonic/gin"
)

// Novo prepara o servidor HTTP e registra as rotas da aplicação.
func Novo(ctx context.Context, db *sql.DB) (*gin.Engine, error) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				limpeza, cancelar := context.WithTimeout(ctx, 5*time.Second)
				_, err := db.ExecContext(limpeza, "DELETE FROM limite_requisicoes WHERE atualizado < now() - interval '1 hour'")
				cancelar()
				if err != nil {
					slog.Error("falha ao limpar limites", "err", err)
				}
			}
		}
	}()
	go acompanharFila(ctx, db)
	r := gin.New()
	var proxies []string
	for value := range strings.SplitSeq(os.Getenv("TRUSTED_PROXIES"), ",") {
		if proxy := strings.TrimSpace(value); proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	if err := r.SetTrustedProxies(proxies); err != nil {
		return nil, err
	}
	r.Use(func(c *gin.Context) {
		inicio := time.Now()
		c.Next()
		slog.Info("requisição HTTP", "rota", c.FullPath(), "status", c.Writer.Status(), "duracao_ms", time.Since(inicio).Milliseconds())
	})
	r.Use(gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		slog.Error("panic recuperado na requisição", "rota", c.FullPath())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro interno do servidor"})
	}))
	r.Use(middleware.SegurancaHeaders(), middleware.LimiteWebSocket(16), middleware.LimiteConcorrencia(config.RequisicoesSimultaneas))
	r.GET("/readyz", func(c *gin.Context) {
		prazo, cancelar := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancelar()
		if err := db.PingContext(prazo); err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Status(http.StatusOK)
	})
	r.Use(
		middleware.PrazoContexto(config.PrazoRequisicaoHTTP), middleware.CORS(), middleware.GatewayToken(),
		middleware.NewRateLimiter(ctx, config.RequisicoesPorIP, config.RajadaPorIP).Handler(),
		middleware.LimiteCompartilhado(db, "geral", config.RequisicoesPorIP, config.RajadaPorIP),
		middleware.LimiteCorpo(1<<20))
	if err := SetupRoutes(ctx, r, db); err != nil {
		return nil, err
	}
	return r, nil
}

// SetupRoutes é o ponto de registro das novas rotas do Lava.
func SetupRoutes(ctx context.Context, r *gin.Engine, db *sql.DB) error {
	local, err := time.LoadLocation("America/Fortaleza")
	if err != nil {
		return err
	}
	grupo := r.Group("/auth", middleware.ValidarEntradaAuth())
	limitarSenhas := middleware.LimiteConcorrencia(4)
	grupo.POST("/cadastrar", middleware.NewRateLimiter(ctx, 1, 5).Handler(), middleware.LimiteCompartilhado(db, "cadastro", 1, 5), limitarSenhas, func(c *gin.Context) {
		cadastrouser.CadastrarUsuario(c, db)
	})
	grupo.POST("/login", middleware.LimiteLogin(ctx, db), limitarSenhas, func(c *gin.Context) {
		loginuser.Login(c, db)
	})
	grupo.POST("/google", middleware.LimiteCompartilhado(db, "google", 1, 5), func(c *gin.Context) { loginuser.Google(c, db) })
	agenda.Registrar(r, db, local)
	return nil
}

// acompanharFila processa ausências de forma serializada entre instâncias.
func acompanharFila(ctx context.Context, db *sql.DB) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			prazo, cancelar := context.WithTimeout(ctx, 10*time.Second)
			err := agenda.ProcessarAusencias(prazo, db)
			cancelar()
			if err != nil && ctx.Err() == nil {
				slog.Error("falha ao atualizar fila", "err", err)
			}
		}
	}
}
