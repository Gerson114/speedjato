package agenda

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ProcessarAusencias cancela uma ausência e antecipa um cliente presente quando há capacidade.
func ProcessarAusencias(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// O mesmo bloqueio é usado na criação de reservas, preservando a capacidade.
	cfg, err := lerConfig(ctx, tx, true)
	if err != nil {
		return err
	}

	var id int64
	var dia string
	var inicio, limite int
	err = tx.QueryRowContext(ctx, `
		SELECT id, to_char(dia,'YYYY-MM-DD'), inicio,
		       EXTRACT(HOUR FROM now() AT TIME ZONE 'America/Fortaleza')::int * 60 +
		       EXTRACT(MINUTE FROM now() AT TIME ZONE 'America/Fortaleza')::int
		FROM agendamentos
		WHERE status='confirmado' AND NOT chegou AND lavagem_status='aguardando'
		  AND dia = (now() AT TIME ZONE 'America/Fortaleza')::date
		  AND dia + inicio * interval '1 minute' + $1 * interval '1 minute'
		      < (now() AT TIME ZONE 'America/Fortaleza')
		ORDER BY inicio,id LIMIT 1 FOR UPDATE`, cfg.Tolerancia).Scan(&id, &dia, &inicio, &limite)
	if errors.Is(err, sql.ErrNoRows) {
		return tx.Commit()
	}
	if err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `UPDATE agendamentos SET status='cancelado'
		WHERE id=$1 AND status='confirmado' AND NOT chegou AND lavagem_status='aguardando'`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n == 0 {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO avisos_cliente(cliente_id,agendamento_id,tipo,mensagem)
		SELECT cliente_id,id,'ausencia','Seu agendamento foi cancelado porque a chegada não foi registrada dentro do prazo.'
		FROM agendamentos WHERE id=$1 ON CONFLICT (agendamento_id,tipo) DO NOTHING`, id)
	if err != nil {
		return err
	}

	// O horário antecipado começa no minuto atual, preserva a duração e respeita a capacidade.
	var candidato int64
	err = tx.QueryRowContext(ctx, `
		SELECT a.id FROM agendamentos a
		WHERE a.dia=$1::date AND a.inicio>$2 AND a.inicio>$3
		  AND a.status='confirmado' AND a.chegou AND a.lavagem_status='aguardando'
		  AND $3 + (a.fim-a.inicio) <= 1440
		  AND NOT EXISTS (
		    SELECT 1 FROM generate_series($3, $3+(a.fim-a.inicio)-1) minuto
		    WHERE (SELECT count(*) FROM agendamentos ocupacao
		      WHERE ocupacao.dia=a.dia AND ocupacao.id<>a.id AND ocupacao.status<>'cancelado'
		        AND ocupacao.inicio<=minuto AND ocupacao.fim>minuto) >= $4
		  )
		ORDER BY a.inicio,a.id LIMIT 1 FOR UPDATE`, dia, inicio, limite, cfg.Capacidade).Scan(&candidato)
	if errors.Is(err, sql.ErrNoRows) {
		return tx.Commit()
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE agendamentos SET fim=$2+(fim-inicio), inicio=$2 WHERE id=$1`, candidato, limite)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO avisos_cliente(cliente_id,agendamento_id,tipo,mensagem)
		SELECT cliente_id,id,'horario_antecipado',$2 FROM agendamentos WHERE id=$1
		ON CONFLICT (agendamento_id,tipo) DO UPDATE SET mensagem=EXCLUDED.mensagem, criado_em=now(), lido=false`,
		candidato, fmt.Sprintf("Seu atendimento foi antecipado para %s às %02d:%02d.", dia, limite/60, limite%60))
	if err != nil {
		return err
	}
	return tx.Commit()
}
