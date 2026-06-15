#!/usr/bin/env bash
set -euo pipefail

REPO="susugadx/xelyon-cli"
INSTALL_DIR="${HOME}/.local/bin"
VERSION="latest"
DRY_RUN=0
YES=0

usage() {
  cat <<'EOF'
Usage: install.sh [--version <tag>] [--install-dir <dir>] [--dry-run] [--yes]

Installs the xelyon release binary from GitHub Releases.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="${2:-}"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --yes|-y)
      YES=1
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [ -z "$INSTALL_DIR" ]; then
  echo "--install-dir must not be empty" >&2
  exit 2
fi

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Required command not found: $1" >&2
    exit 1
  fi
}

fetch() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --connect-timeout 15 --max-time 120 --retry 2 "$1"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    wget --timeout=30 --tries=3 -qO- "$1"
    return
  fi
  echo "curl or wget is required" >&2
  exit 1
}

download() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --connect-timeout 15 --max-time 120 --retry 2 "$1" -o "$2"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    wget --timeout=30 --tries=3 -qO "$2" "$1"
    return
  fi
  echo "curl or wget is required" >&2
  exit 1
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
    return
  fi
  echo "sha256sum or shasum is required" >&2
  exit 1
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
}

confirm_overwrite() {
  target="$1"
  if [ "$YES" -eq 1 ] || [ ! -e "$target" ]; then
    return
  fi
  if ! ( : </dev/tty ) 2>/dev/null || ! ( : >/dev/tty ) 2>/dev/null; then
    echo "Refusing to overwrite ${target} without an interactive terminal." >&2
    echo "Run with --yes for non-interactive installs." >&2
    exit 1
  fi

  exec 3</dev/tty
  exec 4>/dev/tty
  printf "Overwrite %s? [y/N] " "$target" >&4
  IFS= read -r answer <&3 || answer=""
  exec 3<&-
  exec 4>&-
  case "$answer" in
    y|Y|yes|YES) ;;
    *) echo "Cancelled" >&2; exit 1 ;;
  esac
}

release_api_url() {
  if [ "$VERSION" = "latest" ]; then
    echo "https://api.github.com/repos/${REPO}/releases/latest"
  else
    echo "https://api.github.com/repos/${REPO}/releases/tags/${VERSION}"
  fi
}

OS_NAME="$(detect_os)"
ARCH_NAME="$(detect_arch)"
API_URL="$(release_api_url)"
RELEASE_JSON="$(fetch "$API_URL")"

asset_urls() {
  printf '%s\n' "$RELEASE_JSON" | sed -n 's/.*"browser_download_url":[[:space:]]*"\([^"]*\)".*/\1/p'
}

ASSET_URL="$(asset_urls | grep "_${OS_NAME}_${ARCH_NAME}" | grep -E '\.(tar\.gz|zip)$' | head -n 1 || true)"
CHECKSUM_URL="$(asset_urls | grep '/checksums.txt$' | head -n 1 || true)"

if [ -z "$ASSET_URL" ] || [ -z "$CHECKSUM_URL" ]; then
  echo "Could not find release asset for ${OS_NAME}/${ARCH_NAME}" >&2
  exit 1
fi

echo "xelyon installer"
echo "  release: ${VERSION}"
echo "  asset:   ${ASSET_URL}"
echo "  sums:    ${CHECKSUM_URL}"
echo "  target:  ${INSTALL_DIR}/xelyon"

if [ "$DRY_RUN" -eq 1 ]; then
  exit 0
fi

need_cmd awk
need_cmd grep
need_cmd tar

confirm_overwrite "${INSTALL_DIR}/xelyon"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

ARCHIVE="${TMP_DIR}/$(basename "$ASSET_URL")"
SUMS="${TMP_DIR}/checksums.txt"
download "$ASSET_URL" "$ARCHIVE"
download "$CHECKSUM_URL" "$SUMS"

EXPECTED="$(grep " $(basename "$ASSET_URL")$" "$SUMS" | awk '{print $1}' || true)"
if [ -z "$EXPECTED" ]; then
  echo "Checksum entry not found for $(basename "$ASSET_URL")" >&2
  exit 1
fi
ACTUAL="$(sha256_file "$ARCHIVE")"
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Checksum mismatch for $(basename "$ASSET_URL")" >&2
  echo "expected: $EXPECTED" >&2
  echo "actual:   $ACTUAL" >&2
  exit 1
fi

mkdir -p "$TMP_DIR/extract"
case "$ARCHIVE" in
  *.tar.gz) tar -xzf "$ARCHIVE" -C "$TMP_DIR/extract" ;;
  *.zip)
    need_cmd unzip
    unzip -q "$ARCHIVE" -d "$TMP_DIR/extract"
    ;;
  *) echo "Unsupported archive: $ARCHIVE" >&2; exit 1 ;;
esac

BINARY="$(find "$TMP_DIR/extract" -type f -name xelyon -perm -u+x | head -n 1 || true)"
if [ -z "$BINARY" ]; then
  BINARY="$(find "$TMP_DIR/extract" -type f -name xelyon | head -n 1 || true)"
fi
if [ -z "$BINARY" ]; then
  echo "xelyon binary not found in archive" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
install -m 0755 "$BINARY" "${INSTALL_DIR}/xelyon"

echo "Installed ${INSTALL_DIR}/xelyon"
if ! command -v xelyon >/dev/null 2>&1; then
  echo "Add this to your PATH:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi
"${INSTALL_DIR}/xelyon" --version
