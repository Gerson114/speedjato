package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"lava/internal/config"

	_ "github.com/lib/pq" // driver "postgres" para database/sql
)

// ConnectDB abre uma conexão *sql.DB com o PostgreSQL.
// É essa conexão que os handlers e services usam (todos esperam *sql.DB).
func ConnectDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão: %w", err)
	}

	limite := config.LimiteConexoesBanco()
	db.SetMaxOpenConns(limite)
	db.SetMaxIdleConns(min(config.ConexoesBancoOciosas, limite))
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("erro ao conectar: %w", err)
	}

	return db, nil
}
