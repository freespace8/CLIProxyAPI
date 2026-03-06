#!/usr/bin/env bash
set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly WEB_DIR="${SCRIPT_DIR}/web"
readonly BIN_DIR="${SCRIPT_DIR}/bin/linux-amd64"
readonly BINARY_PATH="${BIN_DIR}/cli-proxy-api"
readonly PUBLISH_DIR="${SCRIPT_DIR}/publish"
readonly RELEASE_DIR="${PUBLISH_DIR}/cli-proxy-api-linux-amd64"
readonly ARCHIVE_PATH="${PUBLISH_DIR}/cli-proxy-api-linux-amd64.tar.gz"
readonly DEFAULT_CONFIG="${SCRIPT_DIR}/config.example.yaml"
readonly STATIC_DIR="${SCRIPT_DIR}/static"
readonly AUTH_DIR="${RELEASE_DIR}/auths"
readonly RUNTIME_SCRIPT="${RELEASE_DIR}/run.sh"
readonly FILELIST_PATH="${RELEASE_DIR}/FILELIST.txt"
readonly TARGET_GOOS="linux"
readonly TARGET_GOARCH="amd64"
readonly GO_BUILD_FLAGS=("-trimpath")
readonly GO_LDFLAGS="-s -w -X 'main.Version=%s' -X 'main.Commit=%s' -X 'main.BuildDate=%s'"

require_command() {
  local command_name="$1"
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "Error: ${command_name} is required" >&2
    exit 1
  fi
}

build_web() {
  if [[ ! -f "${WEB_DIR}/package.json" ]]; then
    echo "Error: ${WEB_DIR}/package.json not found" >&2
    exit 1
  fi

  require_command npm
  if [[ -f "${WEB_DIR}/package-lock.json" ]]; then
    npm --prefix "${WEB_DIR}" ci
    npm --prefix "${WEB_DIR}" run build
    return
  fi

  npm --prefix "${WEB_DIR}" install
  npm --prefix "${WEB_DIR}" run build
}

build_binary() {
  local version
  local commit
  local build_date

  require_command go
  version="$(git -C "${SCRIPT_DIR}" describe --tags --always --dirty 2>/dev/null || echo dev)"
  commit="$(git -C "${SCRIPT_DIR}" rev-parse --short HEAD 2>/dev/null || echo none)"
  build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  mkdir -p "${BIN_DIR}"
  printf 'Building %s\n' "${BINARY_PATH}"
  printf '  GOOS: %s\n' "${TARGET_GOOS}"
  printf '  GOARCH: %s\n' "${TARGET_GOARCH}"
  printf '  Version: %s\n' "${version}"
  printf '  Commit: %s\n' "${commit}"
  printf '  Build Date: %s\n' "${build_date}"

  CGO_ENABLED=0 GOOS="${TARGET_GOOS}" GOARCH="${TARGET_GOARCH}" go build \
    "${GO_BUILD_FLAGS[@]}" \
    -ldflags="$(printf -- "${GO_LDFLAGS}" "${version}" "${commit}" "${build_date}")" \
    -o "${BINARY_PATH}" \
    ./cmd/server/
}

write_runtime_script() {
  cat > "${RUNTIME_SCRIPT}" <<'SCRIPT'
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
SCRIPT

  chmod +x "${RUNTIME_SCRIPT}"
}

prepare_release_dir() {
  rm -rf "${RELEASE_DIR}" "${ARCHIVE_PATH}"
  mkdir -p "${AUTH_DIR}"

  cp "${BINARY_PATH}" "${RELEASE_DIR}/cli-proxy-api"
  cp "${DEFAULT_CONFIG}" "${RELEASE_DIR}/config.yaml"
  touch "${AUTH_DIR}/.gitkeep"
  if [[ -d "${STATIC_DIR}" ]]; then
    cp -R "${STATIC_DIR}" "${RELEASE_DIR}/static"
  fi
  write_runtime_script
}

generate_filelist() {
  (
    cd "${RELEASE_DIR}"
    find . \( -type f -o -type l \) | sort | sed 's#^\./##' > "${FILELIST_PATH}"
  )
}

package_release() {
  mkdir -p "${PUBLISH_DIR}"
  tar -czf "${ARCHIVE_PATH}" -C "${PUBLISH_DIR}" "$(basename "${RELEASE_DIR}")"
}

main() {
  build_web
  build_binary
  prepare_release_dir
  generate_filelist
  package_release

  echo "Publish directory ready: ${RELEASE_DIR}"
  echo "Archive ready: ${ARCHIVE_PATH}"
  echo "File list: ${FILELIST_PATH}"
}

main "$@"
