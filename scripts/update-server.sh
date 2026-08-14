#!/usr/bin/env bash
set -euo pipefail

project_directory="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ "${EUID}" -ne 0 ]]; then
  echo "erro: execute com sudo" >&2
  exit 1
fi

"${project_directory}/scripts/backup-server.sh"
docker compose --project-directory "${project_directory}" config --quiet
docker compose --project-directory "${project_directory}" build
docker compose --project-directory "${project_directory}" up --detach --remove-orphans
"${project_directory}/scripts/status-server.sh"
