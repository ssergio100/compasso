#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-0.1.0-pilot4}"
if [[ ! "${version}" =~ ^[0-9A-Za-z][0-9A-Za-z._-]*$ ]]; then
  echo "erro: versão inválida: ${version}" >&2
  exit 1
fi

distribution_directory="${project_root}/dist"
temporary_directory="$(mktemp -d)"
archive_name="compasso-server-${version}"
package_root="${temporary_directory}/compasso"
archive_path="${distribution_directory}/${archive_name}.tar.gz"
trap 'rm -rf "${temporary_directory}"' EXIT

mkdir -p "${package_root}/scripts" "${package_root}/docs" "${distribution_directory}"
cp "${project_root}/go.mod" "${project_root}/go.sum" "${project_root}/compose.yaml" \
  "${project_root}/.dockerignore" "${project_root}/.env.server.example" "${package_root}/"
mkdir -p "${package_root}/agent"
cp -a "${project_root}/agent/localauth" "${project_root}/agent/policy" \
  "${project_root}/agent/storage" "${package_root}/agent/"
cp -a "${project_root}/protocol" "${package_root}/"
tar --create --file - --directory "${project_root}" --exclude='server/config.toml' server \
  | tar --extract --file - --directory "${package_root}"
cp -a "${project_root}/admin-ui" "${package_root}/"

server_scripts=(
  install-server.sh backup-server.sh restore-server-backup.sh update-server.sh
  status-server.sh
)
for script_name in "${server_scripts[@]}"; do
  install -m 0755 "${project_root}/scripts/${script_name}" "${package_root}/scripts/${script_name}"
done
cp "${project_root}/docs/server-installation.md" "${package_root}/README.md"
cp "${project_root}/docs/server-compose-plan.md" "${package_root}/docs/"

sed -i "s/^COMPASSO_VERSION=.*/COMPASSO_VERSION=${version}/" "${package_root}/.env.server.example"
find "${package_root}" -type f -not -name SHA256SUMS -print0 \
  | sort -z \
  | xargs -0 sha256sum \
  | sed "s#${package_root}/##" > "${package_root}/SHA256SUMS"

tar --create --gzip --file "${archive_path}" --directory "${temporary_directory}" compasso
(
  cd "${distribution_directory}"
  sha256sum "${archive_name}.tar.gz" > "${archive_name}.tar.gz.sha256"
)
echo "pacote criado: ${archive_path}"
