#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
admin_ui_directory="${project_root}/admin-ui"
dist_directory="${admin_ui_directory}/dist"
deployment_config="${COMPASSO_ADMIN_UI_DEPLOY_CONFIG:-${project_root}/.private/deploy/admin-ui.env}"
remote_target="${COMPASSO_ADMIN_UI_DEPLOY_TARGET:-}"
remote_directory="${COMPASSO_ADMIN_UI_DEPLOY_DIRECTORY:-/srv/sites/compasso-admin-ui}"
public_api_base_url="${COMPASSO_API_BASE_URL:-}"

if [[ -f "${deployment_config}" ]]; then
  while IFS='=' read -r key value; do
    case "${key}" in
      COMPASSO_ADMIN_UI_DEPLOY_TARGET)
        [[ -n "${remote_target}" ]] || remote_target="${value}"
        ;;
      COMPASSO_ADMIN_UI_DEPLOY_DIRECTORY)
        remote_directory="${value:-${remote_directory}}"
        ;;
      COMPASSO_API_BASE_URL)
        [[ -n "${public_api_base_url}" ]] || public_api_base_url="${value}"
        ;;
    esac
  done < "${deployment_config}"
fi

show_usage() {
  cat <<'EOF'
Uso: ./scripts/publish-admin-ui.sh [opções]

Compila o painel e copia o build para um host já preparado.

Opções:
  --target USUARIO@HOST  destino aceito por ssh/scp
  --directory CAMINHO   diretório remoto (padrão: /srv/sites/compasso-admin-ui)
  --api-base-url URL     origem HTTPS pública da API, sem caminho
  -h, --help             mostra esta ajuda

Variáveis equivalentes: COMPASSO_ADMIN_UI_DEPLOY_TARGET,
COMPASSO_ADMIN_UI_DEPLOY_DIRECTORY e COMPASSO_API_BASE_URL.
Por padrão, o script também lê .private/deploy/admin-ui.env, se existir.
EOF
}

fail() {
  echo "erro: $*" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target) [[ $# -ge 2 ]] || fail "--target exige um valor"; remote_target="$2"; shift 2 ;;
    --directory) [[ $# -ge 2 ]] || fail "--directory exige um valor"; remote_directory="$2"; shift 2 ;;
    --api-base-url) [[ $# -ge 2 ]] || fail "--api-base-url exige um valor"; public_api_base_url="$2"; shift 2 ;;
    -h|--help) show_usage; exit 0 ;;
    *) fail "opção desconhecida: $1" ;;
  esac
done

[[ -n "${remote_target}" ]] || fail "informe --target ou COMPASSO_ADMIN_UI_DEPLOY_TARGET"
[[ "${remote_target}" =~ ^([A-Za-z0-9._-]+@)?[A-Za-z0-9._-]+$ ]] \
  || fail "destino SSH inválido: ${remote_target}"
[[ "${remote_directory}" =~ ^/[A-Za-z0-9._/-]+$ && "${remote_directory}" != *".."* ]] \
  || fail "diretório remoto deve ser um caminho absoluto simples"
[[ -n "${public_api_base_url}" ]] || fail "informe --api-base-url ou COMPASSO_API_BASE_URL"

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
