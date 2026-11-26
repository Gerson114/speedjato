package agenda

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"lava/internal/registro/auth"
	"strings"
)

func (a api) veiculos(c *gin.Context) {
	rows, err := a.db.QueryContext(c.Request.Context(), "SELECT id,apelido,modelo,placa,cor FROM veiculos WHERE cliente_id=$1 ORDER BY id", c.GetInt(auth.ContextUserID))
	if err != nil {
		falha(c, err)
		return
	}
	defer rows.Close()
	type carro struct {
		ID      int64  `json:"id"`
		Apelido string `json:"apelido"`
		Modelo  string `json:"modelo"`
		Placa   string `json:"placa"`
		Cor     string `json:"cor"`
	}
	lista := []carro{}
	for rows.Next() {
		var v carro
		if err := rows.Scan(&v.ID, &v.Apelido, &v.Modelo, &v.Placa, &v.Cor); err != nil {
			falha(c, err)
			return
		}
		lista = append(lista, v)
	}
	if err := rows.Err(); err != nil {
		falha(c, err)
		return
	}
	c.JSON(200, lista)
}

func (a api) adicionarVeiculo(c *gin.Context) {
	var v struct {
		Apelido string `json:"apelido"`
		Modelo  string `json:"modelo"`
		Placa   string `json:"placa"`
		Cor     string `json:"cor"`
	}
	if c.ShouldBindJSON(&v) != nil || strings.TrimSpace(v.Modelo) == "" || strings.TrimSpace(v.Placa) == "" || len(v.Apelido) > 200 || len(v.Modelo) > 100 || len(v.Placa) > 12 || len(v.Cor) > 60 {
		c.JSON(400, gin.H{"error": "Informe modelo e placa válidos"})
		return
	}
	if strings.TrimSpace(v.Apelido) == "" {
		v.Apelido = strings.TrimSpace(v.Modelo)
	}
	v.Modelo = strings.TrimSpace(v.Modelo)
	v.Apelido = strings.TrimSpace(v.Apelido)
	v.Cor = strings.TrimSpace(v.Cor)
	v.Placa = strings.ToUpper(strings.TrimSpace(v.Placa))
	var id int64
	err := a.db.QueryRowContext(c.Request.Context(), "INSERT INTO veiculos(cliente_id,apelido,modelo,placa,cor) VALUES($1,$2,$3,$4,$5) RETURNING id", c.GetInt(auth.ContextUserID), v.Apelido, v.Modelo, v.Placa, v.Cor).Scan(&id)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(409, gin.H{"error": "Esta placa já está cadastrada na sua conta"})
			return
		}
		falha(c, err)
		return
	}
	c.JSON(201, gin.H{"id": id, "apelido": v.Apelido, "modelo": v.Modelo, "placa": v.Placa, "cor": v.Cor})
}

func (a api) perfil(c *gin.Context) {
	var nome, email string
	err := a.db.QueryRowContext(c.Request.Context(), "SELECT username,email FROM usuarios WHERE id=$1", c.GetInt(auth.ContextUserID)).Scan(&nome, &email)
	if err != nil {
		falha(c, err)
		return
	}
	c.JSON(200, gin.H{"nome": nome, "email": email})
}
