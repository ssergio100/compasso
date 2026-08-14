#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dockerfile_path="${project_root}/packaging/client/Dockerfile.build"
temporary_output_directory="$(mktemp -d)"
builder_image_name="compasso-client-binaries:local"
temporary_container_id=""

cleanup_temporary_output() {
  if [[ -n "${temporary_container_id}" ]]; then
    docker rm --force "${temporary_container_id}" >/dev/null 2>&1 || true
  fi
  rm -rf "${temporary_output_directory}"
}
trap cleanup_temporary_output EXIT

docker build \
  --file "${dockerfile_path}" \
  --target exported-binaries \
  --tag "${builder_image_name}" \
  "${project_root}"

temporary_container_id="$(docker create "${builder_image_name}" /tempo-agent)"

binary_names=(tempo-agent tempo-agent-configure)
for binary_name in "${binary_names[@]}"; do
  docker cp \
    "${temporary_container_id}:/${binary_name}" \
    "${temporary_output_directory}/${binary_name}"

  binary_path="${temporary_output_directory}/${binary_name}"

  if [[ ! -x "${binary_path}" ]]; then
    echo "erro: o build não gerou ${binary_name}" >&2
    exit 1
  fi

  if ldd "${binary_path}" | grep -Fq "libgo"; then
    echo "erro: ${binary_name} depende de libgo" >&2
    exit 1
  fi

  if ldd "${binary_path}" | grep -Fq "not found"; then
    echo "erro: ${binary_name} possui dependência não encontrada" >&2
    ldd "${binary_path}" >&2
    exit 1
  fi
done

install -d "${project_root}/bin"
for binary_name in "${binary_names[@]}"; do
  install -m 0755 \
    "${temporary_output_directory}/${binary_name}" \
    "${project_root}/bin/${binary_name}"
done

echo "Binários portáteis criados em ${project_root}/bin."
