#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
package_version="$(sed -n 's/^Version: //p' "${project_root}/packaging/debian/control")"
package_path="${1:-${project_root}/dist/compasso-server_${package_version}_all.deb}"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

if [[ ! -f "${package_path}" ]]; then
  echo "erro: pacote ausente: ${package_path}" >&2
  exit 1
fi
if [[ "$(dpkg-deb --field "${package_path}" Package)" != "compasso-server" ]]; then
  echo "erro: nome inesperado para o pacote do servidor" >&2
  exit 1
fi
if [[ "$(dpkg-deb --field "${package_path}" Architecture)" != "all" ]]; then
  echo "erro: pacote do servidor deve ser independente de arquitetura" >&2
  exit 1
fi

dpkg-deb --extract "${package_path}" "${temporary_directory}/root"
dpkg-deb --control "${package_path}" "${temporary_directory}/control"
package_root="${temporary_directory}/root/opt/compasso-server"
packaged_container_version="$(sed -n 's/^COMPASSO_VERSION=//p' \
  "${temporary_directory}/root/etc/compasso-server/compasso.env")"
expected_container_version="${package_version//\~/-}"

sh -n "${temporary_directory}/control/postinst"
test -L "${package_root}/.env"
test "$(readlink "${package_root}/.env")" = /etc/compasso-server/compasso.env
test -f "${temporary_directory}/root/etc/compasso-server/compasso.env"
test "${packaged_container_version}" = "${expected_container_version}"
if [[ ! "${packaged_container_version}" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]]; then
  echo "erro: COMPASSO_VERSION não é uma tag Docker válida" >&2
  exit 1
fi
test -f "${package_root}/compose.yaml"
test -x "${package_root}/scripts/install-server.sh"
test ! -e "${package_root}/secrets"
test ! -e "${package_root}/server/config.toml"
test ! -e "${package_root}/agent/config.toml"

if grep -Eqi 'cloudflare|cloudflared|/srv/cloudflare' \
  "${package_root}/scripts/install-server.sh" "${package_root}/compose.yaml"; then
  echo "erro: pacote do servidor depende de infraestrutura de exposição" >&2
  exit 1
fi
if grep -R --exclude=.env -E 'compasso-teste|TEMPO_ADMIN_PASSWORD=' "${package_root}"; then
  echo "erro: possível segredo encontrado no pacote" >&2
  exit 1
fi

COMPASSO_DATA_DIRECTORY="${temporary_directory}/data" \
  docker compose --env-file \
    "${temporary_directory}/root/etc/compasso-server/compasso.env" \
    --project-directory "${package_root}" config --quiet
test ! -e "${package_root}/admin-ui"
bash -n "${package_root}"/scripts/*.sh

echo "pacote Debian do servidor validado"
