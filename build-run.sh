#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${SCRIPT_DIR}/bin"
BINARY="${BIN_DIR}/cli-proxy-api"
DEFAULT_CONFIG="${SCRIPT_DIR}/config.yaml"
EXAMPLE_CONFIG="${SCRIPT_DIR}/config.example.yaml"
WEB_DIR="${SCRIPT_DIR}/web"

BUILD_ONLY=false
CONFIG_PATH=""
declare -a RUN_ARGS=()

while (($# > 0)); do
  case "$1" in
    --build-only)
      BUILD_ONLY=true
      ;;
    -config)
      if (($# < 2)); then
        echo "Error: -config requires a file path" >&2
        exit 1
      fi
      CONFIG_PATH="$2"
      RUN_ARGS+=("$1" "$2")
      shift
      ;;
    -config=*)
      CONFIG_PATH="${1#*=}"
      RUN_ARGS+=("$1")
      ;;
    *)
      RUN_ARGS+=("$1")
      ;;
  esac
  shift
done

if [[ ! -f "${WEB_DIR}/package.json" ]]; then
  echo "Error: ${WEB_DIR}/package.json not found" >&2
  exit 1
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "Error: npm is required to build ${WEB_DIR}" >&2
  exit 1
fi

if [[ "${BUILD_ONLY}" != "true" ]]; then
  if [[ -z "${CONFIG_PATH}" ]]; then
    CONFIG_PATH="${DEFAULT_CONFIG}"
    if ((${#RUN_ARGS[@]} > 0)); then
      RUN_ARGS=(-config "${CONFIG_PATH}" "${RUN_ARGS[@]}")
    else
      RUN_ARGS=(-config "${CONFIG_PATH}")
    fi

    if [[ ! -f "${CONFIG_PATH}" ]]; then
      if [[ ! -f "${EXAMPLE_CONFIG}" ]]; then
        echo "Error: ${EXAMPLE_CONFIG} not found" >&2
        exit 1
      fi

      cp "${EXAMPLE_CONFIG}" "${CONFIG_PATH}"
      echo "Created default config at ${CONFIG_PATH} from config.example.yaml"
      echo "Edit config.yaml before real use if you need custom API keys, providers, or ports"
    fi
  elif [[ ! -f "${CONFIG_PATH}" ]]; then
    echo "Error: config file not found: ${CONFIG_PATH}" >&2
    exit 1
  fi
fi

mkdir -p "${BIN_DIR}"

if [[ -f "${WEB_DIR}/package-lock.json" ]]; then
  npm --prefix "${WEB_DIR}" ci
else
  npm --prefix "${WEB_DIR}" install
fi

npm --prefix "${WEB_DIR}" run build

VERSION="$(git -C "${SCRIPT_DIR}" describe --tags --always --dirty 2>/dev/null || echo dev)"
COMMIT="$(git -C "${SCRIPT_DIR}" rev-parse --short HEAD 2>/dev/null || echo none)"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

printf 'Building %s\n' "${BINARY}"
printf '  Version: %s\n' "${VERSION}"
printf '  Commit: %s\n' "${COMMIT}"
printf '  Build Date: %s\n' "${BUILD_DATE}"

CGO_ENABLED=0 go build \
  -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" \
  -o "${BINARY}" \
  ./cmd/server/

if [[ "${BUILD_ONLY}" == "true" ]]; then
  echo "Build finished: ${BINARY}"
  exit 0
fi

echo "Starting ${BINARY} ${RUN_ARGS[*]}"
exec "${BINARY}" "${RUN_ARGS[@]}"
