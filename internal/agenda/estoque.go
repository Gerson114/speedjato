package agenda

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"net/http"
	"strconv"
	"strings"
)

type produto struct {
	ID         int64   `json:"id"`
	Nome       string  `json:"nome"`
	Unidade    string  `json:"unidade"`
	Quantidade float64 `json:"quantidade"`
	Consumo    float64 `json:"consumoDiario"`
	Dias       float64 `json:"diasRestantes"`
}

func (a api) listarEstoque(c *gin.Context) {
	rows, err := a.db.QueryContext(c.Request.Context(), `SELECT id,nome,unidade,quantidade,consumo_diario FROM estoque_produtos ORDER BY nome`)
	if err != nil {
		falha(c, err)
		return
	}
	defer rows.Close()
	itens := []produto{}
	for rows.Next() {
		var p produto
		if err = rows.Scan(&p.ID, &p.Nome, &p.Unidade, &p.Quantidade, &p.Consumo); err != nil {
			falha(c, err)
			return
		}
		if p.Consumo > 0 {
			p.Dias = p.Quantidade / p.Consumo
		}
		itens = append(itens, p)
	}
	if err = rows.Err(); err != nil {
		falha(c, err)
		return
	}
	c.JSON(http.StatusOK, itens)
}
func (a api) salvarEstoque(c *gin.Context) {
	var in struct {
		Nome       string  `json:"nome"`
		Unidade    string  `json:"unidade"`
		Quantidade float64 `json:"quantidade"`
		Consumo    float64 `json:"consumoDiario"`
	}
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.Nome) == "" || strings.TrimSpace(in.Unidade) == "" || in.Quantidade < 0 || in.Consumo < 0 {
		c.JSON(400, gin.H{"error": "Informe nome, unidade e valores válidos"})
		return
	}
	if len(in.Nome) > 100 || len(in.Unidade) > 20 || in.Quantidade > 1e9 || in.Consumo > 1e9 {
		c.JSON(400, gin.H{"error": "Valores fora do limite"})
		return
	}
	var p produto
	err := a.db.QueryRowContext(c.Request.Context(), `INSERT INTO estoque_produtos(nome,unidade,quantidade,consumo_diario) VALUES($1,$2,$3,$4) RETURNING id,nome,unidade,quantidade,consumo_diario`, strings.TrimSpace(in.Nome), strings.TrimSpace(in.Unidade), in.Quantidade, in.Consumo).Scan(&p.ID, &p.Nome, &p.Unidade, &p.Quantidade, &p.Consumo)
	if err != nil {
		falha(c, err)
		return
	}
	if p.Consumo > 0 {
		p.Dias = p.Quantidade / p.Consumo
	}
	c.JSON(201, p)
}
func (a api) consumirEstoque(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	var in struct {
		Quantidade float64 `json:"quantidade"`
	}
	if err != nil || id < 1 || c.ShouldBindJSON(&in) != nil || in.Quantidade <= 0 || in.Quantidade > 1e9 {
		c.JSON(400, gin.H{"error": "Quantidade inválida"})
		return
	}
	result, err := a.db.ExecContext(c.Request.Context(), `UPDATE estoque_produtos SET quantidade=quantidade-$1 WHERE id=$2 AND quantidade >= $1`, in.Quantidade, id)
	if err != nil {
		falha(c, err)
		return
	}
	n, err := result.RowsAffected()
	if err != nil {
		falha(c, err)
		return
	}
	if n == 0 {
		c.JSON(409, gin.H{"error": "Produto não encontrado ou estoque insuficiente"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (a api) excluirProduto(c *gin.Context) {
	id, ok := idPositivo(c.Param("id"))
	if !ok {
		c.JSON(400, gin.H{"error": "Produto inválido"})
		return
	}
	result, err := a.db.ExecContext(c.Request.Context(), "DELETE FROM estoque_produtos WHERE id=$1", id)
	if err != nil {
		var dbErr *pq.Error
		if errors.As(err, &dbErr) && dbErr.Code == "23503" {
			c.JSON(409, gin.H{"error": "Remova o produto dos serviços vinculados antes de excluí-lo."})
			return
		}
		falha(c, err)
		return
	}
	n, err := result.RowsAffected()
	if err != nil {
		falha(c, err)
		return
	}
	if n == 0 {
		c.JSON(404, gin.H{"error": "Produto não encontrado"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
