#!/usr/bin/env bash
set -euo pipefail

BINARY_NAME=bubblecode
REPO=gausszhou/bubblecode
DEFAULT_INSTALL_DIR="$HOME/.local/bin"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { printf "${GREEN}%s${NC}\n" "$*"; }
warn()  { printf "${YELLOW}WARNING: %s${NC}\n" "$*"; }
error() { printf "${RED}ERROR: %s${NC}\n" "$*"; exit 1; }

detect_os() {
  case "$(uname -s)" in
    Linux)                      echo "linux" ;;
    Darwin)                     echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*)       echo "windows" ;;
    *)                          error "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *)             error "unsupported architecture: $(uname -m)" ;;
  esac
}

get_latest_version() {
  curl -sfL "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": "\(.*\)",.*/\1/'
}

main() {
  local version="${1:-}"
  local os arch install_dir install_path

  os="$(detect_os)"
  arch="$(detect_arch)"

  if [ -z "$version" ]; then
    info "detecting latest version..."
    version="$(get_latest_version)"
    [ -z "$version" ] && error "failed to detect latest version from GitHub"
  fi

  version="${version#v}"

  if [ "$os" = "windows" ]; then
    install_dir="${INSTALL_DIR:-$HOME/bin}"
    install_path="$install_dir/$BINARY_NAME.exe"
  else
    install_dir="${INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
    install_path="$install_dir/$BINARY_NAME"
  fi

  local tarball="${BINARY_NAME}-${os}-${arch}.tar.gz"
  local url="https://github.com/$REPO/releases/download/v${version}/${tarball}"

  info "detected: $os/$arch"
  info "version:  v$version"
  info "target:   $install_path"



  info "downloading $url ..."
  curl -sfL "$url" -o "/tmp/$tarball"
  trap 'rm -f /tmp/$tarball' EXIT

  mkdir -p "$install_dir"
  info "extracting..."
  tar xzf "/tmp/$tarball" -C "$install_dir"
  chmod +x "$install_path"

  info "$BINARY_NAME v$version installed to $install_path"

  if ! command -v opencode &>/dev/null; then
    warn "'opencode' not found in PATH — $BINARY_NAME requires it at runtime"
    warn "install from: https://opencode.ai"
  fi

  if [ "$os" = "windows" ]; then
    warn "$install_dir may not be in your PATH — add this to your ~/.bashrc:"
    warn "  export PATH=\"\$PATH:$install_dir\""
  fi
}

main "$@"
