package loginuser

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	database "lava/external/db"
	"lava/internal/registro/auth"
	"lava/internal/registro/entrada"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// hashFicticio é um hash bcrypt válido usado quando o email não existe.
// Comparar contra ele faz o login gastar o mesmo tempo de um email real,
// evitando que um atacante descubra quais emails estão cadastrados medindo
// o tempo de resposta (enumeração de usuários por timing).
var hashFicticio, _ = bcrypt.GenerateFromPassword([]byte("senha-ficticia-para-timing"), bcrypt.DefaultCost)

func Login(c *gin.Context, db *sql.DB) {
	ctx := c.Request.Context()

	var credenciais entrada.LoginRequest

	// 1. Ler email e senha do corpo da requisição
	if err := c.ShouldBindJSON(&credenciais); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	credenciais.Email = strings.TrimSpace(credenciais.Email)
	if len(credenciais.Password) > 72 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou senha incorretos"})
		return
	}

	// 2. Buscar o usuário no banco pelo email
	cred, err := database.BuscarCredenciais(ctx, db, credenciais.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// email não existe: gastamos o mesmo tempo de um bcrypt real
			// e devolvemos a mesma mensagem de uma senha errada.
			_ = bcrypt.CompareHashAndPassword(hashFicticio, []byte(credenciais.Password))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou senha incorretos"})
			return
		}
		slog.Error("falha ao buscar usuário no login", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao realizar login"})
		return
	}

	// 3. Comparar a senha enviada com o hash guardado no banco
	if err := bcrypt.CompareHashAndPassword([]byte(cred.SenhaHash), []byte(credenciais.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou senha incorretos"})
		return
	}

	// 4. Verificar se a conta de cliente está bloqueada.
	if cred.Role != auth.RoleAdmin && !cred.Liberado {
		c.JSON(http.StatusForbidden, gin.H{"error": "Acesso da conta bloqueado"})
		return
	}

	if c.Query("painel") == "admin" && cred.Role != auth.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Esta conta não é de administrador"})
		return
	}

	if c.Query("painel") != "admin" && cred.Role == auth.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Esta conta é de administrador. Use o painel administrativo ou entre com uma conta de cliente."})
		return
	}

	emitirSessao(c, cred, credenciais.Email, credenciais.RememberMe)
}

func emitirSessao(c *gin.Context, cred database.Credenciais, email string, remember bool) {
	// 5. Gerar o token JWT (carrega a role para autorizar as rotas /admin)
	expiraEm := time.Now().UTC().Truncate(time.Second).Add(auth.DuracaoSessao(remember))
	token, err := auth.GerarToken(cred.ID, email, cred.Role, cred.SessaoVersao, expiraEm)
	if err != nil {
		slog.Error("falha ao gerar token no login", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao realizar login"})
		return
	}

	// O JWT é entregue somente no cookie HttpOnly; o corpo contém apenas dados da UI.
	auth.DefinirCookieToken(c, token, expiraEm)
	c.JSON(http.StatusOK, gin.H{
		"message":   "Login realizado com sucesso",
		"userId":    cred.ID,
		"expiresAt": expiraEm,
		"role":      cred.Role,
	})
}
