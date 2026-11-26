-- A migração inteira roda em uma transação. Não há como inferir a
-- titularidade de veículos e reservas quando dois cadastros coincidem.
LOCK TABLE usuarios IN SHARE MODE;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM usuarios GROUP BY lower(email) HAVING count(*) > 1) THEN
        RAISE EXCEPTION 'Há contas com o mesmo e-mail ignorando maiúsculas e minúsculas.'
            USING HINT = 'Confirme a titularidade dos IDs duplicados, corrija os endereços e execute migrate novamente.';
    END IF;
END;
$$;
CREATE UNIQUE INDEX usuarios_email_minusculo_uidx ON usuarios (lower(email));
