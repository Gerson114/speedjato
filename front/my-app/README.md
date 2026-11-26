# Painel administrativo Lava

Execute `npm run dev -- --port 3100` nesta pasta e abra `http://localhost:3100/admin`. Inicie também a API Go com PostgreSQL conforme o README da raiz. Faça login com uma conta com papel `admin`.

A tela Agenda consulta as reservas por data. Em Serviços, o administrador cadastra e edita a duração das lavagens. Em Funcionamento, configura as vagas simultâneas, o intervalo entre inícios e os períodos de cada dia da semana.

O Next encaminha chamadas para `BACKEND_URL` (padrão `http://127.0.0.1:8080`) e usa apenas o cookie administrativo HttpOnly. O botão Sair remove esse cookie do navegador. Se o gateway do backend estiver ativo, configure o mesmo `GATEWAY_TOKEN` no ambiente do servidor Next. O endereço público do painel deve constar em `CORS_ORIGINS` na API. Atrás de um proxy, configure `APP_ORIGIN` com a origem pública do painel. Se a API exigir o IP do cliente, defina `TRUSTED_CLIENT_IP_HEADER` com o nome de um cabeçalho que o proxy de entrada sempre sobrescreve com o IP verificado.

A fotografia de acesso está em `public/lava-garagem.webp`. Foi criada com a ferramenta integrada ImageGen a partir deste prompt:

> Portrait editorial automotive photograph for a Brazilian car wash administrator sign-in page. Tight front three-quarter crop of a charcoal graphite sports sedan in a real dark detailing garage, cool silver LED strips on the sculpted hood, water beads and fine hose mist in the warm overhead light, unbranded. The car fills the lower two thirds of the photograph and the headlight is sharp and recognizable. Natural grain, deep shadows with visible texture, premium analog motorsport magazine aesthetic. No lettering, no logos, no graphics, no text.
