# futu CLI Release Guide

This document is for maintainers who build, publish, and verify `futu` CLI releases. User-facing installation instructions live in `README.md`.

## Release Source

The default GitHub Release source is:

```text
https://github.com/FutunnOpen/futu-cli
```

The CLI resolves release metadata from:

```text
{release_base}/releases/latest
{release_base}/releases/download/{version}/{asset}
```

Use `FUTU_CLI_RELEASE_BASE` only when testing a fork or private release source.

## Release Assets

Each release must include these archives and checksum file:

```text
futu_v<version>_darwin_amd64.tar.gz
futu_v<version>_darwin_arm64.tar.gz
futu_v<version>_linux_amd64.tar.gz
futu_v<version>_linux_arm64.tar.gz
futu_v<version>_linux_musl_amd64.tar.gz
futu_v<version>_linux_musl_arm64.tar.gz
futu_v<version>_windows_amd64.zip
futu_checksums.txt
```

The release tag, binary version, npm wrapper version, and asset names must describe the same version. For example, npm version `0.1.0` maps to Git tag `v0.1.0`.

## GitHub Release Flow

Stable releases are published with the manual GitHub Actions `Release` workflow.

The workflow accepts a release version such as `v0.1.0`. If omitted, it reads the root `package.json` version and prefixes it with `v`.

The workflow runs:

```text
go test ./...
scripts/build-release.sh <version>
git tag -a <version> <current-commit>
git push origin <version>
gh release create <version> dist/* --verify-tag
```

If the tag already exists, the workflow verifies that it points to the current commit. If it points elsewhere, publishing stops so source tags and binary assets cannot drift.

## Local Build Check

Build local release assets before publishing when changing release scripts:

```bash
scripts/build-release.sh v0.1.0-test.local
```

The generated files are written to `dist/`, which is ignored by Git.

## Installer Smoke Tests

macOS / Linux:

```bash
FUTU_CLI_VERSION=v0.1.0-test.local sh scripts/install.sh
futu version
futu update --check
```

Windows PowerShell:

```powershell
$env:FUTU_CLI_VERSION="v0.1.0-test.local"
powershell -ExecutionPolicy Bypass -File scripts/install.ps1
futu version
```

To test a fork or private release source:

```bash
FUTU_CLI_RELEASE_BASE=https://github.com/<owner>/<repo> sh scripts/install.sh
```

Linux defaults to the musl package. To test the glibc package:

```bash
FUTU_CLI_LIBC=glibc sh scripts/install.sh
```

## npm Wrapper

The npm wrapper downloads the platform archive from GitHub Releases during `postinstall`.

Install from GitHub while npm registry publication is still in progress:

```bash
npm install -g github:FutunnOpen/futu-cli
```

Once the registry package is published, the public install command will be:

```bash
npm install -g @futunn/futu-cli
```

Test the local npm wrapper against a temporary release:

```bash
FUTU_CLI_VERSION=v0.1.0-test.local npm install -g .
```

## Post-Release Verification

```bash
futu version
futu update --check
futu config init
futu auth login
futu auth status
npm pack --dry-run
```
