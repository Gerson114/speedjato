-- Cancelados são registros históricos; um reagendamento recebe outro ID.
ALTER TABLE agendamentos DROP CONSTRAINT IF EXISTS agendamentos_cliente_id_servico_id_dia_inicio_key;
CREATE UNIQUE INDEX agendamentos_reserva_ativa ON agendamentos(cliente_id,servico_id,veiculo_id,dia,inicio)
 WHERE status <> 'cancelado';

-- A versão muda na mesma transação da reserva e é compartilhada entre instâncias.
CREATE TABLE agenda_versao (
 id INTEGER PRIMARY KEY CHECK (id=1),
 versao BIGINT NOT NULL DEFAULT 0
);
INSERT INTO agenda_versao(id) VALUES(1);
CREATE FUNCTION atualizar_versao_agenda() RETURNS TRIGGER AS $$
BEGIN
 UPDATE agenda_versao SET versao=versao+1 WHERE id=1;
 RETURN NULL;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER agendamentos_atualizar_versao
 AFTER INSERT OR UPDATE OR DELETE ON agendamentos
 FOR EACH STATEMENT EXECUTE FUNCTION atualizar_versao_agenda();
