#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
server_url="${TEMPO_TEST_SERVER_URL:-http://127.0.0.1:8081}"
controlled_user="$(id -un)"

read -r -p "device_id exibido no painel: " device_id
read -r -s -p "device_token exibido no painel: " device_token
printf '\n'

if [[ ! "${device_id}" =~ ^[A-Za-z0-9-]+$ ]]; then
  echo "erro: device_id inválido" >&2
  exit 1
fi
if [[ ! "${device_token}" =~ ^[A-Za-z0-9_-]{40,}$ ]]; then
  echo "erro: device_token inválido" >&2
  exit 1
fi

umask 077
mkdir -p "${project_root}/var/agent"
{
  printf 'database_path = "./var/agent/phase8-test.db"\n'
  printf 'controlled_user = "%s"\n' "${controlled_user}"
  printf 'tick_interval = "1s"\n'
  printf 'checkpoint_interval = "5s"\n'
  printf 'loginctl_path = "/usr/bin/loginctl"\n'
  printf 'server_url = "%s"\n' "${server_url}"
  printf 'device_id = "%s"\n' "${device_id}"
  printf 'device_token = "%s"\n' "${device_token}"
  printf 'heartbeat_interval = "10s"\n'
  printf 'http_timeout = "8s"\n'
} > "${project_root}/agent/config.toml"

echo "Configuração local criada em agent/config.toml para ${controlled_user}."
