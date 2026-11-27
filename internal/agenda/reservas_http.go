package agenda

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"lava/internal/registro/auth"
	"lava/internal/service"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

func (a api) reservasDoDia(c *gin.Context) {
	if c.Query("visao") == "historico" {
		filtro := "WHERE (a.status='cancelado' OR a.lavagem_status='concluida')"
		switch c.Query("categoria") {
		case "concluidos":
			filtro = "WHERE a.status <> 'cancelado' AND a.lavagem_status='concluida'"
		case "cancelados":
			filtro = "WHERE a.status='cancelado'"
		}
		var args []any
		if busca := strings.TrimSpace(c.Query("busca")); busca != "" {
			filtro += " AND strpos(lower(concat_ws(' ',u.username,v.placa,v.modelo,a.veiculo,a.nome_servico)),lower($1::text)) > 0"
			args = append(args, busca)
		}
		a.listar(c, filtro+" ORDER BY a.dia DESC,a.inicio DESC,a.id DESC", args...)
		return
	}
	dia, ok := a.data(c.Query("data"))
	if !ok {
		c.JSON(400, gin.H{"error": "Informe a data (AAAA-MM-DD)"})
		return
	}
	condicoes := []string{"a.dia = $1"}
	if c.Query("visao") == "ativos" {
		condicoes = append(condicoes, "a.status <> 'cancelado' AND a.lavagem_status <> 'concluida'")
	}
	valores := []any{dia.Format(time.DateOnly)}
	for _, filtro := range []struct{ query, column string }{{"cliente", "u.username"}, {"placa", "v.placa"}, {"servico", "a.nome_servico"}} {
		valor := strings.TrimSpace(c.Query(filtro.query))
		if valor != "" {
			valores = append(valores, "%"+valor+"%")
			condicoes = append(condicoes, fmt.Sprintf("%s ILIKE $%d", filtro.column, len(valores)))
		}
	}
	if status := c.Query("status"); status == "pendente" || status == "confirmado" || status == "cancelado" {
		valores = append(valores, status)
		condicoes = append(condicoes, fmt.Sprintf("a.status = $%d", len(valores)))
	}
	if lavagem := c.Query("lavagem"); lavagem == "aguardando" || lavagem == "proximo" || lavagem == "em_andamento" || lavagem == "concluida" {
		valores = append(valores, lavagem)
		condicoes = append(condicoes, fmt.Sprintf("a.lavagem_status = $%d", len(valores)))
	}
	a.listar(c, "WHERE "+strings.Join(condicoes, " AND ")+" ORDER BY a.dia,a.inicio,a.id", valores...)
}

func (a api) minhasReservas(c *gin.Context) {
	args := []any{c.GetInt(auth.ContextUserID)}
	where := "WHERE a.cliente_id = $1"
	if placa := strings.TrimSpace(c.Query("placa")); placa != "" {
		args = append(args, placa)
		where += fmt.Sprintf(" AND v.placa = $%d", len(args))
	}
	if veiculo := strings.TrimSpace(c.Query("veiculoId")); veiculo != "" {
		if id, err := strconv.ParseInt(veiculo, 10, 64); err == nil && id > 0 {
			args = append(args, id)
			where += fmt.Sprintf(" AND a.veiculo_id = $%d", len(args))
		}
	}
	a.listar(c, where+" ORDER BY a.dia DESC,a.inicio DESC,a.id DESC", args...)
}

