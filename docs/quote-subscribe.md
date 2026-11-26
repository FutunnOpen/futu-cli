# `futu quote subscribe`

`futu quote subscribe` opens the quote-push socket, authenticates with the stored OAuth token, subscribes, and prints each server message as one compact JSON line. The token is never printed. The command stops at `--duration` (default `20s`, maximum `30m`) or after `--max-messages` subscribed pushes (default `50`; `0` means no cap).

```bash
futu quote subscribe --symbols HK.00700 --types quote,order_book --duration 20s
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--symbols` | required | Comma-separated `MARKET.CODE` symbols |
| `--types` | `quote` | `quote`, `order_book`, `ticker`, `kline` |
| `--duration` | `20s` | How long to listen |
| `--max-messages` | `50` | Stop after this many subscribed pushes |
| `--kline-period` | `1m` | Used when `types` includes `kline` |
| `--kline-adjust` | `none` | Used when `types` includes `kline` |

## Endpoint

`wss://webapi-quote.futunn.com/ws`, derived from `api_base` by rewriting the `webapi.` host to `webapi-quote.`.

Official reference: [quote push overview](https://open.futunn.com/zh-cn/api/quote/push/overview).

## What the command does

1. Connect.
2. Send an OAuth auth frame (`action` `auth`, `auth_type` `oauth2`, `authorization` `Bearer <access_token>`). A successful reply has `session_id` and `server_time`.
3. Send `action` `subscribe`. Request fields match `--types`: `quote`, `order_book`, `ticker`, and `kline` (`symbol`, `period`, `adjust`).
4. Print every server message. Keep the socket alive with WebSocket ping/pong. Refresh the stored token and send a refresh frame about every 5 minutes. After a disconnect, connect again, authenticate again, and subscribe again.

## Output

A line with `ret_code` `0` and `data.id` `sub-1` is the subscribe acknowledgement. A line with `type` is a push.

| `--types` value | Push `type` |
| --- | --- |
| `quote` | `QUOTE` |
| `order_book` | `ORDER_BOOK`, and `BROKER_QUEUE` when the server attaches a broker queue |
| `ticker` | `TICKER` |
| `kline` | `KLINE` |

`BROKER_QUEUE` is not a `--types` value. It arrives with `order_book` for an eligible symbol, often a Hong Kong stock, and counts toward `--max-messages` only when `order_book` is subscribed. `MARKET_STATE` is printed and does not count. `futu broker` is a REST query and is not this stream.

A non-zero `ret_code`, `data.code`, or top-level `code` stops the command and prints the server message.
