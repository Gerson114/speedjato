package agenda

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"time"
)

type ProdutoServico struct {
	ID   int64  `json:"id"`
	Nome string `json:"nome"`
}

type Servico struct {
	ID            int              `json:"id"`
	Nome          string           `json:"nome"`
	Produtos      []ProdutoServico `json:"produtos"`
	ProdutoIDs    []int64          `json:"produtoIds"`
	Descricao     string           `json:"descricao"`
	Itens         []string         `json:"itens"`
	Duracao       int              `json:"duracaoMinutos"`
	PrecoCentavos int64            `json:"precoCentavos"`
}

type Reserva struct {
	Chegou        bool   `json:"chegou"`
	ID            int64  `json:"id"`
	ServicoID     int    `json:"servicoId"`
	Servico       string `json:"servico"`
	Data          string `json:"data"`
	Inicio        string `json:"inicio"`
	Fim           string `json:"fim"`
	Cliente       string `json:"cliente,omitempty"`
	Veiculo       string `json:"veiculo"`
	PrecoCentavos int64  `json:"precoCentavos"`
	Status        string `json:"status"`
	LavagemStatus string `json:"lavagemStatus"`
	Email         string `json:"email,omitempty"`
	Modelo        string `json:"modelo,omitempty"`
	Placa         string `json:"placa,omitempty"`
	Cor           string `json:"cor,omitempty"`
}

type api struct {
	db    *sql.DB
	local *time.Location
}

type leitor interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func lerConfig(ctx context.Context, q leitor, bloquear bool) (Configuracao, error) {
	query := "SELECT capacidade, intervalo, horarios, tolerancia_minutos FROM agenda_config WHERE id = 1"
	if bloquear {
		query += " FOR UPDATE"
	}
	var c Configuracao
	var raw []byte
	if err := q.QueryRowContext(ctx, query).Scan(&c.Capacidade, &c.Intervalo, &raw, &c.Tolerancia); err != nil {
		return c, err
	}
	err := json.Unmarshal(raw, &c.Horarios)
	return c, err
}

func lerServico(ctx context.Context, tx *sql.Tx, id int, bloquear bool) (Servico, error) {
	query := "SELECT id, nome, duracao, preco_centavos FROM agenda_servicos WHERE id = $1 AND NOT excluido"
	if bloquear {
		query += " FOR SHARE"
	}
	var s Servico
	err := tx.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.Nome, &s.Duracao, &s.PrecoCentavos)
	return s, err
}

func vagasDoDia(ctx context.Context, tx *sql.Tx, cfg Configuracao, dia time.Time, servico Servico) ([]string, error) {
	reservas, err := reservasDoDia(ctx, tx, dia)
	if err != nil {
		return nil, err
	}
	return cfg.vagas(dia, servico.Duracao, reservas, time.Now()), nil
}

func reservasDoDia(ctx context.Context, tx *sql.Tx, dia time.Time) ([]periodo, error) {
	rows, err := tx.QueryContext(ctx, "SELECT inicio, fim FROM agendamentos WHERE dia = $1 AND status <> 'cancelado'", dia.Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reservas []periodo
	for rows.Next() {
		var p periodo
		if err := rows.Scan(&p.inicio, &p.fim); err != nil {
			return nil, err
		}
		reservas = append(reservas, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reservas, nil
}

var errOcupado = errors.New("Horário indisponível")

func (a api) confirmar(ctx context.Context, cliente, servicoID int, veiculoID int64, dia time.Time, hora string) (Reserva, bool, error) {
	var r Reserva
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return r, false, err
	}
	defer tx.Rollback()
	return a.finalizarReserva(ctx, tx, cliente, servicoID, veiculoID, dia, hora)
}

// finalizarReserva confirma a transação recebida somente após validar a vaga
// e gravar a reserva e seu aviso. O chamador cuida do rollback em caso de erro.
func (a api) finalizarReserva(ctx context.Context, tx *sql.Tx, cliente, servicoID int, veiculoID int64, dia time.Time, hora string) (Reserva, bool, error) {
	var r Reserva
	// A configuração é compartilhada por todos os serviços do lava-jato. Depois
	// de obter o lock, a consulta de ocupação já vê a última reserva confirmada.
	cfg, err := lerConfig(ctx, tx, true)
	if err != nil {
		return r, false, err
	}
	s, err := lerServico(ctx, tx, servicoID, true)
	if err != nil {
		return r, false, err
	}
	var veiculo string
	if err := tx.QueryRowContext(ctx, "SELECT apelido FROM veiculos WHERE id=$1 AND cliente_id=$2", veiculoID, cliente).Scan(&veiculo); err != nil {
		return r, false, err
	}
	r.Veiculo = veiculo

	data := dia.Format(time.DateOnly)
	inicio, _ := minuto(hora)
	r = Reserva{ServicoID: s.ID, Data: data, Inicio: hora, Veiculo: veiculo, PrecoCentavos: s.PrecoCentavos}
	var fim int
	var status string
	err = tx.QueryRowContext(ctx, `SELECT id, nome_servico, fim, status, lavagem_status, preco_centavos FROM agendamentos
		WHERE cliente_id=$1 AND servico_id=$2 AND veiculo_id=$3 AND dia=$4 AND inicio=$5 AND status <> 'cancelado'`,
		cliente, s.ID, veiculoID, data, inicio).Scan(&r.ID, &r.Servico, &fim, &status, &r.LavagemStatus, &r.PrecoCentavos)
	if err == nil {
		r.Fim = relogio(fim)
		r.Veiculo = veiculo
		if status != "cancelado" {
			r.Status = status
			return r, false, tx.Commit()
		}
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return r, false, err
	}
	vagas, err := vagasDoDia(ctx, tx, cfg, dia, s)
	if err != nil {
		return r, false, err
	}
	if !slices.Contains(vagas, hora) {
		return r, false, errOcupado
	}
	fim = inicio + s.Duracao
	r.Servico, r.Fim = s.Nome, relogio(fim)
	err = tx.QueryRowContext(ctx, `INSERT INTO agendamentos
 (cliente_id,servico_id,veiculo_id,veiculo,preco_centavos,nome_servico,dia,inicio,fim,status,lavagem_status)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'pendente','aguardando') RETURNING id`,
		cliente, s.ID, veiculoID, veiculo, s.PrecoCentavos, s.Nome, data, inicio, fim).Scan(&r.ID)
	r.Status = "pendente"
	r.LavagemStatus = "aguardando"
	r.PrecoCentavos = s.PrecoCentavos
	if err != nil {
		return r, false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO avisos_cliente(cliente_id,agendamento_id,tipo,mensagem) VALUES($1,$2,'agendamento_recebido',$3) ON CONFLICT DO NOTHING`, cliente, r.ID, "Recebemos seu pedido de agendamento para "+data+" às "+hora+". Ele aguarda confirmação do lava-jato.")
	if err != nil {
		return r, false, err
	}
	if err := tx.Commit(); err != nil {
		return r, false, err
	}
	return r, true, nil
}
