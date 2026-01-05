#!/usr/bin/env bash
# Download gotty binaries for web terminal functionality
# Source: https://github.com/sorenisanerd/gotty

set -e

GOTTY_VERSION="v1.5.0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

download_gotty() {
    local arch="$1"
    local url="$2"
    local output="${SCRIPT_DIR}/../runtime/gotty-${arch}"

    if [[ -f "$output" ]]; then
        echo "gotty-${arch} already exists, skipping download"
        return 0
    fi

    echo "Downloading gotty ${GOTTY_VERSION} for ${arch}..."

    # Download and extract
    local tmpdir=$(mktemp -d)
    trap "rm -rf $tmpdir" EXIT

    curl -sL "$url" -o "${tmpdir}/gotty.tar.gz"
    tar -xzf "${tmpdir}/gotty.tar.gz" -C "$tmpdir"
    mv "${tmpdir}/gotty" "$output"
    chmod +x "$output"

    echo "Downloaded gotty-${arch} successfully"
}

# Download for current architecture only, or both if --all flag
if [[ "$1" == "--all" ]]; then
    download_gotty "aarch64" "https://github.com/sorenisanerd/gotty/releases/download/${GOTTY_VERSION}/gotty_${GOTTY_VERSION}_linux_arm64.tar.gz"
    download_gotty "x86_64" "https://github.com/sorenisanerd/gotty/releases/download/${GOTTY_VERSION}/gotty_${GOTTY_VERSION}_linux_amd64.tar.gz"
else
    ARCH=$(uname -m)
    case "${ARCH}" in
        aarch64)
            download_gotty "aarch64" "https://github.com/sorenisanerd/gotty/releases/download/${GOTTY_VERSION}/gotty_${GOTTY_VERSION}_linux_arm64.tar.gz"
            ;;
        x86_64)
            download_gotty "x86_64" "https://github.com/sorenisanerd/gotty/releases/download/${GOTTY_VERSION}/gotty_${GOTTY_VERSION}_linux_amd64.tar.gz"
            ;;
        *)
            echo "Unsupported architecture: ${ARCH}"
            exit 1
            ;;
    esac
fi

echo "Done! gotty binaries are ready."
