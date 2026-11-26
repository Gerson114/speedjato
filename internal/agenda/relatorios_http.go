package agenda

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func (a api) resumo(c *gin.Context) {
	de, ate, ok := a.periodoRelatorio(c)
	if !ok {
		c.JSON(400, gin.H{"error": "Informe um período válido de até um ano"})
		return
	}
	var total int64
	var atendidos, cancelados, pendentes, aguardando, proximos, emLavagem int
	err := a.db.QueryRowContext(c.Request.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE status <> 'cancelado' AND lavagem_status = 'concluida'),
			COUNT(*) FILTER (WHERE status = 'cancelado'),
			COUNT(*) FILTER (WHERE status = 'pendente' AND lavagem_status <> 'concluida'),
			COUNT(*) FILTER (WHERE status = 'confirmado' AND lavagem_status = 'aguardando'),
			COUNT(*) FILTER (WHERE status = 'confirmado' AND lavagem_status = 'proximo'),
			COUNT(*) FILTER (WHERE status = 'confirmado' AND lavagem_status = 'em_andamento'),
			COALESCE(SUM(preco_centavos) FILTER (WHERE status <> 'cancelado'), 0)
		FROM agendamentos WHERE dia BETWEEN $1 AND $2`,
		de.Format(time.DateOnly), ate.Format(time.DateOnly),
	).Scan(&atendidos, &cancelados, &pendentes, &aguardando, &proximos, &emLavagem, &total)
	if err != nil {
		falha(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"carrosAtendidos": atendidos, "cancelados": cancelados, "pendentes": pendentes,
		"aguardando": aguardando, "proximos": proximos, "emLavagem": emLavagem, "totalCentavos": total, "de": de.Format(time.DateOnly), "ate": ate.Format(time.DateOnly)})
}

func (a api) relatorios(c *gin.Context) {
	de, ate, ok := a.periodoRelatorio(c)
	if !ok {
		c.JSON(400, gin.H{"error": "Informe um período válido de até um ano"})
		return
	}
	ctx := c.Request.Context()
	var relatorio struct {
		Servicos []gin.H `json:"servicosMaisVendidos"`
		Clientes []gin.H `json:"clientesRecorrentes"`
	}
	rows, err := a.db.QueryContext(ctx, `SELECT nome_servico,COUNT(*),COALESCE(SUM(preco_centavos),0) FROM agendamentos WHERE dia BETWEEN $1 AND $2 AND status <> 'cancelado' GROUP BY nome_servico ORDER BY COUNT(*) DESC,nome_servico LIMIT 5`, de.Format(time.DateOnly), ate.Format(time.DateOnly))
	if err != nil {
		falha(c, err)
		return
	}
	relatorio.Servicos = []gin.H{}
	for rows.Next() {
		var nome string
		var quantidade int
		var total int64
		if err = rows.Scan(&nome, &quantidade, &total); err != nil {
			rows.Close()
			falha(c, err)
			return
		}
		relatorio.Servicos = append(relatorio.Servicos, gin.H{"servico": nome, "quantidade": quantidade, "totalCentavos": total})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		falha(c, err)
		return
	}
	rows, err = a.db.QueryContext(ctx, `SELECT u.username,COUNT(*),COALESCE(SUM(a.preco_centavos),0) FROM agendamentos a JOIN usuarios u ON u.id=a.cliente_id WHERE a.dia BETWEEN $1 AND $2 AND a.status <> 'cancelado' GROUP BY u.id,u.username HAVING COUNT(*) > 1 ORDER BY COUNT(*) DESC,u.username LIMIT 5`, de.Format(time.DateOnly), ate.Format(time.DateOnly))
	if err != nil {
		falha(c, err)
		return
	}
	relatorio.Clientes = []gin.H{}
	for rows.Next() {
		var nome string
		var quantidade int
		var total int64
		if err = rows.Scan(&nome, &quantidade, &total); err != nil {
			rows.Close()
			falha(c, err)
			return
		}
		relatorio.Clientes = append(relatorio.Clientes, gin.H{"cliente": nome, "quantidade": quantidade, "totalCentavos": total})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		falha(c, err)
		return
	}
	rows.Close()
	c.JSON(http.StatusOK, relatorio)
}

func (a api) periodoRelatorio(c *gin.Context) (time.Time, time.Time, bool) {
	hoje, _ := a.data(time.Now().In(a.local).Format(time.DateOnly))
	switch c.Query("periodo") {
	case "hoje":
		return hoje, hoje, true
	case "7dias":
		return hoje.AddDate(0, 0, -6), hoje, true
	case "mes":
		return hoje.AddDate(0, 0, 1-hoje.Day()), hoje, true
	case "":
		de, okDe := a.data(c.Query("de"))
		ate, okAte := a.data(c.Query("ate"))
		return de, ate, okDe && okAte && !ate.Before(de) && ate.Sub(de) <= 366*24*time.Hour
	default:
		return time.Time{}, time.Time{}, false
	}
}
