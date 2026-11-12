# Strand live launch checklist

Use this before giving Strand to someone else (or before your own first live run).
Each item maps to console APIs / UI already in the main path.

## 1. Claw402 (AI fee wallet)

- [ ] Open **Autopilot → Connections** (or **Settings → Models**) and enable a Claw402 model
- [ ] Create or import a Base USDC fee wallet; back up the private key
- [ ] Confirm balance ≥ **$1 USDC** on Base (`GET /api/onboarding/beginner/current` or Autopilot step 1)
- [ ] Optional smoke: `POST /api/models/test` for the claw402 row

## 2. Hyperliquid (wallet authorize)

- [ ] Connect wallet in Autopilot / Hyperliquid setup (approve Agent + builder fee)
- [ ] Deposit trading USDC on Arbitrum into Hyperliquid (≥ **$12** recommended)
- [ ] `POST /api/launch/preflight` shows `exchange_account` + `exchange_trade_auth` green
- [ ] Failures deep-link with `?setup=hyperliquid` or `hyperliquid-funds`

## 3. Binance / other CEX

- [ ] Add API key with **futures trade** permission (no withdrawal)
- [ ] Whitelist Strand server IP (`GET /api/server-ip`)
- [ ] `POST /api/exchanges/test` succeeds; account-state shows equity
- [ ] Preflight annotates checks with `exchange_id` / `exchange_type`

## 4. Strategy & risk gates

- [ ] Activate a strategy on **Strategy**
- [ ] Set daily loss / max drawdown hard gates (saved on next scan)
- [ ] Optional: run **Backtest** on the strategy page (SMA20 smoke)

## 5. Start Autopilot

- [ ] AutopilotLaunchPanel steps 1–3 green → Start
- [ ] **Portfolio** shows equity curve, decisions, pause/resume
- [ ] Telegram bound (`/start`) receives pause / safe-mode alerts

## 6. Optional channels

- [ ] **Chat** → Discord bot token saved (`/api/notify/discord`)
- [ ] **Chat** → Slack webhook saved (`/api/notify/slack`)

## Quick API smoke

```bash
# after login JWT
curl -s -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"ai_model_id":"","exchange_id":""}' \
  http://127.0.0.1:8080/api/launch/preflight | jq .
```

See also: [WEB_API.md](./WEB_API.md), [OPS_RUNBOOK.md](./OPS_RUNBOOK.md).
