CREATE TABLE IF NOT EXISTS usuarios (
    id         SERIAL PRIMARY KEY,
    username   TEXT NOT NULL,
    email      TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS liberado BOOLEAN NOT NULL DEFAULT false;
-- Invalida sessões quando senha, papel ou liberação mudam, inclusive via SQL.
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS sessao_versao BIGINT NOT NULL DEFAULT 1;
CREATE OR REPLACE FUNCTION invalidar_sessoes_usuario() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.password IS DISTINCT FROM OLD.password
       OR NEW.role IS DISTINCT FROM OLD.role
       OR NEW.liberado IS DISTINCT FROM OLD.liberado THEN
        NEW.sessao_versao := OLD.sessao_versao + 1;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE TRIGGER usuarios_invalidar_sessoes
BEFORE UPDATE OF password, role, liberado ON usuarios
FOR EACH ROW EXECUTE FUNCTION invalidar_sessoes_usuario();

CREATE TABLE IF NOT EXISTS agenda_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    capacidade INTEGER NOT NULL CHECK (capacidade >= 0),
    intervalo INTEGER NOT NULL CHECK (intervalo BETWEEN 1 AND 1440),
    horarios JSONB NOT NULL
);
INSERT INTO agenda_config VALUES (1, 0, 1, '[]') ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS agenda_servicos (
    id SERIAL PRIMARY KEY,
    nome TEXT NOT NULL,
    duracao INTEGER NOT NULL CHECK (duracao BETWEEN 1 AND 1440),
    preco_centavos BIGINT NOT NULL DEFAULT 0 CHECK (preco_centavos >= 0)
);

ALTER TABLE agenda_servicos ADD COLUMN IF NOT EXISTS preco_centavos BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS veiculos (
    id BIGSERIAL PRIMARY KEY,
    cliente_id INTEGER NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    apelido TEXT NOT NULL,
    modelo TEXT NOT NULL,
    placa TEXT NOT NULL,
    cor TEXT NOT NULL DEFAULT '',
    UNIQUE (cliente_id, placa)
);
CREATE INDEX IF NOT EXISTS veiculos_cliente ON veiculos (cliente_id, id);

CREATE TABLE IF NOT EXISTS agendamentos (
    id BIGSERIAL PRIMARY KEY,
    cliente_id INTEGER NOT NULL REFERENCES usuarios(id),
    servico_id INTEGER NOT NULL REFERENCES agenda_servicos(id),
    veiculo_id BIGINT NOT NULL REFERENCES veiculos(id),
    veiculo TEXT NOT NULL DEFAULT '',
    preco_centavos BIGINT NOT NULL DEFAULT 0,
    nome_servico TEXT NOT NULL,
    dia DATE NOT NULL,
    inicio INTEGER NOT NULL CHECK (inicio >= 0),
    fim INTEGER NOT NULL CHECK (fim > inicio AND fim <= 1440),
    status TEXT NOT NULL DEFAULT 'pendente' CHECK (status IN ('pendente','confirmado','cancelado')),
    lavagem_status TEXT NOT NULL DEFAULT 'aguardando' CHECK (lavagem_status IN ('aguardando','em_andamento','concluida')),
    UNIQUE (cliente_id, servico_id, dia, inicio)
);
CREATE INDEX IF NOT EXISTS agendamentos_dia_inicio ON agendamentos (dia, inicio);

ALTER TABLE agendamentos ADD COLUMN IF NOT EXISTS veiculo_id BIGINT REFERENCES veiculos(id);
ALTER TABLE agendamentos ADD COLUMN IF NOT EXISTS veiculo TEXT NOT NULL DEFAULT '';
ALTER TABLE agendamentos ADD COLUMN IF NOT EXISTS preco_centavos BIGINT NOT NULL DEFAULT 0;
ALTER TABLE agendamentos ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'confirmado';
ALTER TABLE agendamentos ALTER COLUMN status SET DEFAULT 'pendente';
ALTER TABLE agendamentos ADD COLUMN IF NOT EXISTS lavagem_status TEXT NOT NULL DEFAULT 'aguardando';
ALTER TABLE agendamentos DROP CONSTRAINT IF EXISTS agendamentos_lavagem_status_check;
ALTER TABLE agendamentos ADD CONSTRAINT agendamentos_lavagem_status_check
    CHECK (lavagem_status IN ('aguardando','proximo','em_andamento','concluida'));
CREATE UNIQUE INDEX IF NOT EXISTS agendamentos_proximo_por_dia ON agendamentos(dia)
    WHERE lavagem_status='proximo' AND status <> 'cancelado';

CREATE TABLE IF NOT EXISTS avisos_cliente (
    id BIGSERIAL PRIMARY KEY,
    cliente_id INTEGER NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    agendamento_id BIGINT REFERENCES agendamentos(id) ON DELETE CASCADE,
    tipo TEXT NOT NULL,
    mensagem TEXT NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lido BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (agendamento_id, tipo)
);
CREATE INDEX IF NOT EXISTS avisos_cliente_usuario ON avisos_cliente (cliente_id, criado_em DESC);

CREATE TABLE IF NOT EXISTS estoque_produtos (
 id BIGSERIAL PRIMARY KEY,
 nome TEXT NOT NULL,
 unidade TEXT NOT NULL,
 quantidade DOUBLE PRECISION NOT NULL CHECK (quantidade >= 0),
 consumo_diario DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (consumo_diario >= 0),
 criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
