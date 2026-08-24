#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist_directory="${project_root}/dist"

requested_version=""
server_host="${COMPASSO_DEPLOY_HOST:-}"
ssh_user="${COMPASSO_DEPLOY_USER:-${USER:-}}"
ssh_port="${COMPASSO_DEPLOY_PORT:-22}"
identity_file="${COMPASSO_DEPLOY_IDENTITY:-}"
health_url="${COMPASSO_DEPLOY_HEALTH_URL:-}"
run_tests=true
build_only=false
assume_yes=false
deployment_started=false

show_usage() {
  cat <<'EOF'
Uso: ./scripts/publish-server.sh [opções]

Gera, valida e, por padrão, publica um pacote do Compasso Server.
Sem opções, todas as informações relevantes são perguntadas.

Opções:
  --version VERSAO       versão Debian ou piloto (ex.: 28, pilot28 ou 0.1.0~pilot28)
  --host ENDERECO        endereço SSH do servidor
  --user USUARIO         usuário SSH
  --port PORTA           porta SSH (padrão: 22)
  --identity ARQUIVO     chave SSH específica
  --health-url URL       healthcheck externo (padrão: http://ENDERECO:8181/healthz)
  --skip-tests           não executa os testes locais
  --build-only           gera e valida o pacote, sem acessar servidor
  --yes                  aceita confirmações não críticas
  -h, --help             mostra esta ajuda

Variáveis equivalentes: COMPASSO_DEPLOY_HOST, COMPASSO_DEPLOY_USER,
COMPASSO_DEPLOY_PORT, COMPASSO_DEPLOY_IDENTITY e COMPASSO_DEPLOY_HEALTH_URL.
EOF
}

fail() {
  echo "erro: $*" >&2
  exit 1
}

on_error() {
  local exit_code=$?
  if [[ "${deployment_started}" == true ]]; then
    echo >&2
    echo "A publicação remota não terminou." >&2
    echo "O pacote foi mantido no servidor para diagnóstico." >&2
    echo "Consulte: ssh -p ${ssh_port} ${ssh_user}@${server_host} sudo /opt/compasso-server/scripts/status-server.sh" >&2
  fi
  exit "${exit_code}"
}
trap on_error ERR

prompt_value() {
  local label="$1"
  local default_value="$2"
  local answer=""
  if [[ -n "${default_value}" ]]; then
    read -r -p "${label} [${default_value}]: " answer
    printf '%s' "${answer:-${default_value}}"
  else
    read -r -p "${label}: " answer
    printf '%s' "${answer}"
  fi
}

confirm() {
  local question="$1"
  local default_answer="${2:-yes}"
  local prompt="[S/n]"
  local answer=""

  if [[ "${assume_yes}" == true ]]; then
    return 0
  fi
  if [[ "${default_answer}" == no ]]; then
    prompt="[s/N]"
  fi
  read -r -p "${question} ${prompt} " answer
  if [[ -z "${answer}" ]]; then
    [[ "${default_answer}" == yes ]]
    return
  fi
  [[ "${answer}" =~ ^[sS]$ ]]
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) [[ $# -ge 2 ]] || fail "--version exige um valor"; requested_version="$2"; shift 2 ;;
    --host) [[ $# -ge 2 ]] || fail "--host exige um valor"; server_host="$2"; shift 2 ;;
    --user) [[ $# -ge 2 ]] || fail "--user exige um valor"; ssh_user="$2"; shift 2 ;;
    --port) [[ $# -ge 2 ]] || fail "--port exige um valor"; ssh_port="$2"; shift 2 ;;
    --identity) [[ $# -ge 2 ]] || fail "--identity exige um valor"; identity_file="$2"; shift 2 ;;
    --health-url) [[ $# -ge 2 ]] || fail "--health-url exige um valor"; health_url="$2"; shift 2 ;;
    --skip-tests) run_tests=false; shift ;;
    --build-only) build_only=true; shift ;;
    --yes) assume_yes=true; shift ;;
    -h|--help) show_usage; exit 0 ;;
    *) fail "opção desconhecida: $1" ;;
  esac
done

required_commands=(awk cut docker dpkg dpkg-deb find go sed sha256sum sort)
if [[ "${build_only}" == false ]]; then
  required_commands+=(curl scp ssh)
fi
for command_name in "${required_commands[@]}"; do
  command -v "${command_name}" >/dev/null 2>&1 || fail "comando obrigatório não encontrado: ${command_name}"
done

mkdir -p "${dist_directory}"
latest_line="$(find "${dist_directory}" -maxdepth 1 -type f -name 'compasso-server_*_all.deb' -printf '%T@\t%p\n' | sort -nr | sed -n '1p')"
latest_package="${latest_line#*$'\t'}"
latest_version=""
if [[ -n "${latest_line}" && -f "${latest_package}" ]]; then
  latest_version="$(dpkg-deb --field "${latest_package}" Version)"
  echo "Último pacote gerado: $(basename "${latest_package}")"
  echo "Último piloto: ${latest_version}"
else
  echo "Nenhum pacote anterior do servidor foi encontrado em dist/."
fi

version_reference="${latest_version:-$(sed -n 's/^Version: //p' "${project_root}/packaging/debian/control")}"
version_prefix="${version_reference%%~pilot*}"
suggested_version="${version_reference}"
if [[ "${version_reference}" =~ ^(.+)~pilot([0-9]+)$ ]]; then
  version_prefix="${BASH_REMATCH[1]}"
  suggested_version="${version_prefix}~pilot$((10#${BASH_REMATCH[2]} + 1))"
fi

if [[ -z "${requested_version}" ]]; then
  requested_version="$(prompt_value "Novo piloto (número, nome pilotN ou versão completa)" "${suggested_version}")"
fi
case "${requested_version}" in
  '' ) fail "a versão não pode ficar vazia" ;;
  *[!0-9]* )
    if [[ "${requested_version}" == pilot* ]]; then
      package_version="${version_prefix}~${requested_version}"
    else
      package_version="${requested_version}"
    fi
    ;;
  * ) package_version="${version_prefix}~pilot${requested_version}" ;;
