package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
)

//go:embed migrations/*.sql
var migracoes embed.FS

func Migrar(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serializa executores, inclusive na primeira instalação.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(719284103)"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
 nome TEXT PRIMARY KEY, checksum TEXT NOT NULL, aplicada_em TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	nomes, err := fs.Glob(migracoes, "migrations/*.sql")
	if err != nil {
		return err
	}
	for _, nome := range nomes {
		conteudo, err := migracoes.ReadFile(nome)
		if err != nil {
			return err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(conteudo))
		var existente string
		err = tx.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE nome=$1", nome).Scan(&existente)
		if err == nil {
			if existente != checksum {
				return fmt.Errorf("migração já aplicada foi modificada: %s", nome)
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(conteudo)); err != nil {
			return fmt.Errorf("aplicar %s: %w", nome, err)
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(nome,checksum) VALUES($1,$2)", nome, checksum); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// VerificarMigracoes não modifica o banco durante a inicialização em produção.
func VerificarMigracoes(ctx context.Context, db *sql.DB) error {
	nomes, err := fs.Glob(migracoes, "migrations/*.sql")
	if err != nil {
		return err
	}
	for _, nome := range nomes {
		conteudo, err := migracoes.ReadFile(nome)
		if err != nil {
			return err
		}
		var checksum string
		if err = db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE nome=$1", nome).Scan(&checksum); err != nil {
			return fmt.Errorf("migração pendente %s: execute o comando migrate: %w", nome, err)
		}
		if checksum != fmt.Sprintf("%x", sha256.Sum256(conteudo)) {
			return fmt.Errorf("checksum inválido: %s", nome)
		}
	}
	return nil
}
