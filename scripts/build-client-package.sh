#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
package_version="${1:-0.1.0-pilot}"
if [[ ! "${package_version}" =~ ^[0-9]+.[0-9]+.[0-9]+(-[A-Za-z0-9.]+)?$ ]]; then
  echo "erro: versão inválida: ${package_version}" >&2
  exit 1
fi

package_architecture="$(go env GOARCH)"
package_name="compasso-client-${package_version}-linux-${package_architecture}"
temporary_root="$(mktemp -d)"
package_root="${temporary_root}/${package_name}"
cleanup_temporary_root() {
  rm -rf "${temporary_root}"
}
trap cleanup_temporary_root EXIT

required_sources=(
  "${project_root}/bin/tempo-agent"
  "${project_root}/bin/tempo-pam-check"
  "${project_root}/bin/tempo-pam-setup"
  "${project_root}/local-ui/bonus_dialog.py"
  "${project_root}/packaging/applications/br.com.tempo.LocalBonus.desktop"
  "${project_root}/packaging/dbus/br.com.tempo.Agent.conf"
  "${project_root}/packaging/systemd/tempo-agent.service"
  "${project_root}/scripts/install-agent-securely.sh"
  "${project_root}/scripts/install-pilot-components.sh"
  "${project_root}/scripts/recover-pilot-login.sh"
  "${project_root}/scripts/schedule-pilot-recovery.sh"
  "${project_root}/scripts/uninstall-compasso.sh"
  "${project_root}/docs/phase-10.md"
)
for required_source in "${required_sources[@]}"; do
  if [[ ! -f "${required_source}" ]]; then
    echo "erro: artefato ausente: ${required_source}" >&2
    exit 1
  fi
done

install -d \
  "${package_root}/bin" \
  "${package_root}/local-ui" \
  "${package_root}/packaging/applications" \
  "${package_root}/packaging/dbus" \
  "${package_root}/packaging/systemd" \
  "${package_root}/scripts" \
  "${package_root}/docs" \
  "${project_root}/dist"
install -m 0755 "${project_root}/bin/tempo-agent" "${package_root}/bin/tempo-agent"
install -m 0755 "${project_root}/bin/tempo-pam-check" "${package_root}/bin/tempo-pam-check"
install -m 0755 "${project_root}/bin/tempo-pam-setup" "${package_root}/bin/tempo-pam-setup"
install -m 0755 "${project_root}/local-ui/bonus_dialog.py" "${package_root}/local-ui/bonus_dialog.py"
install -m 0644 "${project_root}/packaging/applications/br.com.tempo.LocalBonus.desktop" "${package_root}/packaging/applications/"
install -m 0644 "${project_root}/packaging/dbus/br.com.tempo.Agent.conf" "${package_root}/packaging/dbus/"
install -m 0644 "${project_root}/packaging/systemd/tempo-agent.service" "${package_root}/packaging/systemd/"
for installer_script in \
  install-agent-securely.sh \
  install-pilot-components.sh \
  recover-pilot-login.sh \
  schedule-pilot-recovery.sh \
  uninstall-compasso.sh; do
  install -m 0755 "${project_root}/scripts/${installer_script}" "${package_root}/scripts/${installer_script}"
done
install -m 0644 "${project_root}/docs/phase-10.md" "${package_root}/README.md"

package_archive="${project_root}/dist/${package_name}.tar.gz"
tar -C "${temporary_root}" -czf "${package_archive}" "${package_name}"
sha256sum "${package_archive}" > "${package_archive}.sha256"
echo "Pacote criado em ${package_archive}"
