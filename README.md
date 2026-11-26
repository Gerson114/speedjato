# Backend Lava

Base em Go com Gin e PostgreSQL. O servidor abre um pool de conexões e inicia a API na porta 8080. Em desenvolvimento, aplica migrações pendentes. Em produção, exige que as migrações tenham sido executadas antes de iniciar.

## Executar com Docker

```sh
cp .env.example .env
sed -i "s/^JWT_SECRET=$/JWT_SECRET=$(openssl rand -hex 32)/" .env
docker compose up --build
```

A API fica em `http://localhost:8081`. A variável `PORT` altera a porta publicada no host. O PostgreSQL fica na rede interna do Compose e persiste os dados no volume `pgdata`.

## Executar com Go

Use Go 1.26 e um PostgreSQL acessível na sua máquina. Prepare o `.env` e a chave JWT como acima, descomente `DATABASE_URL` no `.env` e ajuste a conexão para esse banco.

```sh
go run ./cmd
```

## Cadastro e login

Envie JSON com `Content-Type: application/json`. Se o gateway estiver habilitado, envie também seu `X-Gateway-Token`.

| Método e rota | Campos | Resposta |
| --- | --- | --- |
| `POST /auth/cadastrar` | `username`, `email`, `password` | `201` cria uma conta de cliente com acesso imediato ao login. E-mail já cadastrado retorna `409`. |
| `POST /auth/login` | `email`, `password`, `rememberMe` (opcional) | `200` define o cookie HttpOnly e retorna `userId`, `role` e `expiresAt`. Credenciais incorretas retornam `401` e contas bloqueadas retornam `403`. |

O cadastro exige senha de pelo menos 15 caracteres e até 72 bytes. Se uma conta criada na versão anterior ainda aguarda liberação, você pode ativá-la no PostgreSQL pelo e-mail cadastrado.

```sql
UPDATE usuarios SET liberado = true WHERE email = 'pessoa@example.com';
```

A sessão dura 24 horas ou 30 dias com `rememberMe: true`. O cookie se chama `token`. O login com `?painel=admin` exige uma conta com papel `admin` e usa o cookie `admin_token`.

## Agenda do lava-jato

O backend atende um estabelecimento, no fuso `America/Fortaleza`. A agenda começa fechada. O dono cadastra os serviços e define quantos carros podem ser atendidos ao mesmo tempo, os dias e períodos de funcionamento e o intervalo entre as opções de início.

### Painel do administrador

Para habilitar a primeira conta do proprietário, cadastre-a e atribua o papel no PostgreSQL usando o e-mail verificado do dono.

```sql
UPDATE usuarios SET role = 'admin', liberado = true WHERE email = 'dono@example.com';
```

O painel administrativo faz login em `POST /auth/login?painel=admin` e usa o cookie `admin_token`.

| Método e rota | Ação |
| --- | --- |
| `GET /admin/agenda/configuracao` | Ler a configuração semanal. |
| `PUT /admin/agenda/configuracao` | Substituir a capacidade, o intervalo e os períodos da semana. |
| `GET /admin/servicos` | Listar os serviços. |
| `POST /admin/servicos` | Cadastrar um serviço com `{"nome":"Lavagem simples","duracaoMinutos":45}`. |
| `PUT /admin/servicos/:id` | Atualizar o nome e a duração do serviço com os mesmos campos do cadastro. |
| `GET /admin/agendamentos?data=2030-01-07` | Ver todas as reservas do dia, com cliente, serviço, início e fim. |

Exemplo de configuração para duas vagas simultâneas, com inícios a cada 15 minutos e pausa para almoço na segunda-feira.

```json
{
  "capacidade": 2,
  "intervaloMinutos": 15,
  "horarios": [
    {"diaSemana": 1, "abre": "08:00", "fecha": "12:00"},
    {"diaSemana": 1, "abre": "13:00", "fecha": "18:00"}
  ]
}
```

Os dias vão de 1 (segunda) a 7 (domingo). Dias omitidos ficam fechados. Alterações na configuração ou na duração de um serviço valem para novas reservas. As já confirmadas conservam o período reservado e continuam contando na ocupação.

### Painel do cliente

O cliente faz login em `POST /auth/login` e usa o cookie `token`. Em desenvolvimento, as origens padrão dos painéis são `http://localhost:3000` (cliente) e `http://localhost:3100` (administrador). Para enviar cookies em chamadas de outra origem, use `credentials: "include"` no frontend.

