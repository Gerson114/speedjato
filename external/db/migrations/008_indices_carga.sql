-- Atende histórico e busca de lembretes sem percorrer os agendamentos de outros clientes.
CREATE INDEX agendamentos_cliente_dia_inicio ON agendamentos (cliente_id, dia DESC, inicio DESC, id DESC);

-- O painel histórico lista apenas reservas canceladas ou concluídas.
CREATE INDEX agendamentos_historico_dia ON agendamentos (dia DESC, inicio DESC, id DESC)
 WHERE status = 'cancelado' OR lavagem_status = 'concluida';
