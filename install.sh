#!/bin/sh
# dockup bootstrap — downloads the latest dockup release, verifies its
# SHA-256 checksum and launches it. All install logic lives in dockup itself.
#
#   curl -fsSL https://raw.githubusercontent.com/paulo-amaral/dockup/master/install.sh | sh
set -eu

REPO="paulo-amaral/dockup"
PROJECT="dockup"

say() { printf '\033[1;33m[dockup]\033[0m %s\n' "$1"; }
die() { printf '\033[1;31m[dockup]\033[0m %s\n' "$1" >&2; exit 1; }

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
    x86_64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) die "unsupported architecture: $arch" ;;
esac
case "$os" in
    linux) ;;
    darwin) [ "$arch" = arm64 ] || die "on Intel Macs use Docker Desktop; dockup supports Apple Silicon only" ;;
    *) die "unsupported OS: $os (dockup targets Linux and macOS)" ;;
esac

command -v curl >/dev/null 2>&1 || die "curl is required"

say "resolving latest release..."
tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
    sed -n 's/.*"tag_name"[^"]*"\([^"]*\)".*/\1/p' | head -n1)
[ -n "$tag" ] || die "could not resolve the latest release tag"
version=${tag#v}

archive="${PROJECT}_${version}_${os}_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/download/${tag}"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

say "downloading ${PROJECT} ${tag} (${os}/${arch})..."
curl -fsSL -o "${tmp}/${archive}" "${base}/${archive}"
curl -fsSL -o "${tmp}/checksums.txt" "${base}/checksums.txt"

say "verifying SHA-256 checksum..."
expected=$(grep " ${archive}\$" "${tmp}/checksums.txt" | awk '{print $1}')
[ -n "$expected" ] || die "checksum for ${archive} not found in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "${tmp}/${archive}" | awk '{print $1}')
else
    actual=$(shasum -a 256 "${tmp}/${archive}" | awk '{print $1}')
fi
[ "$expected" = "$actual" ] || die "checksum mismatch — aborting (expected ${expected}, got ${actual})"

tar -xzf "${tmp}/${archive}" -C "$tmp"

dest="/usr/local/bin"
if [ -w "$dest" ]; then
    install -m 0755 "${tmp}/${PROJECT}" "${dest}/${PROJECT}"
elif command -v sudo >/dev/null 2>&1; then
    say "installing to ${dest} (sudo may prompt for your password)..."
    sudo install -m 0755 "${tmp}/${PROJECT}" "${dest}/${PROJECT}"
else
    dest="${HOME}/.local/bin"
    mkdir -p "$dest"
    install -m 0755 "${tmp}/${PROJECT}" "${dest}/${PROJECT}"
    say "installed to ${dest}; make sure it is on your PATH"
fi

say "installed: $("${dest}/${PROJECT}" --version)"
say "launching TUI (run '${PROJECT}' anytime; use 'sudo ${PROJECT}' on Linux to install)"
exec "${dest}/${PROJECT}"
