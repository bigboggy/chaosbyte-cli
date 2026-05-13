#!/usr/bin/env bash
# install.sh — install vibespace to $HOME/.local/bin
# Usage: curl -fsSL https://raw.githubusercontent.com/bigboggy/vibespace-cli/main/install.sh | bash
#        bash install.sh --uninstall

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────
REPO="bigboggy/vibespace-cli"
INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="vibespace"
VERSION="${VERSION:-latest}"

# ── Helpers ───────────────────────────────────────────────────────────────────
info()  { printf "\033[32m>> %s\033[0m\n" "$*"; }
warn()  { printf "\033[33m!! %s\033[0m\n" "$*"; }
error() { printf "\033[31m!! %s\033[0m\n" "$*" >&2; }

# ── Platform detection ────────────────────────────────────────────────────────
detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *)       echo "unknown" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *)       echo "unknown" ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

# ── Uninstall ─────────────────────────────────────────────────────────────────
if [[ "${1:-}" == "--uninstall" ]]; then
  BINARY="$INSTALL_DIR/$BINARY_NAME"
  if [[ -f "$BINARY" ]]; then
    info "Uninstalling $BINARY_NAME from $BINARY ..."
    rm "$BINARY"
    info "Done. You may want to remove $INSTALL_DIR from PATH if it was added by this script."
  else
    warn "$BINARY_NAME not found in $INSTALL_DIR"
  fi
  exit 0
fi

# ── Pre-flight ────────────────────────────────────────────────────────────────
if [[ "$OS" == "unknown" ]]; then
  error "Unsupported OS: $(uname -s)"
  error "This installer supports Linux and macOS only."
  exit 1
fi

if [[ "$ARCH" == "unknown" ]]; then
  error "Unsupported architecture: $(uname -m)"
  exit 1
fi

# ── Resolve version ───────────────────────────────────────────────────────────
if [[ "$VERSION" == "latest" ]]; then
  info "Fetching latest release info ..."
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')"
  info "Latest release: $VERSION"
fi

# ── Download ──────────────────────────────────────────────────────────────────
if [[ "$OS" == "darwin" ]]; then
  ASSET="${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
else
  ASSET="${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
fi

URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET"

info "Downloading $ASSET ..."
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

if ! curl -fsSL --retry 3 -o "$TMPDIR/$ASSET" "$URL"; then
  error "Failed to download $URL"
  error "Check that release $VERSION exists for $OS/$ARCH."
  exit 1
fi

# ── Install ───────────────────────────────────────────────────────────────────
mkdir -p "$INSTALL_DIR"

info "Extracting ..."
tar -xzf "$TMPDIR/$ASSET" -C "$TMPDIR"

# Find the binary in the archive (may be named gitstatus, gitstatus-darwin-arm64, etc.)
BINARY_PATH=""
BINARY_PATH="$(find "$TMPDIR" -maxdepth 1 -type f -not -name '*.tar.gz' -not -name '*.zip' \( -name "${BINARY_NAME}" -o -name "${BINARY_NAME}-*" \) | head -1)"
[[ -z "$BINARY_PATH" ]] && BINARY_PATH="$(find "$TMPDIR" -maxdepth 1 -type f -executable | head -1)"

if [[ -z "$BINARY_PATH" ]]; then
  error "Could not find $BINARY_NAME in the archive"
  exit 1
fi

info "Installing to $INSTALL_DIR/$BINARY_NAME ..."
cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# ── PATH check ────────────────────────────────────────────────────────────────
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  warn "$INSTALL_DIR is not in PATH."
  warn "Add this to your shell config (~/.bashrc, ~/.zshrc, etc.):"
  warn "  export PATH=\"$INSTALL_DIR:\$PATH\""
else
  info "$INSTALL_DIR is already in PATH."
fi

info "$BINARY_NAME installed (version $VERSION)"
info "Run '$BINARY_NAME' to get started!"
