# Strand

[English](README.md) | [中文](README.zh-CN.md)

Open-source, **self-hosted AI trading terminal**. Run the console and API on your own machine — keys and funds stay with you, not a third-party custodian.

**Intelligence that can trade. Control that stays yours.**

| You want to… | Go to |
|--------------|--------|
| Browse multi-exchange perps / symbol detail | **Markets** (no login) |
| Compare market / limit / Ladder / TWAP, then confirm | **Trade** |
| Follow the Claw402 direction board automatically | **Autopilot** |
| Watch equity, positions, hard gates, decisions | **Portfolio** |
| Tune leverage / daily-loss hard risk | **Strategy** |
| Compare opt-in instances publicly | **Leaderboard** (browse anonymously) |

Exchanges: **Hyperliquid**, **Binance**, **Bybit**, **OKX**, **Gate**, **Bitget**, and more. AI fee wallet (Base USDC / Claw402) is separate from trading margin.

> **Risk:** Crypto and leveraged trading can lose principal. Strand is software, not investment advice, and does not guarantee returns. See [DISCLAIMER.md](./DISCLAIMER.md).

---

## Architecture

```
┌─────────────────┐     /api/*      ┌──────────────────────┐
│  Web console    │ ───────────────▶│  Strand API (Go)     │
│  Vite / nginx   │                 │  cmd/strand          │
│  :3000          │                 │  :8080               │
└─────────────────┘                 └──────────┬───────────┘
                                               │
                    Public tickers / klines
                    Hyperliquid · CEX · Claw402 / Vergex
                    SQLite (default) or Postgres
```

Locally, Vite proxies `/api` to `localhost:8080`. Docker Compose uses nginx the same way.

---

## Quick start (local)

### Requirements

- **Go 1.26+** (SQLite driver needs CGO / system sqlite)
- **Node 20+** and **pnpm** (web)
- Outbound network for exchange public data and (if used) OpenAI-compatible endpoints; set `HTTPS_PROXY` on restricted networks

### 1. Initialize config

```bash
go run ./cmd/strand config init
# writes ~/.strand/config.toml and ~/.strand/.env (JWT_SECRET / STRAND_API_TOKEN)
```

See also [`config.example.toml`](./config.example.toml) and [`.env.example`](./.env.example).

### 2. Start the API

```bash
go run ./cmd/strand start
# or: ./start.sh
# or: make api
# background: strand start -d   then   strand status / strand stop
```

Default: `http://127.0.0.1:8080`. Run `strand --help` for the full CLI.

### 3. Start the web console

```bash
cd web
pnpm install
pnpm run dev
```

Open `http://127.0.0.1:3000` → register / sign in → **Markets**, or follow [LAUNCH_CHECKLIST](./docs/LAUNCH_CHECKLIST.md) for Autopilot.

Useful Make targets: `make help`, `make deps`, `make build`, `make web`, `make docker-up`.

---

## CLI cheatsheet

```bash
strand --help                 # overview + quick start
strand config init [--dir …]  # create config + .env
strand config show            # masked config dump
strand start [-d]             # foreground or detach
strand stop | status
strand daemon logs -n 100
strand messaging supported
strand messaging add --type telegram --name tg1 --set token=…
strand resource test model <id> --token "$STRAND_API_TOKEN"
```

Global flag: `--config` / `-c` / `$STRAND_CONFIG`.

---

## Docker

Full stack (API + Web):

```bash
docker compose up --build
```

| Service | URL |
|---------|-----|
| Web console | http://127.0.0.1:3000 |
| API (direct) | http://127.0.0.1:8080 |

- Volume `strand-data` → `/data` (config, SQLite, runtime)
- First boot without config runs `strand config init --dir /data`
- API only: `docker build -t strand-api -f docker/Dockerfile.backend .`
- Web only: `docker build -t strand-web -f docker/Dockerfile.frontend .`

For outbound proxy, set `HTTPS_PROXY` / `HTTP_PROXY` in compose or the environment ([OPS_RUNBOOK](./docs/OPS_RUNBOOK.md)).

---

## Product surface (current)

- **Markets** — multi-venue USDT-perp board; `/markets/:exchange/:symbol` detail (candles, convert, peers)
- **Trade** — human-in-the-loop execution planning
- **Autopilot** — Claw402 board + Preflight + hard risk gates
- **Portfolio / Strategy / Leaderboard / Chat / Help / Settings**
- Public examples: `GET /api/symbols`, `GET /api/klines`, `GET /api/competition` (no login)

In-app guides: console **Help** (`/help`).  
Nav map: [docs/WEB_TOP_NAV.md](./docs/WEB_TOP_NAV.md)  
Product intro (中文): [docs/PRODUCT_INTRO.md](./docs/PRODUCT_INTRO.md)  
Console API: [docs/WEB_API.md](./docs/WEB_API.md)  
Ops: [docs/OPS_RUNBOOK.md](./docs/OPS_RUNBOOK.md) · Security: [SECURITY.md](./SECURITY.md)

---

## Repository layout

```
cmd/strand/          CLI entry (config / start / stop / status / messaging …)
internal/            API, trading, market data, strategy, messaging bridges
web/                 React console (Vite + pnpm)
docker/              Dockerfiles, nginx, entrypoint
docs/                Product & ops docs
config.example.toml  Config sample
.env.example         Secret env sample
```

---

## Security & secrets

- **Do not** commit real `.env`, wallet keys, or exchange API keys (see `.gitignore`)
- Hyperliquid Agent can trade but cannot withdraw; CEX keys should disable withdrawal and use IP allowlists
- Default backups omit secrets; secret exports must be stored offline (Settings / [OPS_RUNBOOK](./docs/OPS_RUNBOOK.md))

---

## License

[MIT](./LICENSE) © Strand