func (a api) listar(c *gin.Context, filtro string, valores ...any) {
	{
		pagina := 1
		if valor := c.Query("pagina"); valor != "" {
			n, err := strconv.Atoi(valor)
			if err != nil || n < 1 || n > 100000 {
				c.JSON(400, gin.H{"error": "Página inválida"})
				return
			}
			pagina = n
		}
		valores = append(valores, 51, (pagina-1)*50)
		filtro += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(valores)-1, len(valores))
	}
	rows, err := a.db.QueryContext(c.Request.Context(), `SELECT a.id,a.servico_id,a.nome_servico,to_char(a.dia,'YYYY-MM-DD'),a.inicio,a.fim,u.username,u.email,a.veiculo,a.preco_centavos,a.status,a.lavagem_status,a.chegou,
		COALESCE(v.modelo,''),COALESCE(v.placa,''),COALESCE(v.cor,'')
		FROM agendamentos a JOIN usuarios u ON u.id=a.cliente_id LEFT JOIN veiculos v ON v.id=a.veiculo_id `+filtro, valores...)
	if err != nil {
		falha(c, err)
		return
	}
	defer rows.Close()
	lista := []itemAgenda{}
	for rows.Next() {
		var r Reserva
		var inicio, fim int
		var email, modelo, placa, cor string
		if err := rows.Scan(&r.ID, &r.ServicoID, &r.Servico, &r.Data, &inicio, &fim, &r.Cliente, &email, &r.Veiculo, &r.PrecoCentavos, &r.Status, &r.LavagemStatus, &r.Chegou, &modelo, &placa, &cor); err != nil {
			falha(c, err)
			return
		}
		r.Email, r.Modelo, r.Placa, r.Cor = email, modelo, placa, cor
		r.Inicio, r.Fim = relogio(inicio), relogio(fim)
		lista = append(lista, apresentarReserva(r))
	}
	if err := rows.Err(); err != nil {
		falha(c, err)
		return
	}
	c.JSON(http.StatusOK, lista)
}

func (a api) atualizarStatus(c *gin.Context) {
	id, ok := idPositivo(c.Param("id"))
	if !ok {
		c.JSON(400, gin.H{"error": "Agendamento inválido"})
		return
	}
	var pedido struct {
		Status string `json:"status"`
	}
	if c.ShouldBindJSON(&pedido) != nil {
		c.JSON(400, gin.H{"error": service.ErrStatus.Error()})
		return
	}
	servico := service.Agendamentos{DB: a.db}
	if err := servico.AtualizarStatus(c.Request.Context(), id, pedido.Status); err != nil {
		responderErroAndamento(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "status": pedido.Status})
}

func (a api) atualizarLavagem(c *gin.Context) {
	id, ok := idPositivo(c.Param("id"))
	if !ok {
		c.JSON(400, gin.H{"error": "Agendamento inválido"})
		return
	}
	var pedido struct {
		Status string `json:"status"`
	}
	if c.ShouldBindJSON(&pedido) != nil {
		c.JSON(400, gin.H{"error": service.ErrEtapa.Error()})
		return
	}
	servico := service.Agendamentos{DB: a.db}
	if err := servico.AtualizarLavagem(c.Request.Context(), id, pedido.Status); err != nil {
		responderErroAndamento(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "lavagemStatus": pedido.Status})
}

func responderErroAndamento(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrStatus), errors.Is(err, service.ErrEtapa):
		c.JSON(400, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrIndisponivel), errors.Is(err, service.ErrProximo):
		c.JSON(409, gin.H{"error": err.Error()})
	default:
		falha(c, err)
	}
}

func (a api) agendar(c *gin.Context) {
	var pedido struct {
		ServicoID int    `json:"servicoId" binding:"min=1,max=2147483647"`
		Data      string `json:"data"`
		Inicio    string `json:"inicio"`
		VeiculoID int64  `json:"veiculoId"`
	}
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(400, gin.H{"error": "Pedido de agendamento inválido"})
		return
	}
	dia, dataValida := a.data(pedido.Data)
	m, horaValida := minuto(pedido.Inicio)
	if !dataValida || !horaValida || m >= 1440 || pedido.VeiculoID < 1 {
		c.JSON(400, gin.H{"error": "Informe data (AAAA-MM-DD) e início (HH:MM)"})
		return
	}
	reserva, nova, err := a.confirmar(c.Request.Context(), c.GetInt(auth.ContextUserID), pedido.ServicoID, pedido.VeiculoID, dia, pedido.Inicio)
	if err != nil {
		falha(c, err)
		return
	}
	if nova {
		c.JSON(http.StatusCreated, apresentarReserva(reserva))
		return
	}
	c.JSON(http.StatusOK, apresentarReserva(reserva))
}

