#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
package_version="$(sed -n 's/^Version: //p' "${project_root}/packaging/debian/control")"
package_architecture="$(sed -n 's/^Architecture: //p' "${project_root}/packaging/debian/control")"
package_path="${1:-${project_root}/dist/compasso-client_${package_version}_${package_architecture}.deb}"
temporary_directory="$(mktemp -d)"

cleanup_temporary_directory() {
  rm -rf "${temporary_directory}"
}
trap cleanup_temporary_directory EXIT

if [[ ! -f "${package_path}" ]]; then
  echo "erro: pacote ausente: ${package_path}" >&2
  exit 1
fi

if [[ "$(dpkg-deb --field "${package_path}" Package)" != "compasso-client" ]]; then
  echo "erro: nome de pacote inesperado" >&2
  exit 1
fi
if [[ "$(dpkg-deb --field "${package_path}" Architecture)" != "amd64" ]]; then
  echo "erro: arquitetura de pacote inesperada" >&2
  exit 1
fi
if dpkg-deb --field "${package_path}" Depends | grep -Eqi '(^|[, ])docker([, ]|$)'; then
  echo "erro: o cliente não pode depender de Docker" >&2
  exit 1
fi
if dpkg-deb --field "${package_path}" Depends | grep -Eq '(^|, )libpam-runtime([, ]|$)'; then
  echo "erro: o cliente não usa PAM e não deve depender de libpam-runtime" >&2
  exit 1
fi
if ! dpkg-deb --field "${package_path}" Depends | grep -Eq '(^|, )pkexec([, ]|$)'; then
  echo "erro: o pacote não declara o helper de autorização gráfica pkexec" >&2
  exit 1
fi
if dpkg-deb --contents "${package_path}" | grep -Eqi 'dockerfile|docker-compose|compose[.]ya?ml'; then
  echo "erro: o pacote contém configuração de contêiner" >&2
  exit 1
fi
if dpkg-deb --contents "${package_path}" | grep -Fq 'br.com.tempo.LocalBonus.desktop'; then
  echo "erro: o pacote publica um segundo ícone para uma ação interna do Compasso" >&2
  exit 1
fi
if dpkg-deb --contents "${package_path}" | grep -Eq \
  'tempo-pam|pamgate|tempo-pilot|schedule-recovery'; then
  echo "erro: o pacote contém componentes mortos do protótipo PAM/piloto" >&2
  exit 1
fi

dpkg-deb --extract "${package_path}" "${temporary_directory}/root"
dpkg-deb --control "${package_path}" "${temporary_directory}/control"

for maintainer_script in postinst prerm postrm; do
  sh -n "${temporary_directory}/control/${maintainer_script}"
done
if ! grep -Fq 'systemctl disable --now tempo-agent.service' \
  "${temporary_directory}/control/postinst"; then
	  echo "erro: postinst não mantém uma instalação nova parada" >&2
  exit 1
fi
if ! grep -Fq 'if [ -f /etc/tempo-agent/setup-complete ]' \
  "${temporary_directory}/control/postinst"; then
	  echo "erro: postinst não distingue instalação nova de cliente configurado" >&2
	  exit 1
fi
if ! grep -Fq 'if [ -z "${2:-}" ]' \
  "${temporary_directory}/control/postinst" || \
   ! grep -Fq 'rm -f /etc/tempo-agent/setup-complete' \
  "${temporary_directory}/control/postinst"; then
	  echo "erro: postinst não invalida configuração residual em instalação nova" >&2
  exit 1
fi
if ! grep -Fq 'rm -f /var/lib/tempo-agent/tempo-agent.db' \
  "${temporary_directory}/control/postinst"; then
	  echo "erro: postinst não remove política residual em instalação nova" >&2
  exit 1
fi
if ! grep -Fq 'systemctl restart tempo-agent.service' \
  "${temporary_directory}/control/postinst"; then
	  echo "erro: postinst não reinicia o cliente configurado após atualização" >&2
  exit 1
fi
if ! grep -Fq 'rm -f /etc/systemd/system/tempo-agent.service' \
  "${temporary_directory}/control/postinst"; then
  echo "erro: postinst não remove a unidade legada que sombreia o pacote" >&2
  exit 1
fi

binary_paths=(
  usr/sbin/tempo-agent
  usr/sbin/tempo-agent-configure
)
for relative_binary_path in "${binary_paths[@]}"; do
  binary_path="${temporary_directory}/root/${relative_binary_path}"
  if [[ ! -x "${binary_path}" ]]; then
    echo "erro: binário ausente: ${relative_binary_path}" >&2
    exit 1
  fi
  if ldd "${binary_path}" | grep -Fq "libgo"; then
    echo "erro: ${relative_binary_path} depende de libgo" >&2
    exit 1
  fi
  if ldd "${binary_path}" | grep -Fq "not found"; then
    echo "erro: ${relative_binary_path} possui dependência ausente" >&2
    exit 1
  fi
done

required_package_paths=(
  usr/bin/tempo-local-bonus
  usr/bin/compasso-agent-setup
  usr/share/applications/br.com.compasso.Compasso.desktop
  etc/xdg/autostart/br.com.compasso.AgentSetup.desktop
  usr/share/dbus-1/system.d/br.com.tempo.Agent.conf
  usr/share/polkit-1/actions/br.com.compasso.AgentSetup.policy
  usr/share/metainfo/br.com.compasso.Compasso.metainfo.xml
)
for required_package_path in "${required_package_paths[@]}"; do
  if [[ ! -f "${temporary_directory}/root/${required_package_path}" ]]; then
    echo "erro: arquivo de configuração inicial ausente: ${required_package_path}" >&2
    exit 1
  fi
done

if ! grep -Fq 'GetSynchronizationReport' "${temporary_directory}/root/usr/bin/tempo-local-bonus"; then
  echo "erro: interface local não consulta o diagnóstico de sincronização" >&2
  exit 1
fi
if ! grep -Fq 'send_member="GetSynchronizationReport"' "${temporary_directory}/root/usr/share/dbus-1/system.d/br.com.tempo.Agent.conf"; then
  echo "erro: pacote não expõe o diagnóstico humano de sincronização" >&2
  exit 1
fi

configuration_path="${temporary_directory}/root/etc/tempo-agent/config.toml"
if [[ "$(stat -c '%a' "${configuration_path}")" != "600" ]]; then
  echo "erro: configuração não possui modo 0600" >&2
  exit 1
fi

if command -v appstreamcli >/dev/null 2>&1; then
  appstreamcli validate --no-net \
    "${temporary_directory}/root/usr/share/metainfo/br.com.compasso.Compasso.metainfo.xml"
fi

"${temporary_directory}/root/usr/sbin/tempo-agent" \
  -check-config \
  -config "${configuration_path}"

echo "Pacote Debian validado sem instalação no sistema local."
