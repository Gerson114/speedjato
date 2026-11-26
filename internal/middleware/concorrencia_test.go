package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLimiteWebSocketPreservaCapacidadeHTTP(t *testing.T) {
	r := gin.New()
	r.Use(LimiteWebSocket(1), LimiteConcorrencia(2))
	entrou := make(chan struct{})
	liberar := make(chan struct{})
	r.GET("/", func(c *gin.Context) {
		if c.GetHeader("Upgrade") != "" {
			close(entrou)
			<-liberar
		}
		c.Status(200)
	})
	terminou := make(chan struct{})
	go func() {
		defer close(terminou)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Upgrade", "websocket")
		r.ServeHTTP(httptest.NewRecorder(), req)
	}()
	<-entrou
	ws := httptest.NewRequest("GET", "/", nil)
	ws.Header.Set("Upgrade", "websocket")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, ws)
	if w.Code != 503 {
		t.Fatalf("WebSocket excedente: HTTP %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Fatalf("requisição normal: HTTP %d", w.Code)
	}
	close(liberar)
	<-terminou
}
