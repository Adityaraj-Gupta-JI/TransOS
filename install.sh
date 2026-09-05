#!/usr/bin/env bash
set -e

REPO="https://github.com/Adityaraj-Gupta-JI/TransOS"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="transos"

echo "[+] Installing trans-os..."

# Check Go installation or fetch pre-built binary
if command -v go >/dev/null 2>&1; then
    echo "[+] Go detected. Compiling from source..."
    go build -o "$BINARY_NAME" cmd/transos/main.go
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
else
    echo "[-] Go is not installed. Fetching pre-compiled release binary..."
    # Downloads binary based on OS/Arch
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"
    if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi

    URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}_${OS}_${ARCH}"
    curl -fsSL "$URL" -o "$BINARY_NAME"
    chmod +x "$BINARY_NAME"
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
fi

echo "[✓] Installation complete! Type 'transos' to start."