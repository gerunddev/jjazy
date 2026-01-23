#!/bin/bash
#
# jjazy installer script
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/gerunddev/jjazy/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/gerunddev/jjazy/main/install.sh | bash -s v0.1.0
#

set -euo pipefail

REPO="gerunddev/jjazy"
INSTALL_DIR="${JJAZY_INSTALL_DIR:-$HOME/.local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}==>${NC} $1"
}

warn() {
    echo -e "${YELLOW}Warning:${NC} $1"
}

error() {
    echo -e "${RED}Error:${NC} $1" >&2
    exit 1
}

# Detect OS
detect_os() {
    local os
    os=$(uname -s)
    case "$os" in
        Darwin) echo "darwin" ;;
        Linux) echo "linux" ;;
        *) error "Unsupported operating system: $os" ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
}

# Get the latest release version from GitHub
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
    if [ -z "$version" ]; then
        error "Failed to fetch latest version"
    fi
    echo "$version"
}

# Download and verify checksums
download_and_verify() {
    local version="$1"
    local os="$2"
    local arch="$3"
    local archive="jjazy_${os}_${arch}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${version}/${archive}"
    local checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    local tmpdir
    tmpdir=$(mktemp -d)
    trap "rm -rf '$tmpdir'" EXIT

    info "Downloading ${archive}..."
    if ! curl -fsSL -o "${tmpdir}/${archive}" "$url"; then
        error "Failed to download ${url}"
    fi

    info "Downloading checksums..."
    if ! curl -fsSL -o "${tmpdir}/checksums.txt" "$checksums_url"; then
        error "Failed to download checksums"
    fi

    info "Verifying checksum..."
    local expected_checksum
    expected_checksum=$(grep -F "${archive}" "${tmpdir}/checksums.txt" | awk '{print $1}')
    if [ -z "$expected_checksum" ]; then
        error "Checksum not found for ${archive}"
    fi

    local actual_checksum
    if command -v sha256sum &> /dev/null; then
        actual_checksum=$(sha256sum "${tmpdir}/${archive}" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        actual_checksum=$(shasum -a 256 "${tmpdir}/${archive}" | awk '{print $1}')
    else
        error "Neither sha256sum nor shasum found"
    fi

    if [ "$expected_checksum" != "$actual_checksum" ]; then
        error "Checksum verification failed!\nExpected: ${expected_checksum}\nActual: ${actual_checksum}"
    fi
    info "Checksum verified!"

    info "Extracting..."
    tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}"

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        info "Creating ${INSTALL_DIR}..."
        mkdir -p "$INSTALL_DIR"
    fi

    info "Installing to ${INSTALL_DIR}/jjazy..."
    mv "${tmpdir}/jjazy" "${INSTALL_DIR}/jjazy"
    chmod +x "${INSTALL_DIR}/jjazy"
}

# Check if install directory is in PATH
check_path() {
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        warn "${INSTALL_DIR} is not in your PATH"
        echo ""
        echo "Add this to your shell configuration (~/.bashrc, ~/.zshrc, etc.):"
        echo ""
        echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
    fi
}

main() {
    local version="${1:-}"
    local os
    local arch

    os=$(detect_os)
    arch=$(detect_arch)

    info "Detected: ${os}/${arch}"

    if [ -z "$version" ]; then
        info "Fetching latest version..."
        version=$(get_latest_version)
    fi

    info "Installing jjazy ${version}..."

    download_and_verify "$version" "$os" "$arch"

    echo ""
    info "jjazy ${version} installed successfully!"
    echo ""

    check_path

    # Verify installation
    if command -v jjazy &> /dev/null || [ -x "${INSTALL_DIR}/jjazy" ]; then
        echo "Run 'jjazy --version' to verify the installation."
        echo "Run 'jjazy' in a jj repository to start."
    fi
}

main "$@"
