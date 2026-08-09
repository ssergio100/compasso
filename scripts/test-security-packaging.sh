#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
systemd_unit="${project_root}/packaging/systemd/tempo-agent.service"
installer="${project_root}/scripts/install-agent-securely.sh"
pilot_installer="${project_root}/scripts/install-pilot-components.sh"
pilot_uninstaller="${project_root}/scripts/uninstall-compasso.sh"
pilot_recovery="${project_root}/scripts/recover-pilot-login.sh"
package_builder="${project_root}/scripts/build-client-package.sh"
agent_configuration_helper="${project_root}/agent/cmd/tempo-agent-configure/main.go"
dockerfile="${project_root}/server/Dockerfile"
admin_dockerfile="${project_root}/admin-ui/Dockerfile"
dockerignore="${project_root}/.dockerignore"

require_unit_directive() {
  directive="$1"
  if ! grep -Fqx "${directive}" "${systemd_unit}"; then
    echo "erro: diretiva obrigatória ausente: ${directive}" >&2
    exit 1
  fi
}

required_directives=(
  "User=root"
  "Group=root"
  "StateDirectoryMode=0700"
  "RuntimeDirectory=tempo-agent"
  "RuntimeDirectoryMode=0700"
  "RuntimeDirectoryPreserve=restart"
  "UMask=0077"
  "NoNewPrivileges=true"
  "ProtectSystem=strict"
  "ProtectHome=true"
  "PrivateTmp=true"
  "PrivateDevices=true"
  "ProtectKernelTunables=true"
  "ProtectKernelModules=true"
  "ProtectControlGroups=true"
  "RestrictSUIDSGID=true"
  "RestrictNamespaces=true"
  "CapabilityBoundingSet="
  "ReadWritePaths=/var/lib/tempo-agent"
)

for required_directive in "${required_directives[@]}"; do
  require_unit_directive "${required_directive}"
done

bash -n "${installer}"
bash -n "${pilot_installer}"
bash -n "${pilot_uninstaller}"
bash -n "${pilot_recovery}"
bash -n "${project_root}/scripts/schedule-pilot-recovery.sh"
bash -n "${package_builder}"
grep -Fq '"restart", "tempo-agent.service"' "${agent_configuration_helper}"
grep -Fq 'waitForSuccessfulSynchronization' "${agent_configuration_helper}"
bash -n "${project_root}/scripts/install-server.sh"
bash -n "${project_root}/scripts/backup-server.sh"
bash -n "${project_root}/scripts/restore-server-backup.sh"
bash -n "${project_root}/scripts/update-server.sh"
grep -Fqx 'USER tempo-server:tempo-server' "${dockerfile}"
if grep -Eq 'server/web/(templates|static)' "${dockerfile}"; then
  echo "erro: imagem da API ainda copia o frontend" >&2
  exit 1
fi
grep -Fqx 'USER nginx:nginx' "${admin_dockerfile}"
if grep -Eq 'COPY .*server|tempo-server|go build' "${admin_dockerfile}"; then
  echo "erro: imagem do frontend contém referência ao backend" >&2
  exit 1
fi
grep -Fqx '**/config.toml' "${dockerignore}"
grep -Fqx 'secrets' "${dockerignore}"
if command -v docker >/dev/null 2>&1; then
  COMPASSO_DATA_DIRECTORY=/tmp/compasso-compose-validation \
    docker compose --project-directory "${project_root}" config --quiet
fi
echo "hardening systemd, instaladores e pacote Docker validados"
