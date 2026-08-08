#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "erro: execute este instalador com sudo" >&2
  exit 1
fi

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
installation_started_at="$(date --iso-8601=seconds)"
agent_binary="${project_root}/bin/tempo-agent"
systemd_unit_source="${project_root}/packaging/systemd/tempo-agent.service"
dbus_policy_source="${project_root}/packaging/dbus/br.com.tempo.Agent.conf"
configuration_directory="/etc/tempo-agent"
configuration_path="${configuration_directory}/config.toml"
state_directory="/var/lib/tempo-agent"

if [[ ! -x "${agent_binary}" ]]; then
  echo "erro: bin/tempo-agent não existe; execute make build-agent sem sudo" >&2
  exit 1
fi

default_controlled_user="${SUDO_USER:-}"
read -r -p "Usuário Linux controlado [${default_controlled_user}]: " controlled_user
controlled_user="${controlled_user:-${default_controlled_user}}"
if [[ -z "${controlled_user}" ]] || ! getent passwd "${controlled_user}" >/dev/null; then
  echo "erro: usuário Linux inexistente" >&2
  exit 1
fi

read -r -p "URL do servidor [http://127.0.0.1:8081]: " server_url
server_url="${server_url:-http://127.0.0.1:8081}"
if [[ ! "${server_url}" =~ ^https:// ]] &&
   [[ ! "${server_url}" =~ ^http://(127\.[0-9.]+|localhost|\[::1\])(:[0-9]+)?$ ]]; then
  echo "erro: servidores remotos exigem HTTPS" >&2
  exit 1
fi

read -r -p "device_id exibido no painel: " device_id
read -r -s -p "device_token exibido no painel: " device_token
printf '\n'
if [[ ! "${device_id}" =~ ^[A-Za-z0-9-]+$ ]]; then
  echo "erro: device_id inválido" >&2
  exit 1
fi
if [[ ! "${device_token}" =~ ^[A-Za-z0-9_-]{40,}$ ]]; then
  echo "erro: device_token inválido" >&2
  exit 1
fi

install -d -o root -g root -m 0700 "${configuration_directory}" "${state_directory}"
if [[ -f "${configuration_path}" ]]; then
  backup_timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
  configuration_backup="${configuration_path}.backup-${backup_timestamp}"
  install -o root -g root -m 0600 "${configuration_path}" "${configuration_backup}"
  echo "Configuração anterior preservada em ${configuration_backup}."
fi

temporary_configuration="$(mktemp "${configuration_directory}/.config.toml.XXXXXX")"
cleanup_temporary_configuration() {
  rm -f "${temporary_configuration}"
}
trap cleanup_temporary_configuration EXIT
chmod 0600 "${temporary_configuration}"
{
  printf 'database_path = "/var/lib/tempo-agent/tempo-agent.db"\n'
  printf 'controlled_user = "%s"\n' "${controlled_user}"
  printf 'tick_interval = "1s"\n'
  printf 'checkpoint_interval = "5s"\n'
  printf 'loginctl_path = "/usr/bin/loginctl"\n'
  printf 'server_url = "%s"\n' "${server_url}"
  printf 'device_id = "%s"\n' "${device_id}"
  printf 'device_token = "%s"\n' "${device_token}"
  printf 'heartbeat_interval = "10s"\n'
  printf 'http_timeout = "8s"\n'
} > "${temporary_configuration}"
chown root:root "${temporary_configuration}"

install -o root -g root -m 0755 "${agent_binary}" /usr/sbin/tempo-agent
install -o root -g root -m 0644 "${systemd_unit_source}" /etc/systemd/system/tempo-agent.service
install -o root -g root -m 0644 "${dbus_policy_source}" /etc/dbus-1/system.d/br.com.tempo.Agent.conf
mv -f "${temporary_configuration}" "${configuration_path}"
trap - EXIT
chown root:root "${configuration_path}"
chmod 0600 "${configuration_path}"

/usr/sbin/tempo-agent -check-config -config "${configuration_path}"
systemctl daemon-reload
systemctl enable --now tempo-agent.service

if ! systemctl is-active --quiet tempo-agent.service; then
  echo "erro: tempo-agent não permaneceu ativo após a instalação" >&2
  journalctl -u tempo-agent.service --since "${installation_started_at}" --no-pager -n 30 >&2
  exit 1
fi
if journalctl -u tempo-agent.service --since "${installation_started_at}" --no-pager | grep -Fq -- "${device_token}"; then
  echo "erro: o token do dispositivo apareceu no journal" >&2
  exit 1
fi

echo "tempo-agent instalado com configuração 0600 e estado 0700."
echo "O gate PAM não foi instalado por este procedimento."
