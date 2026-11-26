package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestRejeitaTokensInvalidos(t *testing.T) {
	key := strings.Repeat("a", 32)
	t.Setenv("JWT_SECRET", key)
	agora := time.Now()
	for _, caso := range []string{"valido", "expirado", "sem_expiracao", "sem_emissao", "emissao_futura", "papel_invalido", "sem_versao", "outra_assinatura"} {
		t.Run(caso, func(t *testing.T) {
			claims := Claims{UserID: 1, Email: "conta@example.test", Role: "user", SessaoVersao: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(agora.Add(time.Hour)), IssuedAt: jwt.NewNumericDate(agora.Add(-time.Minute))}}
			method := jwt.SigningMethodHS256
			switch caso {
			case "expirado":
				claims.ExpiresAt = jwt.NewNumericDate(agora.Add(-time.Minute))
			case "sem_expiracao":
				claims.ExpiresAt = nil
			case "sem_emissao":
				claims.IssuedAt = nil
			case "emissao_futura":
				claims.IssuedAt = jwt.NewNumericDate(agora.Add(time.Hour))
			case "papel_invalido":
				claims.Role = "superadmin"
			case "sem_versao":
				claims.SessaoVersao = 0
			case "outra_assinatura":
				method = jwt.SigningMethodHS512
			}
			raw, err := jwt.NewWithClaims(method, claims).SignedString([]byte(key))
			if err != nil {
				t.Fatal(err)
			}
			_, err = ValidarToken(raw)
			if (err == nil) != (caso == "valido") {
				t.Fatalf("validação incorreta: %v", err)
			}
		})
	}
}
