#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "erro: execute este instalador com sudo" >&2
  exit 1
fi

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
required_sources=(
  "${project_root}/bin/tempo-agent"
  "${project_root}/bin/tempo-pam-check"
  "${project_root}/bin/tempo-pam-setup"
  "${project_root}/local-ui/bonus_dialog.py"
  "${project_root}/packaging/applications/br.com.tempo.LocalBonus.desktop"
  "${project_root}/packaging/dbus/br.com.tempo.Agent.conf"
  "${project_root}/packaging/systemd/tempo-agent.service"
  "${project_root}/scripts/recover-pilot-login.sh"
  "${project_root}/scripts/schedule-pilot-recovery.sh"
  "${project_root}/scripts/uninstall-compasso.sh"
  "${project_root}/scripts/install-agent-securely.sh"
)
for required_source in "${required_sources[@]}"; do
  if [[ ! -f "${required_source}" ]]; then
    echo "erro: artefato ausente: ${required_source}" >&2
    exit 1
  fi
done
for required_command in /usr/bin/notify-send /usr/bin/systemd-run /usr/bin/python3; do
  if [[ ! -x "${required_command}" ]]; then
    echo "erro: dependência ausente: ${required_command}" >&2
    exit 1
  fi
done
if ! /usr/bin/python3 -c 'import gi; gi.require_version("Gtk", "4.0"); from gi.repository import Gtk' >/dev/null 2>&1; then
  echo "erro: Python GTK 4 não está disponível" >&2
  echo "No Debian/Zorin, instale python3-gi e gir1.2-gtk-4.0." >&2
  exit 1
fi
if [[ ! -f /etc/tempo-agent/config.toml ]]; then
  echo "Configuração inicial necessária; iniciando pareamento seguro."
  "${project_root}/scripts/install-agent-securely.sh"
fi

install -o root -g root -m 0755 "${project_root}/bin/tempo-agent" /usr/sbin/tempo-agent
install -o root -g root -m 0755 "${project_root}/bin/tempo-pam-check" /usr/libexec/tempo-pam-check
install -o root -g root -m 0755 "${project_root}/bin/tempo-pam-setup" /usr/sbin/tempo-pam-setup
install -o root -g root -m 0755 "${project_root}/local-ui/bonus_dialog.py" /usr/bin/tempo-local-bonus
install -o root -g root -m 0755 "${project_root}/scripts/recover-pilot-login.sh" /usr/sbin/tempo-pilot-recover
install -o root -g root -m 0755 "${project_root}/scripts/schedule-pilot-recovery.sh" /usr/sbin/tempo-schedule-recovery
install -o root -g root -m 0755 "${project_root}/scripts/uninstall-compasso.sh" /usr/sbin/tempo-uninstall
install -o root -g root -m 0644 "${project_root}/packaging/applications/br.com.tempo.LocalBonus.desktop" /usr/share/applications/br.com.tempo.LocalBonus.desktop
install -o root -g root -m 0644 "${project_root}/packaging/dbus/br.com.tempo.Agent.conf" /etc/dbus-1/system.d/br.com.tempo.Agent.conf
install -o root -g root -m 0644 "${project_root}/packaging/systemd/tempo-agent.service" /etc/systemd/system/tempo-agent.service

/usr/sbin/tempo-agent -check-config -config /etc/tempo-agent/config.toml
systemctl daemon-reload
busctl call org.freedesktop.DBus /org/freedesktop/DBus org.freedesktop.DBus ReloadConfig
systemctl enable --now tempo-agent.service
systemctl restart tempo-agent.service
if ! systemctl is-active --quiet tempo-agent.service; then
  echo "erro: tempo-agent ficou inativo após atualização" >&2
  exit 1
fi

echo "Componentes piloto instalados; PAM ainda não foi alterado."
echo "Recuperação disponível em /usr/sbin/tempo-schedule-recovery."
