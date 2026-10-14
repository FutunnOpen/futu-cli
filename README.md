# futu-cli

Futu CLI is a command-line tool for Futu OpenAPI workflows, including OAuth login, market data queries, trading operations, crypto trading, configuration, and self-update.

## Install

The shell installers are the recommended path because they do not require Node.js.

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
npm install -g github:FutunnOpen/futu-cli --install-links=true
```

The npm wrapper requires Node.js 18 or newer and npm 9 or newer. `--install-links=true` is recommended for GitHub installs so npm copies the packed package instead of linking to a temporary git checkout.

The scoped npm registry package is not published yet. After it is published, this will also be available:

```bash
npm install -g @futunn/futu-cli
```

If npm reports `spawn sh ENOENT`, configure the lifecycle script shell:

```bash
npm config set script-shell /bin/sh
```

If npm reports `uv_cwd` or `getcwd`, run the install from an existing directory:

```bash
cd ~
npm install -g github:FutunnOpen/futu-cli --install-links=true
```

If you use nvm, make sure `node` and `npm` come from the same Node installation before installing:

```bash
which node
which npm
node -v
npm -v
npm prefix -g
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
futu quote US.AAPL
```

List simulated trading accounts:

```bash
futu sim account list
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
npm install -g github:FutunnOpen/futu-cli --install-links=true
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
client_id_api_base
client_id_oauth_base
redirect_uri
default_account
auto_update
```

`client_id` is bound to both `client_id_api_base` and `client_id_oauth_base`. If either binding is missing or does not match the current `api_base`, `futu auth login` registers and saves a new `client_id` instead of reusing the old one.

## Commands

Most commands support `--format json` for JSON output and `--format table` (default) for tabular output. Commands that interact with an account accept `--account <account-id>`; use `futu account list` to find the ID. Securities quote/trading symbols use `MARKET.CODE`, for example `US.AAPL` or `HK.00700`; crypto symbols keep their pair format, for example `BTCUSD`.

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

### Market Data

| Command | Description |
|---------|-------------|
| `futu quote <symbol...>` | Query real-time quotes for one or more securities |
| `futu basicinfo <symbol...>` | Query static security information |
| `futu snapshot <symbol...>` | Query market snapshots in batch |
| `futu kline <symbol>` | Query recent K-line/candlestick data |
| `futu history-kline <symbol>` | Query historical K-line data; supports `--start`, `--end`, and pagination via `--next-time` |
| `futu ticker <symbol>` | Query recent time-and-sales ticks |
| `futu rt-data <symbol>` | Query intraday time-sharing data |
| `futu order-book <symbol>` | Query order book depth; `futu depth <symbol>` is an alias |
| `futu rank [type]` | Query market rankings such as volume, turnover, gainers, losers, and amplitude |
| `futu option chain <symbol>` | Query option chains |
| `futu broker <symbol>` | Query broker queues; use `--view history` for historical broker data |
| `futu capital <symbol>` | Query capital flow; use `--view history` or `--view distribution` for other views |
| `futu company <symbol>` | Query company profile, executives, executive background, or operational efficiency |
| `futu corporate-actions <symbol>` | Query dividends, splits, buybacks, or rehab data |
| `futu economic-calendar [keyword]` | Query hot economic events or search economic calendar items |
| `futu financials <symbol>` | Query financial statements, revenue breakdown, and earnings-related data |
| `futu ipo-list` | Query IPO lists by market |
| `futu option-data <symbol>` | Query option expiration, strategy, spread, exercise probability, or volatility data |
| `futu plate [symbol]` | Query plate lists, plate stocks, or owner plates |
| `futu research <symbol>` | Query analyst consensus, Morningstar reports, or rating summaries |
| `futu search [keyword]` | Search news, community posts, or stock feeds |
| `futu shareholders <symbol>` | Query shareholder overview, institutional holders, holding changes, and insider data |
| `futu short <symbol>` | Query short volume or short interest |
| `futu valuation [symbol]` | Query valuation detail, plate stocks, index stocks, or index plates |
| `futu watchlist [symbol...]` | List watchlists or add/delete/move securities |
| `futu future-info <symbol...>` | Query futures contract information |
| `futu market-state <symbol...>` | Query market state for one or more symbols |
| `futu combo-option-quote` | Query combo option quotes; requires `--combo-list` JSON |
| `futu referencefuture-list <symbol>` | Query related futures contracts |
| `futu stock-screen` | Run stock screening queries |
| `futu warrant-screen` | Run warrant screening queries |
| `futu option-screen` | Run option screening queries |
| `futu trading-days` | Query trading days by market and date range |

Examples:

```bash
futu quote HK.00700 US.AAPL --format json
futu kline US.AAPL --ktype 1 --format json
futu capital HK.00700 --view history --period-type WEEK --format json
futu plate --view list --market HK --plate-class INDUSTRY --format json
futu trading-days --market US --start 2026-10-01 --end 2026-10-31 --format json
```

### Trading

| Command | Description |
|---------|-------------|
| `futu account list` | List authorized trading accounts; the Account ID is used as `--account` |
| `futu account funds` | Query account funds (total assets, cash, market value, buying power, P/L, risk status) |
| `futu position` | List positions; filter by `--symbol US.AAPL` or P/L ratio |
| `futu order buy <symbol> <qty>` | Place a buy order (market/limit/stop/limit_if_touched); confirms before submission |
| `futu order sell <symbol> <qty>` | Place a sell order; same parameters as `order buy` |
| `futu order place <qty>` | Submit a generic securities order (SELL_SHORT, BUY_BACK, conditional, multi-leg) |
| `futu order modify <order-id>` | Modify order qty/price/trigger; requires `--exchange`; confirms before submission |
| `futu order cancel <order-id>` | Cancel an order; requires `--exchange`; confirms before submission |
| `futu order list` | List open orders; supports `--market` and `--page-size` |
| `futu order history` | List historical orders; supports `--market`, `--symbol US.AAPL`, `--start`/`--end` (microseconds) |
| `futu order detail <order-id> [order-id...]` | Batch query order details; up to 100 IDs; requires `--exchange` |
| `futu order max-qty <symbol>` | Query max buy/sell/sell-short/buy-back quantities and margin requirements |
| `futu deal` | List today's filled trades |
| `futu deal history` | List historical filled trades; supports `--market`, `--symbol US.AAPL`, `--start`/`--end` (microseconds) |

### Simulated Trading

Simulated trading commands are grouped under `futu sim`. If `--account` is omitted, the CLI uses the first simulated account returned by `futu sim account list`.

| Command | Description |
|---------|-------------|
| `futu sim account list` | List simulated trading accounts |
| `futu sim account funds` | Query simulated account funds and assets |
| `futu sim position` | List simulated positions; supports `--market` |
| `futu sim order list` | List active simulated orders |
| `futu sim order history` | List historical simulated orders; supports `--start`/`--end`, `--page-size`, and `--next-key` |
| `futu sim order max-qty <symbol>` | Query max simulated buy/sell quantity for a symbol |
| `futu sim order buy <symbol> <qty>` | Place a simulated buy order |
| `futu sim order sell <symbol> <qty>` | Place a simulated sell order |
| `futu sim order place <qty>` | Place a generic simulated order; direction via `--side` and symbol via `--symbol` |
| `futu sim order modify <order-id>` | Modify simulated order; requires both `--qty` and `--price` |
| `futu sim order cancel <order-id>` | Cancel a simulated order |

Examples:

```bash
futu sim account list --format json
futu sim account funds --format json
futu sim position --market US --format json
futu sim order max-qty US.AAPL --market US --format json
futu sim order buy US.AAPL 1 --market US --type market --format json
futu sim order modify <order-id> --market US --qty 2 --price 181 --format json
futu sim order cancel <order-id> --market US --format json
```

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