esac
dpkg --validate-version "${package_version}" || fail "versão Debian inválida: ${package_version}"
container_version="${package_version//\~/-}"
[[ "${container_version}" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]] || \
  fail "a versão não pode ser usada como tag Docker: ${container_version}"

if [[ "${build_only}" == false ]]; then
  if [[ "${assume_yes}" == true ]]; then
    [[ -n "${server_host}" ]] || fail "informe --host ao usar --yes"
    [[ -n "${ssh_user}" ]] || fail "informe --user ao usar --yes"
    if [[ -z "${health_url}" ]]; then
      health_url="http://${server_host}:8181/healthz"
    fi
  else
    if [[ -z "${server_host}" ]]; then
      server_host="$(prompt_value "Endereço ou nome do servidor" "")"
    fi
    ssh_user="$(prompt_value "Usuário SSH" "${ssh_user}")"
    ssh_port="$(prompt_value "Porta SSH" "${ssh_port}")"
    if [[ -z "${identity_file}" ]]; then
      identity_file="$(prompt_value "Chave SSH específica (vazio usa a configuração padrão)" "")"
    fi
    if [[ -z "${health_url}" ]]; then
      health_url="$(prompt_value "URL pública para confirmar a API" "http://${server_host}:8181/healthz")"
    fi
  fi

  [[ "${server_host}" =~ ^[A-Za-z0-9._-]+$ ]] || fail "endereço de servidor inválido"
  [[ "${ssh_user}" =~ ^[A-Za-z0-9._-]+$ ]] || fail "usuário SSH inválido"
  [[ "${ssh_port}" =~ ^[0-9]+$ ]] && ((ssh_port >= 1 && ssh_port <= 65535)) || fail "porta SSH inválida"
  if [[ -n "${identity_file}" && ! -r "${identity_file}" ]]; then
    fail "chave SSH não encontrada ou sem leitura: ${identity_file}"
  fi
fi

if command -v git >/dev/null 2>&1 && git -C "${project_root}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  branch_name="$(git -C "${project_root}" branch --show-current)"
  commit_name="$(git -C "${project_root}" rev-parse --short HEAD)"
  echo "Código: ${branch_name:-HEAD destacado} @ ${commit_name}"
  if [[ -n "$(git -C "${project_root}" status --porcelain)" ]]; then
    echo "Atenção: existem alterações locais; elas serão incluídas no pacote quando aplicável."
    git -C "${project_root}" status --short
    confirm "Publicar este estado local?" no || fail "publicação cancelada"
  fi
fi

echo
echo "Resumo da publicação"
echo "  Piloto Debian: ${package_version}"
echo "  Imagem Docker: ${container_version}"
if [[ "${build_only}" == false ]]; then
  echo "  Destino: ${ssh_user}@${server_host}:${ssh_port}"
  echo "  Healthcheck: ${health_url}"
  echo "  Backup dos dados: obrigatório, executado antes da atualização"
else
  echo "  Modo: somente gerar e validar"
fi
echo
confirm "Continuar?" yes || fail "operação cancelada"

if [[ "${run_tests}" == true ]]; then
  echo "Executando testes do servidor..."
  (cd "${project_root}" && go test ./server/... ./protocol/...)
fi

"${project_root}/scripts/build-server-package.sh" "${package_version}"
package_path="${dist_directory}/compasso-server_${package_version}_all.deb"
checksum_path="${package_path}.sha256"
"${project_root}/scripts/test-server-package.sh" "${package_path}"

