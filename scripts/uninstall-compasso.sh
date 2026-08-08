#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "erro: execute a desinstalação com sudo" >&2
  exit 1
fi

pam_service_path="/etc/pam.d/gdm-password"
pam_backup_path="${pam_service_path}.compasso.bak"
if [[ -f "${pam_backup_path}" ]]; then
  /usr/sbin/tempo-pam-setup -action uninstall
fi
if grep -Fq "BEGIN COMPASSO PAM GATE" "${pam_service_path}"; then
  echo "erro: regra PAM ainda presente; nenhum binário será removido" >&2
  exit 1
fi

systemctl stop compasso-pilot-recovery.timer compasso-pilot-recovery.service 2>/dev/null || true
systemctl disable --now tempo-agent.service 2>/dev/null || true

rm -f /etc/systemd/system/tempo-agent.service
rm -f /etc/dbus-1/system.d/br.com.tempo.Agent.conf
rm -f /usr/share/applications/br.com.tempo.LocalBonus.desktop
rm -f /usr/sbin/tempo-agent
rm -f /usr/libexec/tempo-pam-check
rm -f /usr/sbin/tempo-pam-setup
rm -f /usr/bin/tempo-local-bonus
rm -f /usr/sbin/tempo-pilot-recover
rm -f /usr/sbin/tempo-schedule-recovery
rm -f /usr/sbin/tempo-uninstall
rm -f /run/compasso-pam-bypass

systemctl daemon-reload
busctl call org.freedesktop.DBus /org/freedesktop/DBus org.freedesktop.DBus ReloadConfig

echo "Compasso removido e PAM original preservado."
echo "Configuração e estado foram mantidos em /etc/tempo-agent e /var/lib/tempo-agent."
