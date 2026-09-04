#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
admin_ui_directory="${project_root}/admin-ui"
dist_directory="${admin_ui_directory}/dist"
remote_target="sergio@192.168.18.10"
remote_directory="/srv/sites/compasso-admin-ui"
public_api_base_url="${COMPASSO_API_BASE_URL:-https://apicompasso.smresume.com}"

fail() {
  echo "erro: $*" >&2
  exit 1
}

for command_name in npm scp ssh; do
  command -v "${command_name}" >/dev/null 2>&1 || fail "comando obrigatório não encontrado: ${command_name}"
done

[[ -f "${admin_ui_directory}/package.json" ]] || fail "admin-ui/package.json não encontrado"
[[ "${public_api_base_url}" =~ ^https://[A-Za-z0-9.-]+(:[0-9]+)?$ ]] \
  || fail "COMPASSO_API_BASE_URL deve ser uma origem HTTPS sem caminho, consulta ou fragmento"

echo "Gerando o build da interface administrativa..."
(
  cd "${admin_ui_directory}"
  npm run build
)

[[ -f "${dist_directory}/index.html" ]] || fail "o build não gerou dist/index.html"
[[ -f "${dist_directory}/runtime-config.js" ]] || fail "o build não gerou dist/runtime-config.js"

sed -i \
  "s#apiBaseUrl:.*#apiBaseUrl: \"${public_api_base_url}\",#" \
  "${dist_directory}/runtime-config.js"

echo "Validando o destino ${remote_target}:${remote_directory}..."
ssh -o BatchMode=yes "${remote_target}" \
  "test -d '${remote_directory}' && test -w '${remote_directory}'" \
  || fail "diretório remoto ausente ou sem permissão de escrita"

echo "Enviando o conteúdo de admin-ui/dist..."
scp -o BatchMode=yes -r "${dist_directory}/." "${remote_target}:${remote_directory}/"

echo "Interface administrativa publicada em ${remote_target}:${remote_directory}."
