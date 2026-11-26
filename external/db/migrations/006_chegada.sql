ALTER TABLE agenda_config ADD COLUMN tolerancia_minutos INTEGER NOT NULL DEFAULT 15 CHECK (tolerancia_minutos BETWEEN 0 AND 1440);
ALTER TABLE agendamentos ADD COLUMN chegou BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE agendamentos SET chegou = TRUE WHERE lavagem_status IN ('proximo', 'em_andamento', 'concluida');