| Método e rota | Ação |
| --- | --- |
| `GET /usuario/servicos` | Escolher um serviço pelo nome e duração. |
| `GET /usuario/horarios?servicoId=1&data=2030-01-07` | Ver horários disponíveis em `HH:MM` para o serviço inteiro. |
| `POST /usuario/agendamentos` | Reservar com `{"servicoId":1,"data":"2030-01-07","inicio":"08:00"}`. |
| `GET /usuario/agendamentos` | Ver as reservas da própria conta. |

A confirmação retorna `201` com o número da reserva e o horário de término. Se a vaga se esgotar entre a consulta e a confirmação, a API retorna `409`. Reenviar o mesmo pedido com a mesma conta retorna a reserva existente (`200`).

## Adicionar funcionalidades

Registre seus endpoints em `SetupRoutes`, em `internal/router/router.go`. A função recebe o contexto da aplicação, o roteador Gin e o pool `*sql.DB`. A conexão com o banco fica em `external/db`. Para rodar os testes, use `go test ./...`.

Os middlewares em `internal/middleware` aplicam CORS, cabeçalhos de segurança, limites por IP, concorrência, tamanho de corpo e prazo da requisição. Em desenvolvimento, `GATEWAY_TOKEN` é opcional. Com o gateway ativo, o proxy envia `X-Gateway-Token` e pode identificar o IP do cliente com `X-Client-IP`.

Com `APP_ENV=production`, configure `GIN_MODE=release`, origens HTTPS em `CORS_ORIGINS`, um `GATEWAY_TOKEN` de pelo menos 32 bytes e `GATEWAY_CLIENT_IP_REQUIRED=true`. Use também `COOKIE_SECURE=true` e uma `JWT_SECRET` própria, diferente do token do gateway. Nesse modo, forneça as configurações pelo ambiente do processo.

## Atualização para produção

As migrações estão em `external/db/migrations`. Cada arquivo aplicado fica registrado com checksum em `schema_migrations`. Arquivos já aplicados devem ser preservados; novas mudanças usam um novo arquivo numerado. A execução usa transação e lock para impedir aplicação concorrente. A primeira migração reconhece as tabelas da versão anterior.

Com as variáveis de produção configuradas e o banco acessível, execute na implantação:

```sh
export COMPOSE_FILE=docker-compose.yml:docker-compose.prod.yml
docker compose config --quiet
docker compose build app
docker compose up -d db
# Em uma instalação existente, suspenda gravações durante a atualização.
docker compose stop app
# Faça o backup antes da atualização.
./scripts/backup-banco.sh
docker compose run --rm --entrypoint /migrate app
docker compose up -d --wait app
```

Para execução sem Compose, use `go run ./cmd/migrate` antes de iniciar o servidor com `APP_ENV=production`. Em produção o servidor verifica as migrações sem alterar o schema. Em desenvolvimento elas são aplicadas automaticamente.

Nos dois servidores Next.js, configure `BACKEND_URL`, `APP_ORIGIN` com o domínio HTTPS correspondente, `GATEWAY_TOKEN` igual ao backend e `TRUSTED_CLIENT_IP_HEADER=X-Real-IP`. O proxy reverso deve **sobrescrever** esse cabeçalho com o IP da conexão. Para Nginx diretamente exposto aos clientes, o bloco que encaminha ao Next.js precisa de:

```nginx
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_read_timeout 30s;
```

Os processos Next.js devem ser acessíveis somente pelo proxy confiável. Se houver CDN ou outro proxy antes do Nginx, configure os endereços confiáveis nesse proxy antes de usar o IP recebido. Configure os domínios reais e TLS no proxy, e use `npm run build` seguido de `npm run start` nos painéis. O script `dev-rede.mjs` é destinado aos testes locais.

A reserva cancelada é preservada e um reagendamento recebe novo ID. Repetições da reserva ativa continuam retornando o mesmo registro. O PostgreSQL mantém uma versão da agenda dentro da transação de cada alteração. Cada painel administrativo que espera novidades consulta essa versão a cada 2 segundos; a carga dessas consultas aumenta com o número de painéis abertos.

### Capacidade do banco

`DB_MAX_OPEN_CONNS` limita as conexões do pool por instância da API (padrão 50, máximo 50). Ao aumentar o número de réplicas, divida o orçamento disponível entre elas e reserve conexões do PostgreSQL para migrações, backup e administração. Por exemplo, três réplicas com limite 20 podem usar até 60 conexões, além das demais ferramentas. O valor adequado depende do servidor e deve ser definido com teste de carga e monitoramento de conexões, tempo das consultas e CPU.

