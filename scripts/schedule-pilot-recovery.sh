#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "erro: execute com sudo" >&2
  exit 1
fi

recovery_delay="${1:-5min}"
if [[ ! "${recovery_delay}" =~ ^[1-9][0-9]*(s|min)$ ]]; then
  echo "erro: atraso deve usar o formato 90s ou 5min" >&2
  exit 1
fi

systemctl stop compasso-pilot-recovery.timer 2>/dev/null || true
systemctl reset-failed compasso-pilot-recovery.service 2>/dev/null || true
systemd-run \
  --unit=compasso-pilot-recovery \
  --on-active="${recovery_delay}" \
  --timer-property=AccuracySec=1s \
  /usr/sbin/tempo-pilot-recover

echo "Recuperação automática agendada para daqui a ${recovery_delay}."
