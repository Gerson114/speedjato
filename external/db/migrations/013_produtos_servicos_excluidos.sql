DELETE FROM servico_produtos sp USING agenda_servicos s
WHERE s.id = sp.servico_id AND s.excluido;
