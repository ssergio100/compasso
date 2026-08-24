#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version_file="${project_root}/packaging/debian/control"
current_version="$(sed -n 's/^Version: //p' "${version_file}")"
package_architecture="$(sed -n 's/^Architecture: //p' "${version_file}")"

if [[ ! "${current_version}" =~ ^(.+~pilot)([0-9]+)$ ]]; then
  echo "erro: versão atual não termina em ~pilotN: ${current_version}" >&2
  exit 1
fi

version_prefix="${BASH_REMATCH[1]}"
pilot_number="${BASH_REMATCH[2]}"
package_version="${version_prefix}$((10#${pilot_number} + 1))"

if ! dpkg --validate-version "${package_version}"; then
  echo "erro: próxima versão Debian inválida: ${package_version}" >&2
  exit 1
fi

sed -i "s/^Version: .*/Version: ${package_version}/" "${version_file}"
echo "Versão dos pacotes incrementada: ${current_version} -> ${package_version}"

"${project_root}/scripts/build-portable-client-binaries.sh"
"${project_root}/scripts/build-debian-package.sh"
"${project_root}/scripts/build-server-package.sh" "${package_version}"

echo "Pacotes Debian disponíveis em ${project_root}/dist:"
echo "  compasso-client_${package_version}_${package_architecture}.deb"
echo "  compasso-server_${package_version}_all.deb"
