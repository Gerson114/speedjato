CREATE TABLE servico_produtos (
 servico_id INTEGER NOT NULL REFERENCES agenda_servicos(id) ON DELETE CASCADE,
 produto_id BIGINT NOT NULL REFERENCES estoque_produtos(id),
 PRIMARY KEY(servico_id,produto_id)
);
