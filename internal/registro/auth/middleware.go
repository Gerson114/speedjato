package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// chaves usadas para guardar os dados do usuário no contexto da requisição.
const (
	ContextUserID = "user_id"
	ContextEmail  = "email"
	ContextRole   = "role"
)

// RoleAdmin é a role que dá acesso às rotas administrativas.
const RoleAdmin = "admin"

// Nomes dos cookies httpOnly que guardam o JWT no navegador. Há um por painel
// porque cookies são escopados por HOST (ignoram a porta): em localhost, o painel
// do usuário (:3000) e o admin (:3100) compartilham o mesmo host e, com um único
// nome, um login sobrescreveria a sessão do outro. Nomes distintos deixam as duas
// sessões coexistirem. Clientes via header Authorization também continuam aceitos.
const (
	CookieToken      = "token"       // painel do usuário
	CookieTokenAdmin = "admin_token" // painel administrativo
)

// nomeCookiePainel escolhe o cookie pelo parâmetro ?painel=admin (usado no
// login para gravar o cookie do painel correto).
func nomeCookiePainel(c *gin.Context) string {
	if c.Query("painel") == "admin" {
		return CookieTokenAdmin
	}
	return CookieToken
}

// tokenDaRequisicao extrai o JWT: primeiro do header Authorization (Bearer) e, se
// ausente, do cookie httpOnly do painel correspondente. Nunca usa a sessão
// de outro painel como fallback.
func tokenDaRequisicao(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header != "" {
		partes := strings.SplitN(header, " ", 2)
		if len(partes) == 2 && strings.EqualFold(partes[0], "Bearer") {
			return strings.TrimSpace(partes[1])
		}
		return "" // header presente mas malformado
	}

	nome := CookieToken
	if strings.HasPrefix(c.Request.URL.Path, "/admin") {
		nome = CookieTokenAdmin
	}
	if v, err := c.Cookie(nome); err == nil {
		return strings.TrimSpace(v)
	}
	return ""
}

// cookieSecure liga o atributo Secure do cookie (só trafega por HTTPS) quando
// COOKIE_SECURE=true. Em desenvolvimento (http://localhost) fica desligado, senão
// o navegador não envia o cookie.
func cookieSecure() bool {
	return strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true")
}

// DefinirCookieToken grava o JWT num cookie httpOnly + SameSite=Strict — o JS do
// navegador não consegue lê-lo (defesa contra roubo por XSS) e ele não viaja em
// requisições de outros sites (defesa contra CSRF). Validade igual à do token.
func DefinirCookieToken(c *gin.Context, token string, expiraEm time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: nomeCookiePainel(c), Value: token, Path: "/",
		MaxAge: max(1, int(time.Until(expiraEm).Seconds())), Expires: expiraEm,
		HttpOnly: true, Secure: cookieSecure(), SameSite: http.SameSiteStrictMode,
	})
}

// Middleware protege rotas exigindo um token JWT válido no header Authorization.
//
//	Authorization: Bearer <token>
func Middleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := tokenDaRequisicao(c)
		if tokenStr == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Token não fornecido"})
			return
		}

		claims, err := ValidarToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Token inválido ou expirado"})
			return
		}

		var role, email string
		var liberado bool
		var versao int64
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		err = db.QueryRowContext(ctx,
			`SELECT role, email, liberado, sessao_versao FROM usuarios WHERE id = $1`, claims.UserID,
		).Scan(&role, &email, &liberado, &versao)
		cancel()
		if errors.Is(err, sql.ErrNoRows) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Sessão encerrada. Entre novamente."})
			return
		}
		if err != nil {
			slog.Error("falha ao verificar sessão", "err", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Não foi possível verificar a sessão"})
			return
		}
		if versao != claims.SessaoVersao || role != claims.Role || (role != RoleAdmin && !liberado) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Sessão encerrada ou acesso bloqueado. Entre novamente."})
			return
		}

		// disponibiliza os dados do usuário para os próximos handlers
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, email)
		c.Set(ContextRole, role)

		c.Next()
	}
}