echo
echo "Último arquivo gerado: ${package_path}"
echo "Piloto pronto: ${package_version}"
echo "Tamanho: $(du -h "${package_path}" | awk '{print $1}')"
echo "SHA-256: $(cut -d' ' -f1 "${checksum_path}")"

if [[ "${build_only}" == true ]]; then
  exit 0
fi

ssh_options=(-p "${ssh_port}")
scp_options=(-P "${ssh_port}")
if [[ -n "${identity_file}" ]]; then
  ssh_options+=(-i "${identity_file}")
  scp_options+=(-i "${identity_file}")
fi
target="${ssh_user}@${server_host}"
package_name="$(basename "${package_path}")"
checksum_name="$(basename "${checksum_path}")"
remote_package="/tmp/${package_name}"

echo
echo "Verificando acesso, ferramentas e espaço no servidor..."
ssh "${ssh_options[@]}" "${target}" 'set -eu
command -v sudo >/dev/null
command -v apt-get >/dev/null
command -v sha256sum >/dev/null
echo "Sistema: $(. /etc/os-release 2>/dev/null; echo "${PRETTY_NAME:-desconhecido}")"
echo "Espaço disponível para temporários: $(df -h /tmp | awk "NR==2 {print \$4}")"
docker_path=/var/lib/docker
if [ ! -e "${docker_path}" ]; then docker_path=/; fi
docker_free_kb="$(df -Pk "${docker_path}" | awk "NR==2 {print \$4}")"
echo "Espaço disponível para o Docker: $(df -h "${docker_path}" | awk "NR==2 {print \$4}")"
if [ "${docker_free_kb}" -lt 1048576 ]; then
  echo "erro: há menos de 1 GiB livre para reconstruir a imagem Docker" >&2
  exit 1
fi
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  echo "Docker Compose: disponível"
elif dpkg-query -W compasso-server >/dev/null 2>&1; then
  echo "erro: esta é uma atualização, mas Docker e Docker Compose não estão funcionando" >&2
  exit 1
else
  echo "Docker Compose: ausente; a primeira instalação oferecerá a instalação no Debian"
fi'

echo "Validando o sudo antes do envio (a senha pode ser solicitada)..."
ssh -tt "${ssh_options[@]}" "${target}" 'sudo -v'

installed_version="$(ssh "${ssh_options[@]}" "${target}" \
  "dpkg-query -W compasso-server 2>/dev/null | awk '{print \$2}' || true")"
deployment_script="install-server.sh"
downgrade_option=""
if [[ -n "${installed_version}" ]]; then
  deployment_script="update-server.sh"
  echo "Versão instalada: ${installed_version}"
  if dpkg --compare-versions "${package_version}" lt "${installed_version}"; then
    echo "Atenção: ${package_version} é anterior à versão instalada ${installed_version}."
    confirm "Autorizar downgrade?" no || fail "downgrade cancelado"
    downgrade_option="--allow-downgrades"
  elif dpkg --compare-versions "${package_version}" eq "${installed_version}"; then
    confirm "A mesma versão já está instalada. Reinstalar?" no || fail "reinstalação cancelada"
  fi
else
  echo "Primeira instalação detectada. O arquivo compasso.env será criado com os padrões do pacote."
  confirm "Prosseguir com a configuração inicial padrão?" no || fail "revise a configuração antes de instalar"
fi

echo "Enviando pacote e checksum..."
scp "${scp_options[@]}" "${package_path}" "${checksum_path}" "${target}:/tmp/"
ssh "${ssh_options[@]}" "${target}" "cd /tmp && sha256sum --check $(printf '%q' "${checksum_name}")"

deployment_started=true
echo "Instalando e colocando a API no ar; o sudo poderá pedir a senha novamente..."
ssh -tt "${ssh_options[@]}" "${target}" \
  "set -e; sudo env DEBIAN_FRONTEND=noninteractive apt-get -o Dpkg::Options::=--force-confold install --yes --reinstall ${downgrade_option} $(printf '%q' "${remote_package}"); sudo $(printf '%q' "/opt/compasso-server/scripts/${deployment_script}"); dpkg-query -W compasso-server"

echo "Confirmando o healthcheck a partir desta máquina..."
if ! curl --fail --silent --show-error --max-time 15 "${health_url}"; then
  echo >&2
  fail "a atualização interna terminou, mas ${health_url} não respondeu; verifique rede, firewall e proxy"
fi
echo

if confirm "Remover o pacote temporário do servidor?" yes; then
  ssh "${ssh_options[@]}" "${target}" \
    "rm -f $(printf '%q' "${remote_package}") $(printf '%q' "${remote_package}.sha256")"
fi

deployment_started=false
echo
echo "Servidor publicado com sucesso."
echo "  Piloto ativo: ${package_version}"
echo "  API verificada: ${health_url}"
