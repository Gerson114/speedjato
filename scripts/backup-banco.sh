#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
umask 077
mkdir -p backups
arquivo="backups/lava-$(date -u +%Y%m%dT%H%M%SZ)-$$.dump"
trap 'rm -f "$arquivo.tmp"' EXIT
docker compose exec -T db sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' > "$arquivo.tmp"
test -s "$arquivo.tmp"
mv "$arquivo.tmp" "$arquivo"
printf '%s\n' "Backup criado em $arquivo"
