#!/usr/bin/env bash
# Hourgit installer — https://hourgit.com
# Usage: curl -fsSL https://hourgit.com/install.sh | bash
set -euo pipefail

# ── Colors ──────────────────────────────────────────────────────────────────

if [ -t 1 ]; then
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  YELLOW='\033[0;33m'
  BLUE='\033[0;34m'
  BOLD='\033[1m'
  RESET='\033[0m'
else
  RED='' GREEN='' YELLOW='' BLUE='' BOLD='' RESET=''
fi

info()  { printf "${BLUE}info${RESET}  %s\n" "$*"; }
warn()  { printf "${YELLOW}warn${RESET}  %s\n" "$*"; }
error() { printf "${RED}error${RESET} %s\n" "$*" >&2; }
bold()  { printf "${BOLD}%s${RESET}" "$*"; }

# ── Prereqs ─────────────────────────────────────────────────────────────────

require_cmd() {
  if ! command -v "$1" &>/dev/null; then
    error "Required command not found: $1"
    exit 1
  fi
}

require_cmd curl
require_cmd tar  # not strictly needed but validates the shell env

# Detect checksum utility (used later if checksums are available)
SHASUM_CMD=""
if command -v shasum &>/dev/null; then
  SHASUM_CMD="shasum -a 256"
elif command -v sha256sum &>/dev/null; then
  SHASUM_CMD="sha256sum"
fi

# ── Detect platform ────────────────────────────────────────────────────────

detect_os() {
  local os
  os="$(uname -s)"
  case "$os" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *)
      error "Unsupported operating system: $os"
      error "Hourgit supports macOS and Linux."
      exit 1
      ;;
  esac
}

detect_arch() {
  local arch
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)  echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      error "Unsupported architecture: $arch"
      error "Hourgit supports x86_64 (amd64) and ARM64."
      exit 1
      ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

# Detect Rosetta 2 on macOS — if running under translation, use arm64
if [ "$OS" = "darwin" ] && [ "$ARCH" = "amd64" ]; then
  if sysctl -n sysctl.proc_translated 2>/dev/null | grep -q 1; then
    info "Rosetta 2 detected — installing native ARM64 binary"
    ARCH="arm64"
  fi
fi

info "Platform: ${OS}/${ARCH}"

# ── Directories ─────────────────────────────────────────────────────────────

INSTALL_DIR="$HOME/.hourgit/bin"
DOWNLOAD_DIR="$HOME/.hourgit/downloads"
SYMLINK_DIR="$HOME/.local/bin"

mkdir -p "$INSTALL_DIR" "$DOWNLOAD_DIR" "$SYMLINK_DIR"

# Cleanup downloads on exit
cleanup() {
  rm -rf "$DOWNLOAD_DIR"
}
trap cleanup EXIT

# ── Fetch latest version ───────────────────────────────────────────────────

GITHUB_REPO="Flyrell/hourgit"
API_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"

info "Fetching latest release..."
RELEASE_JSON="$(curl -fsSL "$API_URL")"

VERSION="$(echo "$RELEASE_JSON" | grep '"tag_name"' | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')"

if [ -z "$VERSION" ]; then
  error "Could not determine latest version from GitHub API."
  exit 1
fi

info "Latest version: $(bold "$VERSION")"

# ── Download binary + checksums ─────────────────────────────────────────────

BINARY_NAME="hourgit-${OS}-${ARCH}-${VERSION}"
BASE_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}"

info "Downloading ${BINARY_NAME}..."
curl -fsSL -o "${DOWNLOAD_DIR}/${BINARY_NAME}" "${BASE_URL}/${BINARY_NAME}"

info "Downloading checksums..."
if curl -fsSL -o "${DOWNLOAD_DIR}/SHA256SUMS" "${BASE_URL}/SHA256SUMS" 2>/dev/null; then
  # ── Verify checksum ───────────────────────────────────────────────────────

  if [ -z "$SHASUM_CMD" ]; then
    error "Neither shasum nor sha256sum found. Cannot verify download."
    exit 1
  fi

  info "Verifying checksum..."
  EXPECTED="$(grep "${BINARY_NAME}" "${DOWNLOAD_DIR}/SHA256SUMS" | awk '{print $1}')"

  if [ -z "$EXPECTED" ]; then
    error "Binary ${BINARY_NAME} not found in SHA256SUMS."
    exit 1
  fi

  ACTUAL="$(cd "$DOWNLOAD_DIR" && $SHASUM_CMD "$BINARY_NAME" | awk '{print $1}')"

  if [ "$EXPECTED" != "$ACTUAL" ]; then
    error "Checksum verification failed!"
    error "  Expected: ${EXPECTED}"
    error "  Actual:   ${ACTUAL}"
    exit 1
  fi

  info "Checksum verified"
else
  warn "Checksums not available for this release — skipping verification"
fi

# ── Install ─────────────────────────────────────────────────────────────────

mv "${DOWNLOAD_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/hourgit"
chmod +x "${INSTALL_DIR}/hourgit"

# Remove macOS quarantine attribute to prevent Gatekeeper warnings
if [ "$OS" = "darwin" ]; then
  xattr -d com.apple.quarantine "${INSTALL_DIR}/hourgit" 2>/dev/null || true
fi

info "Installed to $(bold "${INSTALL_DIR}/hourgit")"

# Create symlink in ~/.local/bin
ln -sf "${INSTALL_DIR}/hourgit" "${SYMLINK_DIR}/hourgit"
info "Symlinked to $(bold "${SYMLINK_DIR}/hourgit")"

# ── PATH check ──────────────────────────────────────────────────────────────

check_path() {
  case ":${PATH}:" in
    *":${SYMLINK_DIR}:"*) return 0 ;;
    *) return 1 ;;
  esac
}

if ! check_path; then
  warn "${SYMLINK_DIR} is not in your PATH"
  echo ""
  echo "Add it to your shell config:"
  echo ""

  SHELL_NAME="$(basename "${SHELL:-/bin/bash}")"
  case "$SHELL_NAME" in
    zsh)
      echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc"
      echo "  source ~/.zshrc"
      ;;
    bash)
      echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
      echo "  source ~/.bashrc"
      ;;
    fish)
      echo "  fish_add_path ~/.local/bin"
      ;;
    *)
      echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
      ;;
  esac
  echo ""
fi

# ── Shell completions ──────────────────────────────────────────────────────

info "Installing shell completions..."
if "${INSTALL_DIR}/hourgit" completion install --yes 2>/dev/null; then
  info "Shell completions installed"
else
  warn "Could not install shell completions (run 'hourgit completion install' later)"
fi

# ── Done ────────────────────────────────────────────────────────────────────

echo ""
printf "${GREEN}${BOLD}Hourgit ${VERSION} installed successfully!${RESET}\n"
echo ""
echo "Get started:"
echo "  cd your-repo"
echo "  hourgit init"
echo ""
