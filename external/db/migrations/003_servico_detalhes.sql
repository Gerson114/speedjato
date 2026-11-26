ALTER TABLE agenda_servicos ADD COLUMN descricao TEXT NOT NULL DEFAULT '';
ALTER TABLE agenda_servicos ADD COLUMN itens TEXT[] NOT NULL DEFAULT '{}';
