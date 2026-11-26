package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	database "lava/external/db"
	"lava/internal/config"
	"lava/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func prepararAdministrador(ctx context.Context, db *sql.DB) error {
	email := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	senha := os.Getenv("ADMIN_PASSWORD")
	if email == "" || senha == "" {
		return fmt.Errorf("configure ADMIN_EMAIL e ADMIN_PASSWORD no .env")
	}
	return database.GarantirAdministrador(ctx, db, email, senha)
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := executar(); err != nil {
		slog.Error("falha no servidor", "err", err)
		os.Exit(1)
	}
}

func executar() error {
	if os.Getenv("APP_ENV") != "production" {
		if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("não foi possível carregar o arquivo .env")
		}
	}
	if err := config.ValidarProducao(os.Getenv); err != nil {
		return err
	}
	if err := config.ValidarSessao(os.Getenv); err != nil {
		return err
	}
	gin.SetMode(os.Getenv("GIN_MODE"))
	db, err := database.ConnectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()
	schemaCtx, cancelar := context.WithTimeout(ctx, 30*time.Second)
	defer cancelar()
	if os.Getenv("APP_ENV") == "production" {
		err = database.VerificarMigracoes(schemaCtx, db)
	} else {
		err = database.Migrar(schemaCtx, db)
	}
	if err != nil {
		return fmt.Errorf("migrações do banco: %w", err)
	}
	if err := prepararAdministrador(schemaCtx, db); err != nil {
		return fmt.Errorf("preparar administrador: %w", err)
	}
	cancelar()
	r, err := router.Novo(ctx, db)
	if err != nil {
		return fmt.Errorf("configurar roteador: %w", err)
	}
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      config.PrazoRespostaHTTP,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	falha := make(chan error, 1)
	go func() { falha <- srv.Serve(listener) }()
	slog.Info("servidor iniciado", "endereco", srv.Addr)
	select {
	case err := <-falha:
		return err
	case <-ctx.Done():
		prazo, cancelar := context.WithTimeout(context.Background(), config.PrazoRespostaHTTP+5*time.Second)
		defer cancelar()
		return srv.Shutdown(prazo)
	}
}
