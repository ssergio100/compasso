#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "erro: a recuperação exige root" >&2
  exit 1
fi

pam_service_path="/etc/pam.d/gdm-password"
pam_backup_path="${pam_service_path}.compasso.bak"
emergency_bypass_path="/run/compasso-pam-bypass"

# The helper checks this root-owned volatile file before reading policy. It is
# an independent escape hatch even if the PAM backup cannot be restored.
install -o root -g root -m 0600 /dev/null "${emergency_bypass_path}"

if [[ -f "${pam_backup_path}" ]]; then
  /usr/sbin/tempo-pam-setup -action uninstall
fi
systemctl stop tempo-agent.service || true
if grep -Fq "BEGIN COMPASSO PAM GATE" "${pam_service_path}"; then
  logger --tag compasso-pilot-recovery "bypass emergencial ativado; backup PAM indisponível"
  echo "Recuperação parcial: bypass de login ativado e agente parado."
  exit 0
fi
rm -f "${emergency_bypass_path}"
logger --tag compasso-pilot-recovery "PAM restaurado e tempo-agent parado para permitir recuperação de login"
echo "Recuperação concluída: PAM original restaurado e agente parado."
