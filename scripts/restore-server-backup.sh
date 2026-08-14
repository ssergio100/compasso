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
requested_backup="${1:-}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "erro: execute com sudo" >&2
  exit 1
fi
if [[ -z "${requested_backup}" || ! -f "${requested_backup}" ]]; then
  echo "uso: sudo $0 /srv/docker/backups/compasso/compasso-server-DATA.tar.gz" >&2
  exit 1
fi
case "$(realpath "${requested_backup}")" in
  "$(realpath "${backup_directory}")"/*) ;;
  *) echo "erro: o backup deve estar dentro de ${backup_directory}" >&2; exit 1 ;;
esac
if ! tar --list --gzip --file "${requested_backup}" | grep -q '^server/'; then
  echo "erro: arquivo não contém o diretório server esperado" >&2
  exit 1
fi

read -r -p "Digite RESTAURAR para substituir os dados atuais: " confirmation
if [[ "${confirmation}" != "RESTAURAR" ]]; then
  echo "restauração cancelada"
  exit 0
fi

docker compose --project-directory "${project_directory}" stop compasso-api
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
preserved_directory="${backup_directory}/pre-restore-server-${timestamp}"
if [[ -d "${data_directory}/server" ]]; then
  mv "${data_directory}/server" "${preserved_directory}"
fi
mkdir -p "${data_directory}"
if ! tar --extract --gzip --file "${requested_backup}" --directory "${data_directory}"; then
  failed_restore_directory="${backup_directory}/failed-restore-server-${timestamp}"
  if [[ -d "${data_directory}/server" ]]; then mv "${data_directory}/server" "${failed_restore_directory}"; fi
  if [[ -d "${preserved_directory}" ]]; then mv "${preserved_directory}" "${data_directory}/server"; fi
  echo "erro: falha ao extrair; dados anteriores restaurados e tentativa preservada em ${failed_restore_directory}" >&2
  exit 1
fi
chown -R 10001:10001 "${data_directory}/server"
chmod 0700 "${data_directory}/server"
docker compose --project-directory "${project_directory}" start compasso-api
echo "restauração concluída; estado anterior preservado em ${preserved_directory}"
