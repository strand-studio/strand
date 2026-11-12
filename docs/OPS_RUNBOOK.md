# Strand operations runbook

Short ops guide for self-hosted Strand (API + web console).

## Deploy

1. Build / run the Go API (`cmd` / docker as you ship).
2. Serve `web` dist behind the same origin (or set `VITE_API_BASE`).
3. Set secrets: JWT / API token, DB path, encryption key for stored credentials.
4. Confirm `GET /api/config` and `GET /api/crypto/config` respond.

### OpenAI / overseas AI endpoints

Strand’s HTTP client honors `HTTPS_PROXY` / `HTTP_PROXY` / `ALL_PROXY` (and lowercase variants).  
If `api.openai.com` times out from your network, start the API process with a working outbound proxy, **or** set a reachable Custom Base URL (OpenAI-compatible relay) in Settings → AI models.

```bash
export HTTPS_PROXY=http://127.0.0.1:7890
export HTTP_PROXY=http://127.0.0.1:7890
go run ./cmd/strand start
```

Credential tests use `GET /v1/models` for OpenAI-compatible providers (auth check, no chat spend).

## First live check

Follow [LAUNCH_CHECKLIST.md](./LAUNCH_CHECKLIST.md):

- Claw402 fee wallet ≥ $1 USDC on Base
- Hyperliquid Agent + builder fee (or CEX API + IP whitelist)
- Strategy risk gates saved
- `POST /api/launch/preflight` green → start Autopilot

## Backup / restore

| Action | API | UI |
|--------|-----|----|
| Export (metadata only) | `GET /api/backup/export` | Settings → Account → Export (no secrets) |
| Export with secrets | `GET /api/backup/export?include_secrets=1` | Settings → Export (with secrets) |
| Import | `POST /api/backup/import` | Settings → Import backup |

- Default export **omits** API keys and wallet private keys.
- Secret exports must stay offline; treat like a password vault dump.
- Import always recreates strategies as `"(restored)"` copies. Credential restore requires `restore_secrets: true` and a secrets-capable backup.

## Runtime hot updates

While Autopilot is running you can change without full restart:

| Field | How |
|-------|-----|
| Scan interval | `PUT /api/traders/:id/runtime` `{ "scan_interval_minutes": N }` |
| BTC/ETH & alt leverage | same endpoint `{ "btc_eth_leverage", "altcoin_leverage" }` — patches linked strategy `risk_control` and reloads live |
| Strategy risk / coin source | Save on Strategy → running traders with that `strategy_id` force-reload |

Structural changes (exchange, AI model, prompt template) still reload the trader instance.

## Failure attribution

Portfolio (`/portfolio`) shows a **Failure attribution** panel when:

- Runtime crash / stop (`last_runtime_error`)
- Trading auth blocked
- Latest AI cycle error / order action error
- Fee wallet empty/low
- Safe mode / pause (daily loss / drawdown)

Use it before digging through server logs.

## Alerts

Telegram / Discord / Slack notify channels fan out start/stop/pause/safe-mode/crash when bound (see Chat / Settings notify).

## Useful smoke curls

```bash
# after login JWT
curl -s -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8080/api/backup/export | jq .exported_at,.include_secrets

curl -s -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"ai_model_id":"","exchange_id":""}' \
  http://127.0.0.1:8080/api/launch/preflight | jq .
```

See also: [WEB_API.md](./WEB_API.md), [LAUNCH_CHECKLIST.md](./LAUNCH_CHECKLIST.md).
