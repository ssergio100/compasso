#!/usr/bin/env bash
set -euo pipefail

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "erro: sqlite3 não encontrado no PATH" >&2
  exit 1
fi

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf "${test_dir}"' EXIT

test_component() {
  component="$1"
  migrations_dir="${project_root}/${component}/storage/migrations"
  database="${test_dir}/${component}.db"

  migration_count=0
  for migration in "${migrations_dir}"/*.sql; do
    sqlite3 -bail "${database}" < "${migration}"
    migration_count=$((migration_count + 1))
  done

  recorded_count="$(sqlite3 "${database}" 'SELECT COUNT(*) FROM schema_migrations;')"
  integrity="$(sqlite3 "${database}" 'PRAGMA integrity_check;')"
  foreign_keys="$(sqlite3 "${database}" 'PRAGMA foreign_key_check;')"

  if [ "${recorded_count}" -ne "${migration_count}" ]; then
    echo "erro: ${component}: ${recorded_count} versões registradas para ${migration_count} migrações" >&2
    exit 1
  fi
  if [ "${integrity}" != "ok" ] || [ -n "${foreign_keys}" ]; then
    echo "erro: ${component}: falha de integridade no banco criado" >&2
    exit 1
  fi

  echo "${component}: ${migration_count} migração(ões) aplicada(s) com sucesso"
}

test_component agent
test_component server
