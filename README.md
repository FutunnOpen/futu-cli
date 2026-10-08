# futu-cli

Futu CLI is a command-line tool for Futu OpenAPI workflows, including OAuth login, market data queries, trading operations, crypto trading, configuration, and self-update.

## Install

macOS / Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/FutunnOpen/futu-cli/main/scripts/install.sh | sh
```

Restart the terminal after installation, then verify:

```bash
futu version
```

Windows PowerShell:

```powershell
iwr -UseBasicParsing https://raw.githubusercontent.com/FutunnOpen/futu-cli/main/scripts/install.ps1 -OutFile install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

npm:

```bash
npm install -g github:FutunnOpen/futu-cli
```

After the package is published to the npm registry, this will also be available:

```bash
npm install -g @futunn/futu-cli
```

## Quick Start

Initialize configuration and sign in:

```bash
futu config init
```

Check authentication and API connectivity:

```bash
futu auth status
futu check
```

Run a quote query:

```bash
futu quote AAPL.US
```

## Update

Check whether a newer stable GitHub Release is available:

```bash
futu update --check
```

Apply the latest stable release:

```bash
futu update
```

If installed from GitHub through npm, reinstall from GitHub:

```bash
npm install -g github:FutunnOpen/futu-cli
```

After the package is published to the npm registry, use:

```bash
npm update -g @futunn/futu-cli
```

## Configuration

List configuration:

```bash
futu config list
```

Common keys:

```text
api_base
client_id
redirect_uri
default_account
auto_update
```

## Release Assets

Official releases are published from GitHub Actions. The release tag, GitHub Release, binary version, and asset names must use the same version, for example `v0.1.0`.

Each release includes platform archives and one checksum file:

```text
futu_v0.1.0_darwin_amd64.tar.gz
futu_v0.1.0_darwin_arm64.tar.gz
futu_v0.1.0_linux_amd64.tar.gz
futu_v0.1.0_linux_arm64.tar.gz
futu_v0.1.0_linux_musl_amd64.tar.gz
futu_v0.1.0_linux_musl_arm64.tar.gz
futu_v0.1.0_windows_amd64.zip
futu_checksums.txt
```

## Development

```bash
go test ./...
scripts/build-release.sh v0.1.0
```

## License

MIT
