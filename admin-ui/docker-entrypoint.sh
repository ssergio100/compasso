#!/bin/sh
set -eu

: "${COMPASSO_API_BASE_URL:=auto}"
: "${COMPASSO_API_PORT:=8181}"

case "$COMPASSO_API_BASE_URL" in
  auto) COMPASSO_API_CONNECT_SOURCE="http: https:" ;;
  http://*|https://*) COMPASSO_API_CONNECT_SOURCE="$COMPASSO_API_BASE_URL" ;;
  *) echo "COMPASSO_API_BASE_URL deve ser 'auto' ou uma URL HTTP absoluta" >&2; exit 1 ;;
esac
case "$COMPASSO_API_PORT" in
  *[!0-9]*|'') echo "COMPASSO_API_PORT deve ser numérica" >&2; exit 1 ;;
esac
export COMPASSO_API_CONNECT_SOURCE

envsubst '${COMPASSO_API_BASE_URL} ${COMPASSO_API_PORT}' \
  < /opt/compasso/runtime-config.template.js \
  > /tmp/runtime-config.js
envsubst '${COMPASSO_API_CONNECT_SOURCE}' \
  < /opt/compasso/nginx.conf.template \
  > /tmp/nginx.conf

exec nginx -c /tmp/nginx.conf -g 'daemon off;'
