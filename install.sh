#!/bin/sh
# Installs the opentofu-kms-stackit binary from the GitHub releases page.
#
#   curl -fsSL https://raw.githubusercontent.com/stackitcloud/opentofu-kms-stackit/main/install.sh | sh
#
# Environment variables:
#   VERSION      release tag to install, e.g. v0.1.0 (default: latest)
#   INSTALL_DIR  target directory (default: /usr/local/bin)
#   REPO         source repository (default: stackitcloud/opentofu-kms-stackit)
set -eu

REPO="${REPO:-stackitcloud/opentofu-kms-stackit}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"
BINARY="opentofu-kms-stackit"

log() {
  echo "$*" >&2
}

fail() {
  log "error: $*"
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || fail "$1 is required but not installed"
}

# download URL FILE
download() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1" -o "$2" || fail "download failed: $1"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$2" "$1" || fail "download failed: $1"
  else
    fail "curl or wget is required but neither is installed"
  fi
}

detect_platform() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    linux | darwin) ;;
    *) fail "unsupported operating system: $os (linux and darwin have prebuilt binaries)" ;;
  esac

  arch=$(uname -m)
  case "$arch" in
    x86_64 | amd64) arch="amd64" ;;
    aarch64 | arm64) arch="arm64" ;;
    *) fail "unsupported architecture: $arch (amd64 and arm64 have prebuilt binaries)" ;;
  esac
}

# Follows the /releases/latest redirect, which ends in /tag/<version>.
resolve_latest() {
  if command -v curl >/dev/null 2>&1; then
    _url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest") ||
      fail "could not resolve the latest release of ${REPO}"
  else
    need wget
    _url=$(wget --max-redirect=10 -qS --spider "https://github.com/${REPO}/releases/latest" 2>&1 |
      awk '/[Ll]ocation:/ { print $2 }' | tail -n 1) ||
      fail "could not resolve the latest release of ${REPO}"
  fi

  _tag="${_url##*/tag/}"
  case "$_tag" in
    "" | *"/"*) fail "could not read a release tag from ${_url}" ;;
  esac
  echo "$_tag"
}

# Function variables are prefixed with _ because POSIX sh has no locals.
verify_checksum() {
  _archive="$1"
  _checksums="$2"
  _name=$(basename "$_archive")

  # An empty line would make the check below pass without verifying anything.
  _expected=$(grep " ${_name}$" "$_checksums" || true)
  [ -n "$_expected" ] || fail "checksums.txt has no entry for $_name"

  if command -v sha256sum >/dev/null 2>&1; then
    _check="sha256sum -c -"
  elif command -v shasum >/dev/null 2>&1; then
    _check="shasum -a 256 -c -"
  else
    log "warning: neither sha256sum nor shasum found, skipping checksum verification"
    return
  fi

  (cd "$(dirname "$_archive")" && echo "$_expected" | $_check) >/dev/null ||
    fail "checksum verification failed for $_name"
  log "checksum ok"
}

# Uses sudo only when the target directory is not writable.
install_binary() {
  _src="$1"

  if [ -d "$INSTALL_DIR" ] && [ -w "$INSTALL_DIR" ]; then
    install -m 0755 "$_src" "${INSTALL_DIR}/${BINARY}"
  elif command -v sudo >/dev/null 2>&1; then
    log "${INSTALL_DIR} is not writable, using sudo"
    sudo install -d -m 0755 "$INSTALL_DIR"
    sudo install -m 0755 "$_src" "${INSTALL_DIR}/${BINARY}"
  else
    fail "${INSTALL_DIR} is not writable and sudo is not available (set INSTALL_DIR to a writable path)"
  fi
}

main() {
  need tar
  need uname
  need grep
  detect_platform

  if [ "$VERSION" = "latest" ]; then
    tag=$(resolve_latest)
  else
    tag="$VERSION"
  fi
  # Tags carry a leading v, archive names do not.
  version="${tag#v}"
  tag="v${version}"

  archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
  base_url="https://github.com/${REPO}/releases/download/${tag}"

  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT INT TERM

  log "downloading ${archive} (${tag})"
  download "${base_url}/${archive}" "${tmp}/${archive}"
  download "${base_url}/checksums.txt" "${tmp}/checksums.txt"
  verify_checksum "${tmp}/${archive}" "${tmp}/checksums.txt"

  (cd "$tmp" && tar -xzf "$archive" "$BINARY") || fail "could not extract ${BINARY} from ${archive}"
  install_binary "${tmp}/${BINARY}"

  log "installed ${BINARY} ${tag} to ${INSTALL_DIR}/${BINARY}"
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *) log "note: ${INSTALL_DIR} is not on your PATH" ;;
  esac
}

main "$@"
