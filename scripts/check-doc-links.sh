#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
invalid_links=0

while IFS=$'\t' read -r source_file target; do
  target="${target%%#*}"
  [[ -n "${target}" ]] || continue
  case "${target}" in
    http://*|https://*|mailto:*) continue ;;
  esac

  if [[ "${target}" == /* ]]; then
    candidate="${project_root}${target}"
  else
    candidate="$(dirname "${source_file}")/${target}"
  fi
  if [[ ! -e "${candidate}" ]]; then
    echo "link local inválido: ${source_file#"${project_root}/"} -> ${target}" >&2
    invalid_links=1
  fi
done < <(
  find "${project_root}" -type f -name '*.md' \
    -not -path "${project_root}/.git/*" \
    -not -path "${project_root}/.private/*" \
    -not -path "${project_root}/admin-ui/node_modules/*" \
    -print0 \
    | xargs -0 -r perl -ne 'while (/\[[^]]*\]\(([^ )]+)(?:\s+"[^"]*")?\)/g) { print "$ARGV\t$1\n" }'
)

if [[ "${invalid_links}" -ne 0 ]]; then
  exit 1
fi

echo "links locais da documentação validados"
