package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

var (
	ErrStatus       = errors.New("Escolha confirmar ou cancelar o agendamento")
	ErrEtapa        = errors.New("Etapa de lavagem inválida")
	ErrIndisponivel = errors.New("Esta ação não está disponível para o estado atual do agendamento")
	ErrProximo      = errors.New("Já existe um próximo carro neste dia. Atualize esse atendimento primeiro.")
)

type Agendamentos struct{ DB *sql.DB }

func (s Agendamentos) AtualizarStatus(ctx context.Context, id int, status string) error {
	var aviso string
	switch status {
	case "confirmado":
		aviso = "Seu agendamento foi confirmado pelo lava-jato."
	case "cancelado":
		aviso = "Seu agendamento foi cancelado pelo lava-jato."
	default:
		return ErrStatus
	}
	return s.atualizar(ctx, id, status, "UPDATE agendamentos SET status=$1 WHERE id=$2 AND status <> 'cancelado' AND lavagem_status <> 'concluida'", "agendamento_"+status, aviso)
}

func (s Agendamentos) AtualizarLavagem(ctx context.Context, id int, etapa string) error {
	var aviso string
	switch etapa {
	case "aguardando":
		aviso = "Seu veículo está aguardando o início da lavagem."
	case "proximo":
		aviso = "Seu veículo é o próximo na fila de lavagem."
	case "em_andamento":
		aviso = "A lavagem do seu veículo começou."
	case "concluida":
		aviso = "A lavagem do seu veículo foi concluída."
	default:
		return ErrEtapa
	}
	return s.atualizar(ctx, id, etapa, "UPDATE agendamentos SET lavagem_status=$1 WHERE id=$2 AND status = 'confirmado' AND chegou = true AND lavagem_status <> 'concluida'", "lavagem_"+etapa, aviso)
}

func (s Agendamentos) atualizar(ctx context.Context, id int, valor, consulta, tipo, aviso string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, consulta, valor, id)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Constraint == "agendamentos_proximo_por_dia" {
			return ErrProximo
		}
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrIndisponivel
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO avisos_cliente(cliente_id,agendamento_id,tipo,mensagem)
 SELECT cliente_id,id,$2,$3 FROM agendamentos WHERE id=$1
 ON CONFLICT (agendamento_id,tipo) DO NOTHING`, id, tipo, aviso)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// AcoesAgenda descreve os controles disponíveis para uma reserva no painel.
type AcoesAgenda struct {
	Confirmar bool     `json:"confirmar"`
	Cancelar  bool     `json:"cancelar"`
	Etapas    []string `json:"etapas"`
}

func AcoesDisponiveis(status, etapa string) AcoesAgenda {
	if status == "cancelado" || etapa == "concluida" {
		return AcoesAgenda{Etapas: []string{}}
	}
	if status == "pendente" {
		return AcoesAgenda{Confirmar: true, Cancelar: true, Etapas: []string{}}
	}
	return AcoesAgenda{
		Cancelar: true,
		Etapas:   []string{"aguardando", "proximo", "em_andamento", "concluida"},
	}
}

func SituacaoCliente(status, etapa string) string {
	if status == "cancelado" {
		return "Cancelado"
	}
	switch etapa {
	case "concluida":
		return "Concluída"
	case "proximo":
		return "Próximo na fila"
	case "em_andamento":
		return "Em lavagem"
	}
	if status == "pendente" {
		return "Aguardando confirmação"
	}
	return "Agendado"
}
