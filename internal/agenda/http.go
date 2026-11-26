package agenda

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"lava/internal/registro/auth"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// Registrar compartilha a agenda do estabelecimento entre os dois painéis.
func Registrar(r *gin.Engine, db *sql.DB, local *time.Location) {
	a := api{db: db, local: local}
	admin := r.Group("/admin", auth.Middleware(db), exigirPapel(auth.RoleAdmin))
	admin.GET("/agenda/configuracao", a.configuracao)
	admin.PUT("/agenda/configuracao", a.salvarConfiguracao)
	admin.GET("/servicos", a.servicos)
	admin.GET("/estoque", a.listarEstoque)
	admin.POST("/estoque", a.salvarEstoque)
	admin.DELETE("/estoque/:id", a.excluirProduto)
	admin.POST("/estoque/:id/consumo", a.consumirEstoque)
	admin.POST("/servicos", a.salvarServico)
	admin.PUT("/servicos/:id", a.salvarServico)
	admin.DELETE("/servicos/:id", a.excluirServico)
	admin.GET("/agendamentos/novidades", a.aguardarPedido)
	admin.GET("/agendamentos", a.reservasDoDia)
	admin.GET("/horarios", a.horarios)
	admin.POST("/agendamentos/balcao", a.agendarBalcao)
	admin.PUT("/agendamentos/:id/chegada", a.registrarChegada)
	admin.PUT("/agendamentos/:id/status", a.atualizarStatus)
	if os.Getenv("APP_ENV") != "production" {
		admin.POST("/agendamentos/teste", a.agendamentoTeste)
	}
	admin.GET("/resumo", a.resumo)
	admin.GET("/relatorios", a.relatorios)
	admin.PUT("/agendamentos/:id/lavagem", a.atualizarLavagem)
	cliente := r.Group("/usuario", auth.Middleware(db), exigirPapel("user"))
	r.GET("/usuario/servicos", a.servicos)
	cliente.GET("/perfil", a.perfil)
	cliente.GET("/veiculos", a.veiculos)
	cliente.POST("/veiculos", a.adicionarVeiculo)
	cliente.GET("/horarios", a.horarios)
	cliente.GET("/disponibilidade", a.disponibilidade)
	cliente.POST("/agendamentos", a.agendar)
	cliente.GET("/agendamentos", a.minhasReservas)
	cliente.GET("/avisos", a.avisos)
	cliente.POST("/sair", a.sair)
	admin.POST("/sair", a.sair)
}

func exigirPapel(papel string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(auth.ContextRole) != papel {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Acesso não permitido a este painel"})
			return
		}
		c.Next()
	}
}

func falha(c *gin.Context, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		c.JSON(http.StatusNotFound, gin.H{"error": "Registro não encontrado"})
	case errors.Is(err, errOcupado):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		slog.Error("falha na agenda", "rota", c.FullPath(), "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao acessar a agenda"})
	}
}

