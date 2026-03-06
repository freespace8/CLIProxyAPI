#!/usr/bin/env bash
set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly BINARY_PATH="${SCRIPT_DIR}/cli-proxy-api"
readonly CONFIG_PATH="${SCRIPT_DIR}/config.yaml"
readonly AUTH_DIR="${SCRIPT_DIR}/auths"

authorize_runtime() {
  if [[ ! -x "${BINARY_PATH}" ]]; then
    echo "Error: binary not found: ${BINARY_PATH}" >&2
    exit 1
  fi

  if [[ ! -f "${CONFIG_PATH}" ]]; then
    echo "Error: config file not found: ${CONFIG_PATH}" >&2
    exit 1
  fi
}

prepare_runtime_files() {
  mkdir -p "${AUTH_DIR}"
}

main() {
  authorize_runtime
  prepare_runtime_files
  exec "${BINARY_PATH}" -config "${CONFIG_PATH}" "$@"
}

main "$@"
