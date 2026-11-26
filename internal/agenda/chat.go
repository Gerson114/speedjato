package agenda

import (
	"github.com/gin-gonic/gin"
	"lava/internal/registro/auth"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (a api) chatCliente(c *gin.Context) (int, bool) {
	if c.GetString(auth.ContextRole) == "user" {
		return c.GetInt(auth.ContextUserID), true
	}
	id, ok := idPositivo(c.Param("cliente"))
	if !ok {
		c.JSON(400, gin.H{"error": "Cliente inválido"})
		return 0, false
	}
	var existe bool
	if err := a.db.QueryRowContext(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM usuarios WHERE id=$1 AND role='user')", id).Scan(&existe); err != nil {
		falha(c, err)
		return 0, false
	}
	if !existe {
		c.JSON(404, gin.H{"error": "Cliente não encontrado"})
		return 0, false
	}
	return id, true
}
func (a api) conversas(c *gin.Context) {
	rows, err := a.db.QueryContext(c.Request.Context(), `SELECT u.id,u.username,u.email,m.texto,m.criado_em
 FROM usuarios u JOIN LATERAL (SELECT texto,criado_em FROM chat_mensagens WHERE cliente_id=u.id ORDER BY id DESC LIMIT 1) m ON true
 ORDER BY m.criado_em DESC,u.id DESC LIMIT 200`)
	if err != nil {
		falha(c, err)
		return
	}
	defer rows.Close()
	lista := []gin.H{}
	for rows.Next() {
		var id int
		var nome, email, texto string
		var data interface{}
		if err = rows.Scan(&id, &nome, &email, &texto, &data); err != nil {
			falha(c, err)
			return
		}
		lista = append(lista, gin.H{"id": id, "nome": nome, "email": email, "ultimaMensagem": texto, "criadoEm": data})
	}
	if err = rows.Err(); err != nil {
		falha(c, err)
		return
	}
	c.JSON(200, lista)
}
func (a api) mensagens(c *gin.Context) {
	cliente, ok := a.chatCliente(c)
	if !ok {
		return
	}
	antes := int64(0)
	if valor := c.Query("antes"); valor != "" {
		var err error
		antes, err = strconv.ParseInt(valor, 10, 64)
		if err != nil || antes < 1 {
			c.JSON(400, gin.H{"error": "Página inválida"})
			return
		}
	}
	rows, err := a.db.QueryContext(c.Request.Context(), `SELECT id,autor_papel,texto,criado_em FROM chat_mensagens WHERE cliente_id=$1 AND ($2=0 OR id<$2) ORDER BY id DESC LIMIT 50`, cliente, antes)
	if err != nil {
		falha(c, err)
		return
	}
	defer rows.Close()
	lista := []gin.H{}
	for rows.Next() {
		var id int64
		var autor, texto string
		var data interface{}
		if err = rows.Scan(&id, &autor, &texto, &data); err != nil {
			falha(c, err)
			return
		}
		lista = append(lista, gin.H{"id": id, "autor": autor, "texto": texto, "criadoEm": data})
	}
	if err = rows.Err(); err != nil {
		falha(c, err)
		return
	}
	for i, j := 0, len(lista)-1; i < j; i, j = i+1, j-1 {
		lista[i], lista[j] = lista[j], lista[i]
	}
	c.JSON(200, lista)
}
func (a api) enviarMensagem(c *gin.Context) {
	cliente, ok := a.chatCliente(c)
	if !ok {
		return
	}
	var entrada struct {
		Texto string `json:"texto"`
	}
	if c.ShouldBindJSON(&entrada) != nil {
		c.JSON(400, gin.H{"error": "Mensagem inválida"})
		return
	}
	entrada.Texto = strings.TrimSpace(entrada.Texto)
	if !utf8.ValidString(entrada.Texto) || utf8.RuneCountInString(entrada.Texto) < 1 || utf8.RuneCountInString(entrada.Texto) > 2000 || strings.ContainsRune(entrada.Texto, 0) {
		c.JSON(400, gin.H{"error": "Escreva uma mensagem de até 2000 caracteres"})
		return
	}
	var id int64
	var data, notificacao interface{}
	err := a.db.QueryRowContext(c.Request.Context(), `WITH inserida AS (INSERT INTO chat_mensagens(cliente_id,autor_id,autor_papel,texto) VALUES($1,$2,$3,$4) RETURNING id,criado_em,cliente_id) SELECT id,criado_em,pg_notify('lava_chat',cliente_id::text) FROM inserida`, cliente, c.GetInt(auth.ContextUserID), c.GetString(auth.ContextRole), entrada.Texto).Scan(&id, &data, &notificacao)
	if err != nil {
		falha(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "autor": c.GetString(auth.ContextRole), "texto": entrada.Texto, "criadoEm": data})
}