func (a api) configuracao(c *gin.Context) {
	cfg, err := lerConfig(c.Request.Context(), a.db, false)
	if err != nil {
		falha(c, err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (a api) salvarConfiguracao(c *gin.Context) {
	var cfg Configuracao
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(400, gin.H{"error": "Configuração inválida"})
		return
	}
	if err := cfg.validar(); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	horarios, err := json.Marshal(cfg.Horarios)
	if err != nil {
		falha(c, err)
		return
	}
	_, err = a.db.ExecContext(c.Request.Context(), "UPDATE agenda_config SET capacidade=$1, intervalo=$2, horarios=$3, tolerancia_minutos=$4 WHERE id=1", cfg.Capacidade, cfg.Intervalo, string(horarios), cfg.Tolerancia)
	if err != nil {
		falha(c, err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

const consultaServicos = `SELECT s.id,s.nome,s.duracao,s.preco_centavos,s.descricao,s.itens,
 COALESCE((SELECT json_agg(json_build_object('id',p.id,'nome',p.nome) ORDER BY p.nome,p.id) FROM servico_produtos sp JOIN estoque_produtos p ON p.id=sp.produto_id WHERE sp.servico_id=s.id),'[]'::json)
 FROM agenda_servicos s`

func lerServicoDoCatalogo(row interface{ Scan(...any) error }) (Servico, error) {
	var s Servico
	var produtos []byte
	if err := row.Scan(&s.ID, &s.Nome, &s.Duracao, &s.PrecoCentavos, &s.Descricao, pq.Array(&s.Itens), &produtos); err != nil {
		return s, err
	}
	if err := json.Unmarshal(produtos, &s.Produtos); err != nil {
		return s, err
	}
	s.ProdutoIDs = make([]int64, 0, len(s.Produtos))
	for _, p := range s.Produtos {
		s.ProdutoIDs = append(s.ProdutoIDs, p.ID)
	}
	return s, nil
}

func (a api) servicos(c *gin.Context) {
	rows, err := a.db.QueryContext(c.Request.Context(), consultaServicos+" WHERE NOT s.excluido ORDER BY s.nome,s.id")
	if err != nil {
		falha(c, err)
		return
	}
	defer rows.Close()
	lista := []Servico{}
	for rows.Next() {
		s, err := lerServicoDoCatalogo(rows)
		if err != nil {
			falha(c, err)
			return
		}
		lista = append(lista, s)
	}
	if err := rows.Err(); err != nil {
		falha(c, err)
		return
	}
	c.JSON(http.StatusOK, lista)
}

func (a api) excluirServico(c *gin.Context) {
	id, ok := idPositivo(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Serviço inválido"})
		return
	}
	ctx := c.Request.Context()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		falha(c, err)
		return
	}
	defer tx.Rollback()
	var removido int
	if err = tx.QueryRowContext(ctx, "UPDATE agenda_servicos SET excluido=true WHERE id=$1 RETURNING id", id).Scan(&removido); err != nil {
		falha(c, err)
		return
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM servico_produtos WHERE servico_id=$1", id); err != nil {
		falha(c, err)
		return
	}
	if err = tx.Commit(); err != nil {
		falha(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a api) salvarServico(c *gin.Context) {
	var s struct {
		Servico
		ChaveCriacao string `json:"chaveCriacao" binding:"max=64"`
	}
	if err := c.ShouldBindJSON(&s); err != nil || strings.TrimSpace(s.Nome) == "" || len(s.Nome) > 200 || s.Duracao < 1 || s.Duracao > 1440 || s.PrecoCentavos < 0 {
		c.JSON(400, gin.H{"error": "Informe o nome, duração e preço válido do serviço"})
		return
	}
	s.Nome = strings.TrimSpace(s.Nome)
	s.Descricao = strings.TrimSpace(s.Descricao)
	if len(s.Descricao) > 1000 || len(s.Itens) > 20 {
		c.JSON(400, gin.H{"error": "Descrição ou lista de itens excede o limite"})
		return
	}
	for i, item := range s.Itens {
		s.Itens[i] = strings.TrimSpace(item)
		if s.Itens[i] == "" || len(s.Itens[i]) > 200 {
			c.JSON(400, gin.H{"error": "Cada item deve ter entre 1 e 200 caracteres"})
			return
		}
	}
	if s.Itens == nil {
		s.Itens = []string{}
	}
	ctx := c.Request.Context()
	if len(s.ProdutoIDs) > 100 {
		c.JSON(400, gin.H{"error": "Muitos produtos"})
		return
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		falha(c, err)
		return
	}
	defer tx.Rollback()
	status := http.StatusCreated
	if c.Request.Method == http.MethodPost {
		err = tx.QueryRowContext(ctx, `INSERT INTO agenda_servicos (nome,duracao,preco_centavos,descricao,itens,chave_criacao)
 VALUES ($1,$2,$3,$4,$5,NULLIF($6,'')) ON CONFLICT (chave_criacao) DO NOTHING RETURNING id`, s.Nome, s.Duracao, s.PrecoCentavos, s.Descricao, pq.Array(s.Itens), s.ChaveCriacao).Scan(&s.ID)
		if errors.Is(err, sql.ErrNoRows) {
			anterior, err := lerServicoDoCatalogo(tx.QueryRowContext(ctx, consultaServicos+" WHERE s.chave_criacao=$1 AND NOT s.excluido", s.ChaveCriacao))
			if err != nil {
				falha(c, err)
				return
			}
			c.JSON(http.StatusOK, anterior)
			return
		}
	} else {
		id, ok := idPositivo(c.Param("id"))
		if !ok {
			c.JSON(400, gin.H{"error": "Serviço inválido"})
			return
		}
		err = tx.QueryRowContext(ctx, "UPDATE agenda_servicos SET nome=$1,duracao=$2,preco_centavos=$3,descricao=$4,itens=$5 WHERE id=$6 AND NOT excluido RETURNING id", s.Nome, s.Duracao, s.PrecoCentavos, s.Descricao, pq.Array(s.Itens), id).Scan(&s.ID)
		status = http.StatusOK
	}
	if err != nil {
		falha(c, err)
		return
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM servico_produtos WHERE servico_id=$1", s.ID); err != nil {
		falha(c, err)
		return
	}
	vistos := map[int64]bool{}
	for _, id := range s.ProdutoIDs {
		if id < 1 || vistos[id] {
			c.JSON(400, gin.H{"error": "Produto inválido ou duplicado"})
			return
		}
		vistos[id] = true
		var nome string
		if err = tx.QueryRowContext(ctx, "SELECT nome FROM estoque_produtos WHERE id=$1", id).Scan(&nome); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(400, gin.H{"error": "Produto não encontrado"})
				return
			}
			falha(c, err)
			return
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO servico_produtos(servico_id,produto_id) VALUES($1,$2)", s.ID, id); err != nil {
			falha(c, err)
			return
		}
		s.Produtos = append(s.Produtos, ProdutoServico{ID: id, Nome: nome})
	}
	if s.Produtos == nil {
		s.Produtos = []ProdutoServico{}
	}
	if s.ProdutoIDs == nil {
		s.ProdutoIDs = []int64{}
	}
	if err = tx.Commit(); err != nil {
		falha(c, err)
		return
	}
	c.JSON(status, s.Servico)
}

func (a api) data(valor string) (time.Time, bool) {
	dia, err := time.ParseInLocation(time.DateOnly, valor, a.local)
	return dia, err == nil && dia.Year() > 0
}

func (a api) horarios(c *gin.Context) {
	id, ok := idPositivo(c.Query("servicoId"))
	dia, dataValida := a.data(c.Query("data"))
	if !ok || !dataValida {
		c.JSON(400, gin.H{"error": "Informe servicoId e data (AAAA-MM-DD)"})
		return
	}
	ctx := c.Request.Context()
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		falha(c, err)
		return
	}
	defer tx.Rollback()
	cfg, err := lerConfig(ctx, tx, false)
	if err != nil {
		falha(c, err)
		return
	}
	s, err := lerServico(ctx, tx, id, false)
	if err != nil {
		falha(c, err)
		return
	}
	vagas, err := vagasDoDia(ctx, tx, cfg, dia, s)
	if err != nil {
		falha(c, err)
		return
	}
	c.JSON(http.StatusOK, vagas)
}

func (a api) disponibilidade(c *gin.Context) {
	id, ok := idPositivo(c.Query("servicoId"))
	dia, dataValida := a.data(c.Query("data"))
	if !ok || !dataValida {
		c.JSON(400, gin.H{"error": "Informe serviço e data"})
		return
	}
	ctx := c.Request.Context()
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		falha(c, err)
		return
	}
	defer tx.Rollback()
	cfg, err := lerConfig(ctx, tx, false)
	if err != nil {
		falha(c, err)
		return
	}
	s, err := lerServico(ctx, tx, id, false)
	if err != nil {
		falha(c, err)
		return
	}
	reservas, err := reservasDoDia(ctx, tx, dia)
	if err != nil {
		falha(c, err)
		return
	}
	agora := time.Now().In(a.local)
	todos := cfg.horarios(dia, s.Duracao, agora)
	vagas := cfg.vagas(dia, s.Duracao, reservas, agora)
	livres := make(map[string]bool, len(vagas))
	for _, vaga := range vagas {
		livres[vaga] = true
	}
	ocupados := []string{}
	for _, horario := range todos {
		if !livres[horario] {
			ocupados = append(ocupados, horario)
		}
	}
	c.JSON(200, gin.H{"horarios": todos, "disponiveis": vagas, "ocupados": ocupados})
}

func (a api) avisos(c *gin.Context) {
	ctx := c.Request.Context()
	cliente := c.GetInt(auth.ContextUserID)
	agora := time.Now().In(a.local)
	_, err := a.db.ExecContext(ctx, `INSERT INTO avisos_cliente(cliente_id,agendamento_id,tipo,mensagem)
		SELECT $1, a.id, 'lembrete_24h', 'Lembrete: sua lavagem está marcada para ' ||
		       to_char(a.dia,'YYYY-MM-DD') || ' às ' || lpad((a.inicio / 60)::text,2,'0') || ':' || lpad((a.inicio % 60)::text,2,'0') || '.'
		FROM agendamentos a
		WHERE a.cliente_id=$1 AND a.status='confirmado' AND a.lavagem_status<>'concluida'
		  AND a.dia BETWEEN $2::date AND $3::date
		  AND (a.dia + a.inicio * interval '1 minute') AT TIME ZONE $4 > $5::timestamptz
		  AND (a.dia + a.inicio * interval '1 minute') AT TIME ZONE $4 <= $5::timestamptz + interval '24 hours'
		  AND NOT EXISTS (SELECT 1 FROM avisos_cliente aviso WHERE aviso.agendamento_id=a.id AND aviso.tipo='lembrete_24h')
		ON CONFLICT (agendamento_id,tipo) DO NOTHING`, cliente, agora.Format(time.DateOnly), agora.Add(24*time.Hour).Format(time.DateOnly), a.local.String(), agora)
	if err != nil {
		falha(c, err)
		return
	}
	rows, err := a.db.QueryContext(ctx, `SELECT id,mensagem,to_char(criado_em AT TIME ZONE $2,'YYYY-MM-DD HH24:MI'),lido FROM avisos_cliente WHERE cliente_id=$1 ORDER BY criado_em DESC LIMIT 30`, cliente, a.local.String())
	if err != nil {
		falha(c, err)
		return
	}
	defer rows.Close()
	avisos := []gin.H{}
	for rows.Next() {
		var id int64
		var mensagem, data string
		var lido bool
		if err = rows.Scan(&id, &mensagem, &data, &lido); err != nil {
			falha(c, err)
			return
		}
		avisos = append(avisos, gin.H{"id": id, "mensagem": mensagem, "criadoEm": data, "lido": lido})
	}
	if err = rows.Err(); err != nil {
		falha(c, err)
		return
	}
	c.JSON(200, avisos)
}

func (a api) sair(c *gin.Context) {
	_, err := a.db.ExecContext(c.Request.Context(), "UPDATE usuarios SET sessao_versao=sessao_versao+1 WHERE id=$1", c.GetInt(auth.ContextUserID))
	if err != nil {
		falha(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
