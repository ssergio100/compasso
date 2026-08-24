#!/usr/bin/env bash
set -euo pipefail

DEST="sergio@192.168.18.10"
PORT=22
PACKAGE="dist/compasso-server_0.1.0~pilot26_all.deb"

# 1. Enviar o pacote para o outro PC
scp -P "${PORT}" "${PACKAGE}" "${DEST}:/tmp/"

# 2. Instalar (--reinstall porque é a mesma versão) + atualizar (recompila e reinicia)
ssh -p "${PORT}" "${DEST}" \
  "sudo apt install --reinstall /tmp/${PACKAGE##*/} && sudo /opt/compasso-server/scripts/update-server.sh"
