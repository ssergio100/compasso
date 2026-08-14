#!/usr/bin/env bash
set -euo pipefail

project_directory="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
read_environment_setting() {
  local setting_name="$1"
  local fallback_value="$2"
  local configured_value=""
  if [[ -f "${project_directory}/.env" ]]; then
    configured_value="$(awk -F= -v name="${setting_name}" '$1 == name { sub(/^[^=]*=/, ""); print; exit }' "${project_directory}/.env")"
  fi
  printf '%s' "${configured_value:-${fallback_value}}"
}
data_directory="${COMPASSO_DATA_DIRECTORY:-$(read_environment_setting COMPASSO_DATA_DIRECTORY /srv/docker/volumes/compasso)}"
backup_directory="${COMPASSO_BACKUP_DIRECTORY:-$(read_environment_setting COMPASSO_BACKUP_DIRECTORY /srv/docker/backups/compasso)}"
server_data_directory="${data_directory}/server"

if [[ "${EUID}" -ne 0 ]]; then
  echo "erro: execute com sudo" >&2
  exit 1
fi
if [[ ! -d "${server_data_directory}" ]]; then
  echo "erro: dados do servidor não encontrados em ${server_data_directory}" >&2
  exit 1
fi

mkdir -p "${backup_directory}"
chmod 0700 "${backup_directory}"
was_running=false
if docker compose --project-directory "${project_directory}" ps --status running --quiet compasso-api | grep -q .; then
  was_running=true
  docker compose --project-directory "${project_directory}" stop compasso-api
fi
restart_api() {
  if [[ "${was_running}" == true ]]; then
    docker compose --project-directory "${project_directory}" start compasso-api >/dev/null
  fi
}
trap restart_api EXIT

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup_path="${backup_directory}/compasso-server-${timestamp}.tar.gz"
tar --create --gzip --file "${backup_path}" --directory "${data_directory}" server
chmod 0600 "${backup_path}"
tar --list --gzip --file "${backup_path}" >/dev/null

echo "backup concluído: ${backup_path}"
