DO $$
DECLARE
    exemplo record;
    cliente integer;
    carro bigint;
    lavagem integer;
    hoje date := (now() AT TIME ZONE 'America/Fortaleza')::date;
BEGIN
    PERFORM pg_advisory_xact_lock(936174);
    FOR exemplo IN SELECT * FROM (VALUES
        ('Ana','Onix','TST1A01','Lavagem simples',540,30,4000),
        ('Bruno','Corolla','TST2B02','Lavagem completa',660,60,8000),
        ('Carla','HB20','TST3C03','Lavagem externa',840,30,3500)
    ) AS itens(nome,modelo,placa,servico,inicio,duracao,preco)
    LOOP
        INSERT INTO usuarios(username,email,password)
        VALUES (exemplo.nome || ' (teste)', exemplo.placa || '@agenda.invalid','!') ON CONFLICT (email) DO NOTHING;
        SELECT id INTO STRICT cliente FROM usuarios WHERE email=exemplo.placa || '@agenda.invalid' AND password='!';
        INSERT INTO veiculos(cliente_id,apelido,modelo,placa)
        VALUES(cliente,exemplo.modelo,exemplo.modelo,exemplo.placa) ON CONFLICT DO NOTHING;
        SELECT id INTO STRICT carro FROM veiculos WHERE cliente_id=cliente AND placa=exemplo.placa;
        SELECT min(id) INTO lavagem FROM agenda_servicos WHERE nome=exemplo.servico || ' (teste)';
        IF lavagem IS NULL THEN
            INSERT INTO agenda_servicos(nome,duracao,preco_centavos)
            VALUES(exemplo.servico || ' (teste)',exemplo.duracao,exemplo.preco) RETURNING id INTO lavagem;
        END IF;
        INSERT INTO agendamentos(cliente_id,servico_id,veiculo_id,veiculo,preco_centavos,nome_servico,dia,inicio,fim)
        VALUES(cliente,lavagem,carro,exemplo.modelo || ' · ' || exemplo.placa,exemplo.preco,exemplo.servico || ' (teste)',hoje,exemplo.inicio,exemplo.inicio+exemplo.duracao)
        ON CONFLICT DO NOTHING;
    END LOOP;
END $$;
