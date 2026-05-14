#!/usr/bin/env bash
# deploy.sh — build vibespace-server and ship it to vibespace.sh
# Usage: scripts/deploy.sh

set -euo pipefail

REMOTE="root@vibespace.sh"
REMOTE_PORT="22022"
REMOTE_PATH="/usr/local/bin/vibespace-server"
SERVICE="vibespace"
GOOS="linux"
GOARCH="amd64"

info()  { printf "\033[32m>> %s\033[0m\n" "$*"; }
error() { printf "\033[31m!! %s\033[0m\n" "$*" >&2; }

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

command -v go  >/dev/null || { error "go not found"; exit 1; }
command -v ssh >/dev/null || { error "ssh not found"; exit 1; }
command -v scp >/dev/null || { error "scp not found"; exit 1; }

VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$BUILD_DIR"' EXIT

info "Building vibespace-server $VERSION for $GOOS/$GOARCH ..."
GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 \
  go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
  -o "$BUILD_DIR/vibespace-server" ./cmd/server

info "Uploading to $REMOTE:$REMOTE_PATH.new ..."
scp -P "$REMOTE_PORT" "$BUILD_DIR/vibespace-server" "$REMOTE:$REMOTE_PATH.new"

info "Swapping in new binary and restarting $SERVICE ..."
ssh -p "$REMOTE_PORT" "$REMOTE" "mv $REMOTE_PATH.new $REMOTE_PATH && systemctl restart $SERVICE"

info "Deployed $VERSION to $REMOTE"
