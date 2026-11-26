package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var ErrEmailCadastrado = errors.New("email já cadastrado")

// EmailJaCadastrado verifica se o email já existe no banco.
func EmailJaCadastrado(ctx context.Context, db *sql.DB, email string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM usuarios WHERE lower(email) = lower($1))`

	var existe bool
	if err := db.QueryRowContext(ctx, query, email).Scan(&existe); err != nil {
		return false, fmt.Errorf("verificar email cadastrado: %w", err)
	}

	return existe, nil
}

// InserirUsuario cria uma conta de cliente com acesso imediato ao login.
func InserirUsuario(ctx context.Context, db *sql.DB, username, email, senhaHash string) error {
	const query = `INSERT INTO usuarios (username, email, password, liberado) VALUES ($1, $2, $3, true)`

	if _, err := db.ExecContext(ctx, query, username, email, senhaHash); err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailCadastrado
		}
		return fmt.Errorf("inserir usuário: %w", err)
	}

	return nil
}

// GarantirAdministrador cria a conta administrativa inicial ou atualiza suas
// credenciais quando ADMIN_EMAIL/ADMIN_PASSWORD mudam no ambiente.
func GarantirAdministrador(ctx context.Context, db *sql.DB, email, senha string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serialize startup across replicas, including creation of a new administrator.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(824602)"); err != nil {
		return err
	}
	var hash, role string
	err = tx.QueryRowContext(ctx, "SELECT password,role FROM usuarios WHERE lower(email)=lower($1) FOR UPDATE", email).Scan(&hash, &role)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && role != "admin" {
		return errors.New("o email administrativo pertence a uma conta de cliente")
	}
	if err == nil && bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) == nil {
		_, err = tx.ExecContext(ctx, "UPDATE usuarios SET liberado=true WHERE lower(email)=lower($1)", email)
	} else {
		var novo []byte
		novo, err = bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if hash == "" {
			_, err = tx.ExecContext(ctx, "INSERT INTO usuarios(username,email,password,role,liberado) VALUES('Administrador',$1,$2,'admin',true)", email, string(novo))
		} else {
			_, err = tx.ExecContext(ctx, "UPDATE usuarios SET password=$2,liberado=true,sessao_versao=sessao_versao+1 WHERE lower(email)=lower($1)", email, string(novo))
		}
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Credenciais reúne o que o login precisa para autenticar e autorizar.
type Credenciais struct {
	ID           int
	SenhaHash    string `json:"-"`
	Role         string
	Liberado     bool
	SessaoVersao int64
}

// BuscarCredenciais busca id, hash da senha, role e estado de liberação de um
// usuário pelo email. Usado no login para validar a senha E decidir se a conta
// já tem acesso liberado.
//
// Quando o email não existe, devolve sql.ErrNoRows (preservado via %w) para o
// chamador distinguir "não encontrado" de uma falha real do banco.
func BuscarCredenciais(ctx context.Context, db *sql.DB, email string) (Credenciais, error) {
	const query = `SELECT id, password, role, liberado, sessao_versao FROM usuarios WHERE lower(email) = lower($1)`

	var cred Credenciais
	if err := db.QueryRowContext(ctx, query, email).Scan(
		&cred.ID, &cred.SenhaHash, &cred.Role, &cred.Liberado, &cred.SessaoVersao,
	); err != nil {
		return Credenciais{}, fmt.Errorf("buscar credenciais por email: %w", err)
	}

	return cred, nil
}
