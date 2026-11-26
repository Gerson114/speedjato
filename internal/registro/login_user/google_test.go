package loginuser

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type transporteGoogle func(*http.Request) (*http.Response, error)

func (f transporteGoogle) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGoogleRejeitaEmailNaoVerificado(t *testing.T) {
	const origem = "https://cliente.example"
	t.Setenv("GOOGLE_CLIENT_ID", "cliente-teste")
	t.Setenv("GOOGLE_CLIENT_SECRET", "segredo-teste")
	t.Setenv("CLIENTE_ORIGIN", origem)
	anterior := clienteGoogle.Transport
	t.Cleanup(func() { clienteGoogle.Transport = anterior })
	chamadas := 0
	clienteGoogle.Transport = transporteGoogle(func(r *http.Request) (*http.Response, error) {
		chamadas++
		var body string
		switch r.URL.String() {
		case "https://oauth2.googleapis.com/token":
			if r.Method != "POST" || r.FormValue("grant_type") != "authorization_code" || r.FormValue("code") != "codigo-teste" || r.FormValue("redirect_uri") != origem || r.FormValue("client_id") != "cliente-teste" || r.FormValue("client_secret") != "segredo-teste" {
				t.Fatal("Troca de código OAuth inválida")
			}
			body = `{"access_token":"token-teste"}`
		case "https://openidconnect.googleapis.com/v1/userinfo":
			if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer token-teste" {
				t.Fatal("Consulta de perfil sem a credencial esperada")
			}
			body = `{"sub":"123","email":"cliente@example.org","email_verified":false}`
		default:
			t.Fatalf("URL inesperada: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/google", strings.NewReader(`{"code":"codigo-teste"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	Google(c, nil)
	if chamadas != 2 || w.Code != http.StatusUnauthorized || len(w.Result().Cookies()) != 0 {
		t.Fatalf("Esperava rejeição sem cookie após a consulta ao Google, chamadas=%d status=%d", chamadas, w.Code)
	}
}

func TestGoogleEncerraRequisicaoNoRedirecionamento(t *testing.T) {
	anterior := clienteGoogle.Transport
	t.Cleanup(func() { clienteGoogle.Transport = anterior })
	chamadas := 0
	clienteGoogle.Transport = transporteGoogle(func(r *http.Request) (*http.Response, error) {
		chamadas++
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": []string{"https://outro.example/"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader("code=teste"))
	if err != nil {
		t.Fatal(err)
	}
	var resposta map[string]any
	if respostaGoogle(req, &resposta) || chamadas != 1 {
		t.Fatalf("redirecionamento indevido, chamadas=%d", chamadas)
	}
}
