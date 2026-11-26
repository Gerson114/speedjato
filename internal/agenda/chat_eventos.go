package agenda

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lib/pq"
	"lava/internal/registro/auth"
)

var chatOuvintes = struct {
	sync.Mutex
	canais map[chan int]struct{}
}{canais: make(map[chan int]struct{})}

// IniciarChatEventos recebe notificações do PostgreSQL em cada instância.
func IniciarChatEventos(ctx context.Context) {
	listener := pq.NewListener(os.Getenv("DATABASE_URL"), time.Second, 30*time.Second, func(_ pq.ListenerEventType, err error) {
		if err != nil {
			slog.Error("conexão de eventos do chat", "err", err)
		}
	})
	if err := listener.Listen("lava_chat"); err != nil {
		slog.Error("não foi possível escutar o chat", "err", err)
	}
	go func() {
		defer listener.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case evento, ok := <-listener.Notify:
				if !ok {
					return
				}
				// Uma reconexão pode ter perdido eventos. Recarregue todas as conversas.
				cliente := 0
				if evento != nil {
					cliente, _ = strconv.Atoi(evento.Extra)
				}
				chatOuvintes.Lock()
				for canal := range chatOuvintes.canais {
					select {
					case canal <- cliente:
					default:
					}
				}
				chatOuvintes.Unlock()
			}
		}
	}()
}

func (a api) eventosChat(c *gin.Context) {
	cliente := 0
	if c.GetHeader("Origin") == "" {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if c.GetString(auth.ContextRole) == "user" {
		cliente = c.GetInt(auth.ContextUserID)
	} else if c.Param("cliente") != "" {
		var ok bool
		cliente, ok = a.chatCliente(c)
		if !ok {
			return
		}
	}
	// A origem já foi validada pelo middleware CORS contra os domínios configurados.
	conexao, err := (&websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conexao.Close()
	canal := make(chan int, 1)
	chatOuvintes.Lock()
	chatOuvintes.canais[canal] = struct{}{}
	chatOuvintes.Unlock()
	defer func() { chatOuvintes.Lock(); delete(chatOuvintes.canais, canal); chatOuvintes.Unlock() }()
	conexao.SetReadLimit(128)
	fechado := make(chan struct{})
	go func() {
		defer close(fechado)
		for {
			if _, _, err := conexao.ReadMessage(); err != nil {
				return
			}
		}
	}()
	enviar := func() error {
		_ = conexao.SetWriteDeadline(time.Now().Add(5 * time.Second))
		return conexao.WriteJSON(gin.H{"evento": "atualizar"})
	}
	if enviar() != nil {
		return
	}
	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-fechado:
			return
		case <-timer.C:
			return
		case id := <-canal:
			if cliente == 0 || id == 0 || id == cliente {
				if enviar() != nil {
					return
				}
			}
		}
	}
}
