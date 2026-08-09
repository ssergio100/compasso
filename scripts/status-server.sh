#!/usr/bin/env bash
set -euo pipefail

project_directory="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
docker compose --project-directory "${project_directory}" ps
docker compose --project-directory "${project_directory}" exec --no-TTY compasso-api \
  curl --fail --silent --show-error "http://127.0.0.1:8080/healthz"
echo
docker compose --project-directory "${project_directory}" exec --no-TTY compasso-admin-ui \
  wget --quiet --output-document=- "http://127.0.0.1:8080/healthz"
echo
echo "API e interface responderam aos healthchecks internos."
