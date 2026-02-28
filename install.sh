#!/usr/bin/env bash
set -e

INSTALL_DIR="$HOME/bin"
BINARY="$INSTALL_DIR/bt"

mkdir -p "$INSTALL_DIR"
go build -o "$BINARY" ./cmd/bt/main.go
echo "Installed: $BINARY"
