#!/usr/bin/env bash

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$PLUGIN_DIR/bin/tasm"
REPO="JonSchaeffer/tmux-agent-session-manager"

# Already present — nothing to do.
[[ -x "$BIN" ]] && exit 0
command -v tasm &>/dev/null && exit 0

mkdir -p "$PLUGIN_DIR/bin"

# Detect OS and arch.
_os="$(uname -s)"
case "$_os" in
  Darwin) os="darwin" ;;
  Linux)  os="linux"  ;;
  *)
    echo "tasm: unsupported OS $_os — attempting build from source"
    os=""
    ;;
esac

_arch="$(uname -m)"
case "$_arch" in
  arm64|aarch64) arch="arm64" ;;
  x86_64)        arch="amd64" ;;
  *)
    echo "tasm: unsupported arch $_arch — attempting build from source"
    arch=""
    ;;
esac

# Attempt prebuilt download.
if [[ -n "$os" && -n "$arch" ]]; then
  url="https://github.com/${REPO}/releases/latest/download/tasm-${os}-${arch}"
  if curl -fsSL "$url" -o "$BIN" 2>/dev/null; then
    chmod +x "$BIN"
    if "$BIN" version &>/dev/null; then
      exit 0
    fi
    rm -f "$BIN"
  fi
fi

# Source fallback: build with Go if available.
if command -v go &>/dev/null; then
  make -C "$PLUGIN_DIR" build 2>/dev/null && exit 0
fi

echo "tasm: could not download a prebuilt binary or build from source. Install Go, or download manually from the releases page."
exit 1
