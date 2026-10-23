#!/usr/bin/env bash
set -euo pipefail

CHECKSUMS="${1:-}"
OUTPUT="${2:-}"

if [[ -z "$CHECKSUMS" ]]; then
  echo "usage: $0 <checksums-file> [signature-output]" >&2
  exit 1
fi

if [[ ! -f "$CHECKSUMS" ]]; then
  echo "checksums file not found: $CHECKSUMS" >&2
  exit 1
fi

if [[ -z "$OUTPUT" ]]; then
  OUTPUT="${CHECKSUMS}.sig"
fi

TMP_KEY="$(mktemp)"
cleanup() {
  rm -f "$TMP_KEY"
}
trap cleanup EXIT

if [[ -n "${FUTU_CLI_RELEASE_SIGNING_KEY_FILE:-}" ]]; then
  if [[ ! -f "$FUTU_CLI_RELEASE_SIGNING_KEY_FILE" ]]; then
    echo "FUTU_CLI_RELEASE_SIGNING_KEY_FILE does not point to a file: $FUTU_CLI_RELEASE_SIGNING_KEY_FILE" >&2
    exit 1
  fi
  cp "$FUTU_CLI_RELEASE_SIGNING_KEY_FILE" "$TMP_KEY"
elif [[ -n "${FUTU_CLI_RELEASE_SIGNING_KEY:-}" ]]; then
  printf '%s\n' "$FUTU_CLI_RELEASE_SIGNING_KEY" >"$TMP_KEY"
else
  echo "set FUTU_CLI_RELEASE_SIGNING_KEY to the PEM private key contents, or FUTU_CLI_RELEASE_SIGNING_KEY_FILE to a key path" >&2
  exit 1
fi
chmod 600 "$TMP_KEY"

openssl dgst -sha256 -sign "$TMP_KEY" -out "$OUTPUT" "$CHECKSUMS"
echo "wrote signature $OUTPUT"
