#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
systemd_unit="${project_root}/packaging/systemd/tempo-agent.service"
installer="${project_root}/scripts/install-agent-securely.sh"
dockerfile="${project_root}/server/Dockerfile"
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
grep -Fqx 'USER tempo-server:tempo-server' "${dockerfile}"
grep -Fqx '**/config.toml' "${dockerignore}"
if command -v docker >/dev/null 2>&1; then
  docker compose -f "${project_root}/compose.production.yml" config --quiet
fi
echo "hardening systemd, instalador e configuração Docker validados"
