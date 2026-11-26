CREATE TABLE limite_requisicoes (
 chave TEXT PRIMARY KEY,
 tokens DOUBLE PRECISION NOT NULL,
 atualizado TIMESTAMPTZ NOT NULL
);
CREATE INDEX limite_requisicoes_atualizado ON limite_requisicoes(atualizado);
