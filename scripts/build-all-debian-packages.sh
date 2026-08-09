#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
package_version="$(sed -n 's/^Version: //p' "${project_root}/packaging/debian/control")"
package_architecture="$(sed -n 's/^Architecture: //p' "${project_root}/packaging/debian/control")"

"${project_root}/scripts/build-portable-client-binaries.sh"
"${project_root}/scripts/build-debian-package.sh"
"${project_root}/scripts/build-server-package.sh"

echo "Pacotes Debian disponíveis em ${project_root}/dist:"
echo "  compasso-client_${package_version}_${package_architecture}.deb"
echo "  compasso-server_${package_version}_all.deb"
