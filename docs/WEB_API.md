# Strand Web Console API

Web (`strand-web`) talks to the Go server under **`/api/*`**.  
The CLI/operator surface remains at **`/api/v1/*`**.

## Ready for the main path

| Area | Endpoints |
|------|-----------|
| Bootstrap | `GET /api/config`, `GET /api/crypto/config` |
| Auth | `POST /api/register`, `/login`, `/admin-login`, `/logout`, `PUT /api/user/password` |
| Traders | `GET /my-traders`, CRUD `/traders`, `start`/`stop`/`pause`/`resume`, `PUT /traders/:id/runtime` (live scan interval **and** leverage), `GET /traders/:id/config` (strategy lev + hard-gate limits), `competition`, `prompt` |
| Portfolio (console APIs) | `GET /status` (incl. `peak_equity` / `day_start_equity` / gate limits / `last_ai_error` / `last_order_error`), `/account`, `/positions`, `/decisions`, `/statistics`, `/competition`, `/equity-history`, `POST /equity-history-batch`, `/positions/history` |
| Models / Exchanges | `GET/PUT /models`, `POST /models/test`, `GET/POST/PUT/DELETE /exchanges`, `POST /exchanges/test`, `GET /exchanges/account-state`, supported lists |
| Backup | `GET /backup/export` (`?include_secrets=1`), `POST /backup/import` — see [OPS_RUNBOOK.md](./OPS_RUNBOOK.md) |
| Launch readiness | `POST /launch/preflight`, `GET /traders/:id/preflight` — AI wallet + exchange balance + trade-auth (HL agent) checks; start rejects with `error_key=trader.start.preflight_failed` |
| Risk hard gates | Strategy `risk_control.daily_loss_limit_pct` / `max_drawdown_pct` pause Autopilot (keep positions); Portfolio `POST /traders/:id/pause|resume`; Telegram alerts on pause / safe mode / trading blocked / crash |
| Strategies | full CRUD + `active` / `activate` / `duplicate` / `default-config` |
| Markets (public) | `GET /chains`, `/symbols`, `/klines` — no JWT; powers `/markets` page |
| Vergex | `/vergex/direction-change/*`, `/cost-liquidation-heatmap`, `/holder-winrate-map*`, `/flow-markets` |
| AI Chat | `GET /chat/tools`, `POST /chat/tools/test`, `POST /chat/completions`, `POST /chat/stream` |

## Multi-chain / venues

Supported exchange catalog includes Hyperliquid, Binance, Bybit, OKX, Gate, Bitget, KuCoin, Lighter, Aster.  
`GET /api/chains` returns chain/venue metadata (Hyperliquid, Base, Ethereum, CEX venues) plus sample coins (crypto majors + HIP-3).

## AI Market Chat

- Page: `/chat`
- Stream protocol: SSE frames `data: {"type":"...","data":{...}}`
  - `start` · `tool_start` · `tool_result` · `delta` · `error` · `done`
- Tools (read-only): `list_supported_chains`, `list_markets`, `get_klines`, `get_direction_*`, `get_flow_markets`, `get_liquidation_heatmap`, `get_account_snapshot`, `market_brief`
- Manual tool test: `POST /api/chat/tools/test` with `{ "name", "args" }`
- Unit tests: `go test ./internal/api/ -run 'Chat|Normalize|CoinCategory|Arg|Tool'`

## Wallet / Hyperliquid

| Endpoint | Behavior |
|----------|----------|
| `POST /wallet/generate` | Create EVM keypair |
| `POST /wallet/validate` | Derive address from private key |
| `GET /wallet/balance?address=` | Base USDC balance (string) |
| `POST /wallet/export` | Export claw402 model private key |
| `GET /hyperliquid/connect-config` | Strand builder fee config |
| `GET /hyperliquid/account` | Live clearinghouse + spot USDC |
| `GET /hyperliquid/agent` | Agent list + builder fee approval |
| `POST /hyperliquid/submit-exchange` | Proxies signed exchange actions |

FE must send `Authorization: Bearer <jwt>` on all wallet calls.
Preflight exchange checks include `exchange_id` / `exchange_type`. UI maps HL failures to `?setup=hyperliquid` (or `hyperliquid-funds`), and CEX failures to `?setup=exchange&exchange_id=…` (credentials + server IP whitelist). Autopilot launch prefers a ready Hyperliquid account, otherwise the first enabled CEX with credentials.

## Telegram console bot

| Endpoint | Behavior |
|----------|----------|
| `GET /telegram` | Masked token + bind status |
| `POST /telegram` | Save bot token; starts long-poll binder |
| `DELETE /telegram/binding` | Clear chat binding |
| `POST /telegram/model` | Preferred model id for future replies |

After saving a token, open the bot in Telegram and send `/start` to bind. Bound chats receive Autopilot start/stop alerts. Commands: `/status`, `/help`, `/unbind`.

## Beginner onboarding

| Endpoint | Behavior |
|----------|----------|
| `POST /onboarding/beginner` | Find or create an enabled Claw402 model (Base USDC fee wallet); returns address + balance |
| `GET /onboarding/beginner/current` | Lightweight balance poll for the current claw402 wallet (30s cache) |

Creates `{userID}_claw402` with model `deepseek` when none exists. Reuses an existing claw402 key when present.

## M5 extras (shipped)

| Area | Endpoints / UI |
|------|----------------|
| Chat sessions | `GET/POST /api/chat/sessions`, `GET/PUT/DELETE /api/chat/sessions/:id`; stream accepts `session_id` and persists turns |
| Discord / Slack | `GET/POST/DELETE /api/notify/:type` + **runtime alert fan-out** (webhook or bot+channel) aligned with Telegram |
| Strategy backtest | `POST /api/strategies/:id/backtest` — EMA/RSI + RiskControl hard gates + `equity_curve` |
| Competition rules | `GET/PUT /api/competition/rules`; scoring: `pnl_pct` \| `normalized_initial` \| **`window_equity`** |
| Chart-to-order | Portfolio panel → Autopilot seed **or** Trade `compile=1` plan path |
| Trade watchlist | Local prefs (`strand.frontier.prefs.v1`) + star filter on Markets · route `/trade` |
| Console routes | `/markets` · `/trade` · `/portfolio` · `/autopilot` · `/strategy` · `/leaderboard` · `/chat` · `/help` · `/settings` (legacy `/frontier` `/dashboard` `/traders` `/competition` `/faq` redirect) |
| Chat sessions | rename / search (`?q=`), tool turns persisted (`role=tool`) |
| Chat inline chart | `get_klines` tool_result renders `ChatKlineSpark` |
| BYOK usage | Config grid shows “BYOK · usage not tracked” for non-Claw402 models |
| Live checklist | [LAUNCH_CHECKLIST.md](./LAUNCH_CHECKLIST.md) |

## Local verify

```bash
# terminal 1
strand start   # listens :8080 by default

# terminal 2
pnpm --dir web run dev   # :3000, proxies /api → :8080

# main-path API smoke
go test ./internal/api/ -count=1 -run MainPath

# tool smoke (after login token)
curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8080/api/chat/tools
curl -s -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"list_supported_chains","args":{}}' \
  http://127.0.0.1:8080/api/chat/tools/test
```

Flow: open `/` → Setup/register → Settings / Autopilot enable AI model → `/chat` → ask market questions → `/autopilot` → `/portfolio`.
