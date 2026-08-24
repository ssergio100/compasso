#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
systemd_unit="${project_root}/packaging/systemd/tempo-agent.service"
agent_configuration_helper="${project_root}/agent/cmd/tempo-agent-configure/main.go"
dockerfile="${project_root}/server/Dockerfile"
admin_compose="${project_root}/deploy/admin-ui/compose.yml"
admin_nginx="${project_root}/deploy/admin-ui/default.conf"
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

bash -n "${project_root}/scripts/build-debian-package.sh"
bash -n "${project_root}/packaging/debian/postinst"
bash -n "${project_root}/packaging/debian/prerm"
bash -n "${project_root}/packaging/debian/postrm"
grep -Fq '"restart", "tempo-agent.service"' "${agent_configuration_helper}"
grep -Fq 'waitForSuccessfulSynchronization' "${agent_configuration_helper}"
grep -Fq 'send_member="GetSynchronizationReport"' "${project_root}/packaging/dbus/br.com.tempo.Agent.conf"
bash -n "${project_root}/scripts/install-server.sh"
bash -n "${project_root}/scripts/backup-server.sh"
bash -n "${project_root}/scripts/restore-server-backup.sh"
bash -n "${project_root}/scripts/update-server.sh"
bash -n "${project_root}/scripts/publish-server.sh"
grep -Fqx 'USER tempo-server:tempo-server' "${dockerfile}"
if grep -Eq 'server/web/(templates|static)' "${dockerfile}"; then
  echo "erro: imagem da API ainda copia o frontend" >&2
  exit 1
fi
grep -Fq 'image: nginx:alpine' "${admin_compose}"
grep -Fq '/srv/sites/compasso-admin-ui:/usr/share/nginx/html:ro' "${admin_compose}"
grep -Fq 'X-Content-Type-Options "nosniff"' "${admin_nginx}"
grep -Fq 'X-Frame-Options "DENY"' "${admin_nginx}"
grep -Fqx '**/config.toml' "${dockerignore}"
grep -Fqx 'secrets' "${dockerignore}"
if command -v docker >/dev/null 2>&1; then
  COMPASSO_DATA_DIRECTORY=/tmp/compasso-compose-validation \
    docker compose --project-directory "${project_root}" config --quiet
fi
echo "hardening systemd e empacotamento validados"
