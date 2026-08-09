#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
debian_metadata_directory="${project_root}/packaging/debian"
temporary_directory="$(mktemp -d)"

cleanup_temporary_directory() {
  rm -rf "${temporary_directory}"
}
trap cleanup_temporary_directory EXIT

required_sources=(
  "${project_root}/bin/tempo-agent"
  "${project_root}/bin/tempo-agent-configure"
  "${project_root}/bin/compasso-session-logout"
  "${project_root}/bin/tempo-pam-check"
  "${project_root}/bin/tempo-pam-setup"
  "${project_root}/local-ui/bonus_dialog.py"
  "${project_root}/local-ui/configure_agent.py"
  "${project_root}/packaging/applications/br.com.compasso.AgentSetup.desktop"
  "${project_root}/packaging/applications/br.com.tempo.LocalBonus.desktop"
  "${project_root}/packaging/autostart/br.com.compasso.AgentSetup.desktop"
  "${project_root}/packaging/config/tempo-agent.toml"
  "${project_root}/packaging/dbus/br.com.tempo.Agent.conf"
  "${project_root}/packaging/metainfo/br.com.tempo.LocalBonus.metainfo.xml"
  "${project_root}/packaging/metainfo/br.com.compasso.AgentSetup.metainfo.xml"
  "${project_root}/packaging/polkit/br.com.compasso.AgentSetup.policy"
  "${project_root}/packaging/systemd/tempo-agent.service"
  "${project_root}/scripts/recover-pilot-login.sh"
  "${project_root}/scripts/schedule-pilot-recovery.sh"
)
for required_source in "${required_sources[@]}"; do
  if [[ ! -f "${required_source}" ]]; then
    echo "erro: artefato ausente: ${required_source}" >&2
    exit 1
  fi
done

binary_names=(tempo-agent tempo-agent-configure compasso-session-logout tempo-pam-check tempo-pam-setup)
for binary_name in "${binary_names[@]}"; do
  if ldd "${project_root}/bin/${binary_name}" | grep -Fq "libgo"; then
    echo "erro: ${binary_name} depende de libgo; use make build-agent-portable" >&2
    exit 1
  fi
done

package_name="$(dpkg-deb --field "${debian_metadata_directory}/control" Package 2>/dev/null || true)"
package_version="$(sed -n 's/^Version: //p' "${debian_metadata_directory}/control")"
package_architecture="$(sed -n 's/^Architecture: //p' "${debian_metadata_directory}/control")"
if [[ -z "${package_name}" ]]; then
  package_name="$(sed -n 's/^Package: //p' "${debian_metadata_directory}/control")"
fi
if [[ -z "${package_name}" || -z "${package_version}" || -z "${package_architecture}" ]]; then
  echo "erro: metadados Debian incompletos" >&2
  exit 1
fi
if ! dpkg --validate-version "${package_version}"; then
  echo "erro: versão Debian inválida: ${package_version}" >&2
  exit 1
fi

package_root="${temporary_directory}/${package_name}"
install -d \
  "${package_root}/DEBIAN" \
  "${package_root}/etc/tempo-agent" \
  "${package_root}/etc/xdg/autostart" \
  "${package_root}/usr/bin" \
  "${package_root}/usr/lib/systemd/system" \
  "${package_root}/usr/libexec" \
  "${package_root}/usr/sbin" \
  "${package_root}/usr/share/applications" \
  "${package_root}/usr/share/dbus-1/system.d" \
  "${package_root}/usr/share/doc/compasso-client" \
  "${package_root}/usr/share/metainfo"
install -d "${package_root}/usr/share/polkit-1/actions"

install -m 0644 "${debian_metadata_directory}/control" "${package_root}/DEBIAN/control"
install -m 0644 "${debian_metadata_directory}/conffiles" "${package_root}/DEBIAN/conffiles"
for maintainer_script in postinst prerm postrm; do
  install -m 0755 \
    "${debian_metadata_directory}/${maintainer_script}" \
    "${package_root}/DEBIAN/${maintainer_script}"
done

install -m 0600 \
  "${project_root}/packaging/config/tempo-agent.toml" \
  "${package_root}/etc/tempo-agent/config.toml"
install -m 0755 "${project_root}/bin/tempo-agent" "${package_root}/usr/sbin/tempo-agent"
install -m 0755 "${project_root}/bin/tempo-agent-configure" "${package_root}/usr/sbin/tempo-agent-configure"
install -m 0755 "${project_root}/bin/compasso-session-logout" "${package_root}/usr/libexec/compasso-session-logout"
install -m 0755 "${project_root}/bin/tempo-pam-check" "${package_root}/usr/libexec/tempo-pam-check"
install -m 0755 "${project_root}/bin/tempo-pam-setup" "${package_root}/usr/sbin/tempo-pam-setup"
install -m 0755 "${project_root}/local-ui/bonus_dialog.py" "${package_root}/usr/bin/tempo-local-bonus"
install -m 0755 "${project_root}/local-ui/configure_agent.py" "${package_root}/usr/bin/compasso-agent-setup"
install -m 0755 "${project_root}/scripts/recover-pilot-login.sh" "${package_root}/usr/sbin/tempo-pilot-recover"
install -m 0755 "${project_root}/scripts/schedule-pilot-recovery.sh" "${package_root}/usr/sbin/tempo-schedule-recovery"
install -m 0644 \
  "${project_root}/packaging/applications/br.com.compasso.AgentSetup.desktop" \
  "${package_root}/usr/share/applications/br.com.compasso.AgentSetup.desktop"
install -m 0644 \
  "${project_root}/packaging/applications/br.com.tempo.LocalBonus.desktop" \
  "${package_root}/usr/share/applications/br.com.tempo.LocalBonus.desktop"
install -m 0644 \
  "${project_root}/packaging/autostart/br.com.compasso.AgentSetup.desktop" \
  "${package_root}/etc/xdg/autostart/br.com.compasso.AgentSetup.desktop"
install -m 0644 \
  "${project_root}/packaging/dbus/br.com.tempo.Agent.conf" \
  "${package_root}/usr/share/dbus-1/system.d/br.com.tempo.Agent.conf"
install -m 0644 \
  "${project_root}/packaging/metainfo/br.com.compasso.AgentSetup.metainfo.xml" \
  "${package_root}/usr/share/metainfo/br.com.compasso.AgentSetup.metainfo.xml"
install -m 0644 \
  "${project_root}/packaging/metainfo/br.com.tempo.LocalBonus.metainfo.xml" \
  "${package_root}/usr/share/metainfo/br.com.tempo.LocalBonus.metainfo.xml"
install -m 0644 \
  "${project_root}/packaging/polkit/br.com.compasso.AgentSetup.policy" \
  "${package_root}/usr/share/polkit-1/actions/br.com.compasso.AgentSetup.policy"
install -m 0644 \
  "${project_root}/packaging/systemd/tempo-agent.service" \
  "${package_root}/usr/lib/systemd/system/tempo-agent.service"
install -m 0644 \
  "${project_root}/docs/portable-client-plan.md" \
  "${package_root}/usr/share/doc/compasso-client/README.md"

install -d "${project_root}/dist"
package_path="${project_root}/dist/${package_name}_${package_version}_${package_architecture}.deb"
dpkg-deb --root-owner-group --build "${package_root}" "${package_path}"
sha256sum "${package_path}" > "${package_path}.sha256"

echo "Pacote Debian criado em ${package_path}"
