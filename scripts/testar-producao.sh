#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
raiz="$PWD"
nome="lava-teste-$(date +%s)-$$"
limpar() {
  docker rm -f "$nome" >/dev/null 2>&1 || true
  docker network rm "$nome" >/dev/null 2>&1 || true
}
trap limpar EXIT
# Ambiente descartável sem portas publicadas e sem acesso ao banco da aplicação.
docker network create "$nome" >/dev/null
docker run -d --name "$nome" --network "$nome" --network-alias postgres \
  -e POSTGRES_PASSWORD=teste-descartavel -e POSTGRES_DB=lava_teste postgres:16-alpine >/dev/null
for tentativa in $(seq 1 30); do
  if docker exec "$nome" pg_isready -U postgres -d lava_teste >/dev/null 2>&1; then break; fi
  sleep 1
done
docker exec "$nome" pg_isready -U postgres -d lava_teste
dsn='postgres://postgres:teste-descartavel@postgres:5432/lava_teste?sslmode=disable'
docker run --rm --network "$nome" -v "$raiz:/src:ro" -w /src \
  -e LAVA_TEST_DATABASE_URL="$dsn" -e DATABASE_URL="$dsn" \
  -e GOCACHE=/tmp/go-cache golang:1.26.10 \
  sh -c 'go run ./cmd/migrate && go test -race -count=1 ./internal/... ./external/... ./cmd/...'
# Exercita backup e restauração em outro banco do ambiente descartável.
docker exec "$nome" psql -U postgres -d lava_teste -v ON_ERROR_STOP=1 \
  -c "INSERT INTO usuarios(username,email,password) VALUES('Restauração','restore@example.invalid','!')"
docker exec "$nome" createdb -U postgres lava_restaurado
docker exec "$nome" pg_dump -U postgres -d lava_teste -Fc |
  docker exec -i "$nome" pg_restore -U postgres -d lava_restaurado --exit-on-error
resultado=$(docker exec "$nome" psql -U postgres -d lava_restaurado -At -v ON_ERROR_STOP=1 \
  -c "SELECT count(*) FROM usuarios WHERE email='restore@example.invalid'")
[[ "$resultado" == "1" ]]
migracoes=$(docker exec "$nome" psql -U postgres -d lava_restaurado -At -v ON_ERROR_STOP=1 \
  -c 'SELECT count(*) FROM schema_migrations')
esperadas=$(find external/db/migrations -maxdepth 1 -name '*.sql' -type f | wc -l)
[[ "$migracoes" -eq "$esperadas" ]]
printf '%s\n' 'Testes de integração e restauração concluídos.'
