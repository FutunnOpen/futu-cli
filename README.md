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

After the package is published to the npm registry, this will also be available (WIP — not published yet):

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

After the package is published to the npm registry, use (WIP — not published yet):

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

## Commands

Most commands support `--format json` for JSON output and `--format table` (default) for tabular output. Commands that interact with an account accept `--account <account-id>`; use `futu account list` to find the ID. Securities trading commands default to market `HK`; use `--market` (or `-m`) to switch.

### Authentication

| Command | Description |
|---------|-------------|
| `futu auth login` | OAuth2 Authorization Code + PKCE login; reuses valid token, refreshes when needed |
| `futu auth login --no-browser` | Print the authorization URL instead of opening a browser; paste the callback URL back |
| `futu auth status` | Show token validity, permissions, account ID, and expiry from the local token file |
| `futu auth status --format json` | Show local authentication status as JSON |
| `futu auth logout` | Delete the local token file only; no server-side call |
| `futu auth register` | Re-register the OAuth2 client (only if the registered client needs resetting) |
| `futu check` | Verify token validity online and measure API HTTP latency |
| `futu check --format json` | Verify token and API connectivity as JSON |

### Trading

| Command | Description |
|---------|-------------|
| `futu account list` | List authorized trading accounts; the Account ID is used as `--account` |
| `futu account funds` | Query account funds (total assets, cash, market value, buying power, P/L, risk status) |
| `futu position` | List positions; filter by `--symbol` or P/L ratio |
| `futu order buy <symbol> <qty>` | Place a buy order (market/limit/stop/limit_if_touched); confirms before submission |
| `futu order sell <symbol> <qty>` | Place a sell order; same parameters as `order buy` |
| `futu order place <qty>` | Submit a generic securities order (SELL_SHORT, BUY_BACK, conditional, multi-leg) |
| `futu order modify <order-id>` | Modify order qty/price/trigger; requires `--exchange`; confirms before submission |
| `futu order cancel <order-id>` | Cancel an order; requires `--exchange`; confirms before submission |
| `futu order list` | List open orders; supports `--market` and `--page-size` |
| `futu order history` | List historical orders; supports `--market`, `--symbol`, `--start`/`--end` (microseconds) |
| `futu order detail <order-id> [order-id...]` | Batch query order details; up to 100 IDs; requires `--exchange` |
| `futu order max-qty <symbol>` | Query max buy/sell/sell-short/buy-back quantities and margin requirements |
| `futu deal` | List today's filled trades |
| `futu deal history` | List historical filled trades; supports `--market`, `--symbol`, `--start`/`--end` (microseconds) |

### Crypto

| Command | Description |
|---------|-------------|
| `futu crypto account list` | List accounts authorized for crypto trading |
| `futu crypto balance` | Query crypto account total balance (cash + digital assets) |
| `futu crypto max-qty <symbol>` | Query max cash buy quantity, max sell quantity, and max cash buy amount |
| `futu crypto order place <symbol>` | Place a crypto order; direction via `--side`; supports `--qty` or `--cash-qty` |
| `futu crypto order buy <symbol>` | Place a crypto buy order; supports `--qty` or `--cash-qty` (amount-based market buy) |
| `futu crypto order sell <symbol>` | Place a crypto sell order; `--cash-qty` not supported |
| `futu crypto order modify <order-id>` | Modify price/qty/conditional info; requires `--order-version` from order detail |
| `futu crypto order cancel <order-id>` | Cancel a crypto order |
| `futu crypto order list` | List open crypto orders; supports `--page-size` and `--page-flag` |
| `futu crypto order history` | List historical crypto orders; defaults to last 3 months; supports `--symbol`/`--start`/`--end` |
| `futu crypto order detail <order-id> [order-id...]` | Batch query crypto order details; up to 20 IDs |
| `futu crypto deal <order-id>` | Query trade fills for a specific order; supports `--group-by-order` |
| `futu crypto deal history` | List historical crypto fills; defaults to last 3 months; supports `--group-by-order` |

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
