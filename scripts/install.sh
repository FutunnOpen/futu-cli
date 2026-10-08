#!/bin/sh
set -eu

PRODUCT="futu"
BINARY="futu"
ENV_PREFIX="FUTU_CLI"
CHECKSUMS_ASSET="futu_checksums.txt"
DEFAULT_INSTALL_DIR="$HOME/.futu/bin"
DEFAULT_RELEASE_BASE="https://github.com/FutunnOpen/futu-cli"
INSTALL_DIR="${INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
RELEASE_BASE="${FUTU_CLI_RELEASE_BASE:-$DEFAULT_RELEASE_BASE}"
VERSION="${FUTU_CLI_VERSION:-}"
LIBC="${FUTU_CLI_LIBC:-musl}"

RELEASE_BASE="${RELEASE_BASE%/}"
LATEST_URL="$RELEASE_BASE/releases/latest"

validate_version() {
  if ! printf '%s' "$1" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'; then
    echo "invalid version: $1" >&2
    exit 1
  fi
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
}

detect_platform() {
  os="$(uname -s)"
  arch="$(detect_arch)"
  case "$os" in
    Darwin) echo "darwin_$arch" ;;
    Linux)
      if [ "$LIBC" = "glibc" ]; then
        echo "linux_$arch"
      else
        echo "linux_musl_$arch"
      fi
      ;;
    *) echo "unsupported OS: $os" >&2; exit 1 ;;
  esac
}

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
    return
  fi
  sha256sum "$1" | awk '{print $1}'
}

if [ -z "$VERSION" ]; then
  LATEST_REDIRECT="$(curl -fsSI -o /dev/null -w '%{redirect_url}' "$LATEST_URL")"
  if [ -z "$LATEST_REDIRECT" ]; then
    LATEST_REDIRECT="$(curl -fsSL -o /dev/null -w '%{url_effective}' "$LATEST_URL")"
  fi
  VERSION="$(printf '%s' "$LATEST_REDIRECT" | sed 's/[?#].*$//' | sed 's#/$##' | awk -F/ '{print $NF}')"
fi
validate_version "$VERSION"

PLATFORM="$(detect_platform)"
ASSET="${BINARY}_${VERSION}_${PLATFORM}.tar.gz"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

ARCHIVE="$TMP_DIR/$ASSET"
CHECKSUMS="$TMP_DIR/$CHECKSUMS_ASSET"
curl -fsSL "$RELEASE_BASE/releases/download/$VERSION/$ASSET" -o "$ARCHIVE"
curl -fsSL "$RELEASE_BASE/releases/download/$VERSION/$CHECKSUMS_ASSET" -o "$CHECKSUMS"

EXPECTED="$(awk -v asset="$ASSET" '$2 == asset {print $1}' "$CHECKSUMS")"
if [ -z "$EXPECTED" ]; then
  echo "checksum for $ASSET not found" >&2
  exit 1
fi
ACTUAL="$(sha256_file "$ARCHIVE")"
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "checksum mismatch for $ASSET" >&2
  exit 1
fi

mkdir -p "$TMP_DIR/unpack" "$INSTALL_DIR"
tar -xzf "$ARCHIVE" -C "$TMP_DIR/unpack"
if [ ! -f "$TMP_DIR/unpack/$BINARY" ]; then
  echo "$BINARY not found in archive" >&2
  exit 1
fi
chmod 0755 "$TMP_DIR/unpack/$BINARY"
if ! "$TMP_DIR/unpack/$BINARY" version | grep -q "$VERSION"; then
  echo "$BINARY version output does not contain $VERSION" >&2
  exit 1
fi

cp "$TMP_DIR/unpack/$BINARY" "$INSTALL_DIR/$BINARY"
chmod 0755 "$INSTALL_DIR/$BINARY"

echo "$BINARY $VERSION installed to $INSTALL_DIR/$BINARY"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Add $INSTALL_DIR to PATH before running $BINARY." ;;
esac
