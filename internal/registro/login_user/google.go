package loginuser

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	database "lava/external/db"
)

type perfilGoogle struct {
	Subject  string `json:"sub"`
	Email    string `json:"email"`
	Verified bool   `json:"email_verified"`
	Name     string `json:"name"`
}

// O código de uso único vem do seletor de contas em modo popup.
func Google(c *gin.Context, db *sql.DB) {
	id, secret := os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET")
	origin := os.Getenv("CLIENTE_ORIGIN")
	if id == "" || secret == "" || origin == "" {
		c.JSON(503, gin.H{"error": "Login com Google indisponível"})
		return
	}
	var entrada struct {
		Code string `json:"code" binding:"required"`
	}
	if c.ShouldBindJSON(&entrada) != nil {
		c.JSON(400, gin.H{"error": "Código do Google inválido"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
	defer cancel()
	form := url.Values{"client_id": {id}, "client_secret": {secret}, "code": {entrada.Code}, "grant_type": {"authorization_code"}, "redirect_uri": {origin}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var token struct {
		Access string `json:"access_token"`
	}
	if !respostaGoogle(req, &token) || token.Access == "" {
		c.JSON(401, gin.H{"error": "Não foi possível confirmar o login com Google. Tente novamente."})
		return
	}
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+token.Access)
	var p perfilGoogle
	if !respostaGoogle(req, &p) || p.Subject == "" || p.Email == "" || !p.Verified {
		c.JSON(401, gin.H{"error": "O Google precisa confirmar o e-mail da conta."})
		return
	}
	var cred database.Credenciais
	var email string
	err := db.QueryRowContext(ctx, `INSERT INTO usuarios (google_sub,username,email,password,liberado)
 VALUES ($1,$2,$3,'!',true) ON CONFLICT (google_sub) DO UPDATE SET google_sub=EXCLUDED.google_sub
 RETURNING id,email,role,liberado,sessao_versao`, p.Subject, p.Name, strings.ToLower(p.Email)).Scan(&cred.ID, &email, &cred.Role, &cred.Liberado, &cred.SessaoVersao)
	var conflito *pq.Error
	if errors.As(err, &conflito) && conflito.Code == "23505" {
		c.JSON(409, gin.H{"error": "Este e-mail já tem cadastro. Entre com sua senha."})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "Erro ao acessar a conta"})
		return
	}
	if cred.Role != "user" || !cred.Liberado {
		c.JSON(403, gin.H{"error": "Acesso à área do cliente bloqueado"})
		return
	}
	emitirSessao(c, cred, email, false)
}

var clienteGoogle = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

func respostaGoogle(req *http.Request, destino any) bool {
	resp, err := clienteGoogle.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200 && json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(destino) == nil
}
