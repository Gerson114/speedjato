package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGatewayNaoAceitaIPForjadoSemSegredo(t *testing.T) {
	key := strings.Repeat("g", 32)
	t.Setenv("GATEWAY_TOKEN", key)
	t.Setenv("GATEWAY_CLIENT_IP_REQUIRED", "true")
	r := gin.New()
	r.Use(GatewayToken())
	r.GET("/privado", func(c *gin.Context) { c.String(200, ipCliente(c)) })
	for _, tc := range []struct {
		token, ip string
		status    int
	}{
		{"", "198.51.100.1", 403}, {"errado", "198.51.100.1", 403},
		{key, "", 400}, {key, "0.0.0.0", 400}, {key, "198.51.100.1,198.51.100.2", 400}, {key, "198.51.100.1", 200},
	} {
		req := httptest.NewRequest("GET", "/privado", nil)
		req.Header.Set("X-Gateway-Token", tc.token)
		req.Header.Set("X-Client-IP", tc.ip)
		req.Header.Set("X-Forwarded-For", "203.0.113.99")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("IP %q: esperava %d, recebeu %d", tc.ip, tc.status, w.Code)
		}
		if tc.status == 200 && w.Body.String() != tc.ip {
			t.Fatal("IP não autenticado foi utilizado")
		}
	}
}
