package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Durações escolhidas pelo servidor, nunca recebidas diretamente do cliente.
const (
	tokenDuration           = 24 * time.Hour
	rememberedTokenDuration = 30 * 24 * time.Hour
)

func DuracaoSessao(manterConectado bool) time.Duration {
	if manterConectado {
		return rememberedTokenDuration
	}
	return tokenDuration
}

// Claims são os dados que viajam dentro do token JWT.
type Claims struct {
	UserID       int    `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"` // "user" ou "admin" — autoriza as rotas /admin
	SessaoVersao int64  `json:"sessao_versao"`
	jwt.RegisteredClaims
}

// secret lê a chave de assinatura da variável de ambiente JWT_SECRET.
func secret() ([]byte, error) {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		return nil, errors.New("JWT_SECRET não configurado")
	}
	return []byte(s), nil
}

// GerarToken cria um token JWT assinado para o usuário informado.
func GerarToken(userID int, email, role string, sessaoVersao int64, expiraEm time.Time) (string, error) {
	key, err := secret()
	if err != nil {
		return "", err
	}

	claims := Claims{
		UserID:       userID,
		Email:        email,
		Role:         role,
		SessaoVersao: sessaoVersao,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiraEm),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

// ValidarToken verifica a assinatura e a validade do token, retornando as claims.
func ValidarToken(tokenString string) (*Claims, error) {
	key, err := secret()
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("método de assinatura inesperado")
		}
		return key, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.UserID <= 0 || claims.SessaoVersao <= 0 || claims.IssuedAt == nil ||
		(claims.Role != "user" && claims.Role != RoleAdmin) {
		return nil, errors.New("token inválido")
	}

	return claims, nil
}