func (a api) agendarBalcao(c *gin.Context) {
	var p struct {
		Nome      string `json:"nome"`
		Email     string `json:"email"`
		Modelo    string `json:"modelo"`
		Placa     string `json:"placa"`
		Cor       string `json:"cor"`
		ServicoID int    `json:"servicoId"`
		Data      string `json:"data"`
		Inicio    string `json:"inicio"`
	}
	if c.ShouldBindJSON(&p) != nil {
		c.JSON(400, gin.H{"error": "Dados do agendamento inválidos"})
		return
	}
	p.Nome = strings.TrimSpace(p.Nome)
	p.Email = strings.ToLower(strings.TrimSpace(p.Email))
	p.Modelo = strings.TrimSpace(p.Modelo)
	p.Placa = strings.ToUpper(strings.TrimSpace(p.Placa))
	p.Cor = strings.TrimSpace(p.Cor)
	endereco, errEmail := mail.ParseAddress(p.Email)
	if errEmail != nil || endereco.Address != p.Email || len(p.Cor) > 40 || p.Nome == "" || p.Email == "" || p.Modelo == "" || p.Placa == "" || p.ServicoID < 1 || len(p.Nome) > 120 || len(p.Email) > 254 || len(p.Modelo) > 100 || len(p.Placa) > 20 {
		c.JSON(400, gin.H{"error": "Preencha cliente, e-mail, veículo e placa"})
		return
	}
	dia, ok := a.data(p.Data)
	minutoInicio, horaOk := minuto(p.Inicio)
	if !ok || !horaOk || minutoInicio >= 1440 {
		c.JSON(400, gin.H{"error": "Data ou horário inválido"})
		return
	}
	ctx := c.Request.Context()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		falha(c, err)
		return
	}
	defer tx.Rollback()
	var cliente int
	var papel string
	var liberado bool
	err = tx.QueryRowContext(ctx, "SELECT id, role, liberado FROM usuarios WHERE lower(email)=lower($1) FOR SHARE", p.Email).Scan(&cliente, &papel, &liberado)
	if err == nil && (papel != "user" || !liberado) {
		c.JSON(403, gin.H{"error": "Esta conta não pode receber agendamentos"})
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		aleatorio := make([]byte, 32)
		if _, err = rand.Read(aleatorio); err != nil {
			falha(c, err)
			return
		}
		hash, e := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(aleatorio)), bcrypt.DefaultCost)
		if e != nil {
			falha(c, e)
			return
		}
		err = tx.QueryRowContext(ctx, "INSERT INTO usuarios(username,email,password,role,liberado) VALUES($1,$2,$3,'user',true) RETURNING id", p.Nome, p.Email, string(hash)).Scan(&cliente)
	}
	if err != nil {
		falha(c, err)
		return
	}
	var veiculo int64
	err = tx.QueryRowContext(ctx, "INSERT INTO veiculos(cliente_id,apelido,modelo,placa,cor) VALUES($1,$2,$3,$4,$5) ON CONFLICT(cliente_id,placa) DO UPDATE SET apelido=EXCLUDED.apelido,modelo=EXCLUDED.modelo,cor=EXCLUDED.cor RETURNING id", cliente, p.Modelo, p.Modelo, p.Placa, p.Cor).Scan(&veiculo)
	if err != nil {
		falha(c, err)
		return
	}
	reserva, _, err := a.finalizarReserva(ctx, tx, cliente, p.ServicoID, veiculo, dia, p.Inicio)
	if err != nil {
		falha(c, err)
		return
	}
	c.JSON(http.StatusCreated, apresentarReserva(reserva))
}

func (a api) registrarChegada(c *gin.Context) {
	id, ok := idPositivo(c.Param("id"))
	if !ok {
		c.JSON(400, gin.H{"error": "Agendamento inválido"})
		return
	}
	result, err := a.db.ExecContext(c.Request.Context(), "UPDATE agendamentos SET chegou=true WHERE id=$1 AND status='confirmado' AND lavagem_status <> 'concluida'", id)
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
		c.JSON(409, gin.H{"error": "Confirme o agendamento antes de registrar a chegada."})
		return
	}
	c.JSON(200, gin.H{"chegou": true})
}
