#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
archive_path="${project_root}/dist/compasso-server-0.1.0-test.tar.gz"
trap 'rm -rf "${temporary_directory}"; rm -f "${archive_path}" "${archive_path}.sha256"' EXIT

"${project_root}/scripts/build-server-package.sh" 0.1.0-test
(
  cd "${project_root}/dist"
  sha256sum --check compasso-server-0.1.0-test.tar.gz.sha256 >/dev/null
)
tar --extract --gzip --file "${archive_path}" --directory "${temporary_directory}"
package_root="${temporary_directory}/compasso"

test -f "${package_root}/compose.yaml"
test -x "${package_root}/scripts/install-server.sh"
test ! -e "${package_root}/secrets"
test ! -e "${package_root}/.env"
test ! -e "${package_root}/server/config.toml"
test ! -e "${package_root}/agent/config.toml"
test ! -e "${package_root}/scripts/configure-server-cloudflare-ingress.sh"
if grep -Eqi 'cloudflare|cloudflared|/srv/cloudflare' "${package_root}/scripts/install-server.sh" "${package_root}/compose.yaml"; then
  echo "erro: instalador principal ainda depende de infraestrutura de exposição" >&2
  exit 1
fi
grep -Fq 'COMPASSO_API_BIND_ADDRESS' "${package_root}/compose.yaml"
grep -Fq 'COMPASSO_ADMIN_BIND_ADDRESS' "${package_root}/compose.yaml"
if grep -R -E 'compasso-teste|TEMPO_ADMIN_PASSWORD=' "${package_root}"; then
  echo "erro: possível segredo encontrado no pacote" >&2
  exit 1
fi

mkdir -p "${temporary_directory}/data/server"
COMPASSO_DATA_DIRECTORY="${temporary_directory}/data" \
  docker compose --project-directory "${package_root}" config --quiet

node --check "${package_root}/admin-ui/api-client.js"
node --check "${package_root}/admin-ui/app.js"
bash -n "${package_root}"/scripts/*.sh
(
  cd "${package_root}"
  sha256sum --check SHA256SUMS >/dev/null
)

echo "estrutura, ausência de segredos, Compose, JavaScript, shell e checksums do pacote validados"
