CREATE TABLE chat_mensagens (
 id BIGSERIAL PRIMARY KEY,
 cliente_id INTEGER NOT NULL REFERENCES usuarios(id),
 autor_id INTEGER NOT NULL REFERENCES usuarios(id),
 autor_papel TEXT NOT NULL CHECK (autor_papel IN ('admin','user')),
 texto TEXT NOT NULL CHECK (length(texto) BETWEEN 1 AND 2000),
 criado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX chat_mensagens_cliente_id ON chat_mensagens(cliente_id,id DESC);