A migração 008 cria índices para histórico e consultas por cliente. Como as migrações rodam em transação, a criação dos índices pode bloquear gravações enquanto é executada; em um banco com muitos agendamentos, planeje uma janela de manutenção e faça backup antes da atualização.

## Testes de integração e backup

```sh
./scripts/testar-producao.sh
node scripts/proxy-ip.test.cjs
```

O primeiro comando cria um PostgreSQL descartável, aplica migrações, testa reagendamento, pedidos concorrentes, capacidade, permissões de andamento, sincronização entre conexões e rollback de uma falha de gravação. Também cria um backup e o restaura em outro banco descartável, verificando dados e migrações. Requer Docker e acesso às imagens e dependências. Não conecta ao banco configurado no `.env` da aplicação.

Para executar a integração em um PostgreSQL de testes já disponível, defina `LAVA_TEST_DATABASE_URL` como URL `postgres://...` e rode `go test -race -count=1 ./internal/agenda`. O teste cria e remove um schema próprio. Sem essa variável, esse teste é marcado como ignorado.

`./scripts/backup-banco.sh` gera um arquivo `pg_dump` em `backups/`, com permissões restritas. Programe sua execução no servidor e copie os arquivos para armazenamento fora da máquina, com retenção definida. A pasta local e o volume do PostgreSQL precisam de proteção contra perda do host. Valide a restauração de backups reais em um banco separado antes de uma atualização importante.

## Template de produção com os dois painéis

No Dokploy, selecione Docker Compose, use a raiz do repositório como diretório e `compose.template.yml` como arquivo Compose. Os builds usam a raiz como contexto e escolhem o painel por `APP_DIR`; não configure outro Dockerfile no painel da plataforma.

Configure `ADMIN_ORIGIN` e `CLIENTE_ORIGIN` com as duas origens HTTPS públicas, sem barra final. Defina `POSTGRES_PASSWORD`, `ADMIN_EMAIL` e `ADMIN_PASSWORD`. Gere valores independentes para `JWT_SECRET` e `GATEWAY_TOKEN` executando `openssl rand -hex 32` uma vez para cada variável.

No Dokploy, configure um domínio HTTPS para o serviço `admin` na porta interna 3000 e outro para `cliente`, também na porta 3000. A API e o banco não precisam de domínio público. Configure o proxy para encaminhar o IP real pelo cabeçalho `X-Real-IP`, e mantenha `TRUSTED_CLIENT_IP_HEADER=X-Real-IP` nos painéis.

```sh
docker compose -f compose.template.yml config --quiet
docker compose -f compose.template.yml up -d --build --wait
```

A API executa as migrações antes de iniciar e os painéis aguardam a prontidão da API. O volume `pgdata` mantém os dados entre implantações do mesmo projeto. Em atualização de uma instalação existente, confira o projeto e o volume de destino e faça o backup antes de subir a nova versão.

O chat está desativado nos painéis e na API. O histórico permanece no banco.

Os painéis apontam para `http://app:8080` pela rede interna. O Dockerfile de frontend recebe `APP_DIR` no build para compilar cada painel com suas próprias dependências.

### Entrar com o Google na área do cliente

Crie um cliente OAuth do tipo aplicativo Web no Google Cloud e cadastre a origem exata do painel do cliente nas origens JavaScript autorizadas (por exemplo, `http://localhost:3000` no desenvolvimento). Configure a tela de consentimento e, se o aplicativo estiver em teste, adicione os usuários de teste.

Defina `GOOGLE_CLIENT_ID` na API Go e no servidor Next.js de `frontc/my-app`. Na API, defina também `GOOGLE_CLIENT_SECRET` e `CLIENTE_ORIGIN` (a origem exata do painel, sem barra final). Com o Compose de implantação, preencha as duas variáveis no ambiente do projeto e recrie os serviços app e cliente.

O primeiro acesso cria uma conta de cliente. Se o e-mail já pertencer a um cadastro com senha, a tela orienta o cliente a usar a senha desse cadastro.

### Migração de e-mails de clientes existentes

A migração 010 exige um único cadastro por e-mail, independentemente de maiúsculas e minúsculas. Para localizar conflitos antes da implantação, consulte os IDs no PostgreSQL:

```sql
SELECT lower(email) AS email, array_agg(id ORDER BY id) AS contas
FROM usuarios GROUP BY lower(email) HAVING count(*) > 1;
```

Se houver conflitos, confirme os titulares e atribua endereços distintos e verificados às respectivas contas antes de executar a migração. Os IDs preservam as associações de veículos e agendamentos.
