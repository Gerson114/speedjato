package cadastrouser

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	database "lava/external/db"
	"lava/internal/registro/entrada"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func CadastrarUsuario(c *gin.Context, db *sql.DB) {
	ctx := c.Request.Context()

	var req entrada.CadastroRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if req.Username == "" || len(req.Username) > 200 || len(req.Password) > 72 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nome inválido ou senha maior que 72 bytes"})
		return
	}

	emailExiste, err := database.EmailJaCadastrado(ctx, db, req.Email)
	if err != nil {
		slog.Error("falha ao verificar email no cadastro", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao cadastrar usuário"})
		return
	}
	if emailExiste {
		c.JSON(http.StatusConflict, gin.H{"error": "Email já cadastrado"})
		return
	}

	senhaHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("falha ao gerar hash da senha", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao cadastrar usuário"})
		return
	}

	if err := database.InserirUsuario(ctx, db, req.Username, req.Email, string(senhaHash)); err != nil {
		if errors.Is(err, database.ErrEmailCadastrado) {
			c.JSON(http.StatusConflict, gin.H{"error": "Email já cadastrado"})
			return
		}
		slog.Error("falha ao inserir usuário", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao cadastrar usuário"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Usuário cadastrado com sucesso"})
}
