#!/usr/bin/env bash
set -euo pipefail

PRODUCT="futu"
BINARY="futu"
CHECKSUMS_ASSET="futu_checksums.txt"
OUTPUT_DIR="${OUTPUT_DIR:-dist}"
VERSION="${1:-${FUTU_CLI_VERSION:-}}"

if [[ -z "$VERSION" ]]; then
  echo "usage: $0 vMAJOR.MINOR.PATCH[-PRERELEASE]" >&2
  exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid version: $VERSION" >&2
  exit 1
fi

if [[ -z "$OUTPUT_DIR" || "$OUTPUT_DIR" == "/" ]]; then
  echo "invalid output dir: $OUTPUT_DIR" >&2
  exit 1
fi

MODULE="$(go list -m)"
LDFLAGS="-s -w -X ${MODULE}/cmd.Version=${VERSION}"

rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

copy_release_files() {
  local package_dir="$1"

  cp README.md "$package_dir/README.md"
  if [[ -f LICENSE ]]; then
    cp LICENSE "$package_dir/LICENSE"
  fi
  if [[ -f CHANGELOG.md ]]; then
    cp CHANGELOG.md "$package_dir/CHANGELOG.md"
  fi
}

archive_package() {
  local package_dir="$1"
  local asset="$2"
  local ext="$3"

  case "$ext" in
    tar.gz)
      tar -czf "$OUTPUT_DIR/$asset" -C "$package_dir" .
      ;;
    zip)
      if ! command -v zip >/dev/null 2>&1; then
        echo "zip is required to build $asset" >&2
        exit 1
      fi
      (cd "$package_dir" && zip -qr "../$asset" .)
      ;;
    *)
      echo "unsupported archive extension: $ext" >&2
      exit 1
      ;;
  esac
}

build_package() {
  local goos="$1"
  local goarch="$2"
  local platform="$3"
  local ext="$4"
  local exe="$BINARY"
  local package_dir="$OUTPUT_DIR/package_$platform"
  local asset="${PRODUCT}_${VERSION}_${platform}.${ext}"

  if [[ "$goos" == "windows" ]]; then
    exe="${BINARY}.exe"
  fi

  mkdir -p "$package_dir"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "$LDFLAGS" -o "$package_dir/$exe" .
  copy_release_files "$package_dir"
  archive_package "$package_dir" "$asset" "$ext"
  rm -rf "$package_dir"
}

write_checksums() {
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$OUTPUT_DIR" && sha256sum "${PRODUCT}_${VERSION}_"* > "$CHECKSUMS_ASSET")
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    (cd "$OUTPUT_DIR" && shasum -a 256 "${PRODUCT}_${VERSION}_"* > "$CHECKSUMS_ASSET")
    return
  fi
  echo "sha256sum or shasum is required to write checksums" >&2
  exit 1
}

build_package darwin amd64 darwin_amd64 tar.gz
build_package darwin arm64 darwin_arm64 tar.gz
build_package linux amd64 linux_amd64 tar.gz
build_package linux arm64 linux_arm64 tar.gz
build_package linux amd64 linux_musl_amd64 tar.gz
build_package linux arm64 linux_musl_arm64 tar.gz
build_package windows amd64 windows_amd64 zip

write_checksums

echo "release assets written to $OUTPUT_DIR"
