import { useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { Link } from 'react-router'
import type { CSSProperties } from 'react'
import useSWR, { mutate } from 'swr'
import { api } from '../../lib/api'
import { confirmToast, notify } from '../../lib/notify'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import type {
  SystemStatus,
  AccountInfo,
  Position,
  DecisionRecord,
  TraderInfo,
  ExchangeAccountState,
} from '../../types'
import { OrchestrationTopology } from './OrchestrationTopology'
import { OrderBook } from './OrderBook'
import { LiquidationMap } from './LiquidationMap'
import { WinrateMatrix } from './WinrateMatrix'
import { KlineChart } from './KlineChart'
import { ExecutionLog } from './ExecutionLog'
import { FailureAttributionPanel } from './FailureAttributionPanel'
import { EquityChart } from '../charts/EquityChart'
import { ChartToOrderPanel } from '../charts/ChartToOrderPanel'
import { SignalMatrix } from './SignalMatrix'
import { RiskRadar } from './RiskRadar'
import { EdgeProfile } from './EdgeProfile'
import { GridRiskPanel } from '../strategy/GridRiskPanel'
import { PositionHistory } from '../trader/PositionHistory'
import { useDemoEngine } from '../../lib/demo/useDemoEngine'
import { ROUTES } from '../../router/paths'

// crypto majors trade on the Hyperliquid main dex (no hip3 cost/liq heatmap);
// everything else in the universe is an xyz-dex synthetic market that does.
const CRYPTO_MAJORS = new Set([
  'BTC',
  'ETH',
  'SOL',
  'HYPE',
  'BNB',
  'XRP',
  'DOGE',
  'AVAX',
  'LINK',
  'SUI',
  'APT',
  'ARB',
  'OP',
  'TON',
  'ADA',
  'TRX',
  'LTC',
  'BCH',
  'NEAR',
  'INJ',
  'SEI',
  'TIA',
  'PEPE',
  'WIF',
  'BONK',
  'AAVE',
  'UNI',
  'ENA',
  'ONDO',
  'JUP',
  'PENDLE',
  'KPEPE',
  'ZEC',
  'XPL',
  'LIT',
])

// fixed height for the three row-1 panels so the row stays balanced at any width
const ROW1_H = 500
import { FlowMarkets } from './FlowMarkets'
import './terminal.css'

interface TerminalDashboardProps {
  selectedTrader?: TraderInfo
  traders?: TraderInfo[]
  selectedTraderId?: string
  onTraderSelect: (traderId: string) => void
  status?: SystemStatus
  account?: AccountInfo
  positions?: Position[]
  decisions?: DecisionRecord[]
  /** True when account polling gave up after retries. */
  accountFailed?: boolean
  onRetryAccount?: () => void
  /** True when positions polling gave up after retries. */
  positionsFailed?: boolean
  onRetryPositions?: () => void
}

function fmtUsd(n: number | undefined, signed = false): string {
  if (n == null || Number.isNaN(n)) return '—'
  const sign = signed && n > 0 ? '+' : n < 0 ? '-' : ''
  return `${sign}$${Math.abs(n).toLocaleString('en-US', { maximumFractionDigits: 2 })}`
}
function fmtPct(n: number | undefined): string {
  if (n == null || Number.isNaN(n)) return '—'
  return `${n >= 0 ? '+' : ''}${n.toFixed(2)}%`
}
/** Price with magnitude-aware precision: 64,416 · 184.2 · 2.3775 · 0.0067 */
function fmtPx(n: number | undefined): string {
  if (n == null || Number.isNaN(n) || n === 0) return '—'
  const dp = n >= 1000 ? 0 : n >= 100 ? 1 : n >= 1 ? 2 : 4
  return n.toLocaleString('en-US', {
    minimumFractionDigits: dp,
    maximumFractionDigits: dp,
  })
}
function baseLabel(raw?: string): string {
  if (!raw) return ''
  return raw
    .toUpperCase()
    .replace(/^XYZ:/, '')
    .replace(/[-_]/g, '')
    .replace(/(USDT|USDC|USD)$/, '')
}
function parseScanMinutes(scan?: string): number {
  if (!scan) return 15
  const m = scan.match(/(\d+)\s*m/i)
  if (m) return parseInt(m[1], 10)
  const h = scan.match(/(\d+)\s*h/i)
  if (h) return parseInt(h[1], 10) * 60
  const n = parseInt(scan, 10)
  return Number.isFinite(n) && n > 0 ? n : 15
}
function fmtTime(raw?: string | number): string {
  if (raw == null || raw === '') return ''
  let n = typeof raw === 'number' ? raw : Number(raw)
  if (Number.isFinite(n)) {
    if (n < 1e12) n *= 1000
    return new Date(n).toLocaleString('en-GB', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    })
  }
  const d = new Date(raw as string)
  return Number.isNaN(d.getTime())
    ? ''
    : d.toLocaleString('en-GB', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      })
}

/** Hold duration from entry/exit epoch-ms as a compact 45m / 2h10 / 1d3h. */
function fmtHold(entry?: number, exit?: number): string {
  if (!entry || !exit || exit <= entry) return '—'
  const mins = Math.round((exit - entry) / 60000)
  if (mins < 60) return `${mins}m`
  const h = Math.floor(mins / 60)
  const m = mins % 60
  if (h < 24) return m ? `${h}h${m}` : `${h}h`
  const d = Math.floor(h / 24)
  return `${d}d${h % 24}h`
}

function useTick(ms = 1000) {
  const [, set] = useState(0)
  useEffect(() => {
    const id = setInterval(() => set((n) => n + 1), ms)
    return () => clearInterval(id)
  }, [ms])
}

export function TerminalDashboard({
  selectedTrader,
  traders,
  selectedTraderId,
  onTraderSelect,
  status: propStatus,
  account: propAccount,
  positions: propPositions,
  decisions: propDecisions,
  accountFailed = false,
  onRetryAccount,
  positionsFailed = false,
  onRetryPositions,
}: TerminalDashboardProps) {
  const { language } = useLanguage()
  const traderId = selectedTrader?.trader_id || selectedTraderId
  const exchangeId = selectedTrader?.exchange_id
  useTick(1000)
  const clock = new Date().toLocaleTimeString('en-GB', { hour12: false })
  const [closing, setClosing] = useState<string | null>(null)

  const { data: exchangeStates, isLoading: exchangeStateLoading } = useSWR(
    exchangeId ? ['exchange-account-state', exchangeId] : null,
    async () => {
      const res = await api.getExchangeAccountState()
      return res.states || {}
    },
    { refreshInterval: 60000, shouldRetryOnError: false }
  )
  const exchangeState: ExchangeAccountState | undefined = exchangeId
    ? exchangeStates?.[exchangeId]
    : undefined
  const exchangeSetupHref = exchangeId
    ? `${ROUTES.traders}?setup=exchange&exchange_id=${encodeURIComponent(exchangeId)}`
    : `${ROUTES.traders}?setup=exchange`

  async function closePositionRow(symbol: string, side: 'LONG' | 'SHORT') {
    if (!traderId || closing) return
    const ok = await confirmToast(`Market-close ${symbol} ${side}?`, {
      title: 'Close position',
      okText: 'Close',
      cancelText: 'Cancel',
    })
    if (!ok) return
    setClosing(symbol)
    try {
      await api.closePosition(traderId, symbol, side)
      notify.success(`${symbol} ${side} closed`)
      await Promise.all([
        mutate(`positions-${traderId}`),
        mutate(`account-${traderId}`),
      ])
    } catch (err) {
      notify.error(err instanceof Error ? err.message : 'Close failed')
    } finally {
      setClosing(null)
    }
  }

  async function closeAllPositions(open: Position[]) {
    if (!traderId || closing || open.length === 0) return
    const ok = await confirmToast(
      `Market-close ALL ${open.length} open positions?`,
      { title: 'Flatten book', okText: 'Close all', cancelText: 'Cancel' }
    )
    if (!ok) return
    setClosing('__all__')
    let failed = 0
    // Sequential: parallel closes race on exchange nonces / rate limits.
    for (const p of open) {
      const side = /long|buy/i.test(p.side) ? 'LONG' : 'SHORT'
      try {
        await api.closePosition(traderId, p.symbol, side)
      } catch {
        failed++
      }
    }
    await Promise.all([
      mutate(`positions-${traderId}`),
      mutate(`account-${traderId}`),
    ])
    if (failed === 0) notify.success('All positions closed')
    else notify.error(`${failed}/${open.length} closes failed`)
    setClosing(null)
  }

  const { data: realFullStats } = useSWR(
    traderId ? ['full-stats', traderId] : null,
    () => api.getFullStats(traderId!, true),
    { refreshInterval: 30000, shouldRetryOnError: false }
  )
  const { data: realHistory } = useSWR(
    traderId ? ['pos-history', traderId] : null,
    () => api.getPositionHistory(traderId!, 50, true),
    { refreshInterval: 60000, shouldRetryOnError: false }
  )
  const { data: realConfig } = useSWR(
    traderId ? ['trader-config', traderId] : null,
    () => api.getTraderConfig(traderId!, true),
    { refreshInterval: 120000, shouldRetryOnError: false }
  )
  const { data: realFlow } = useSWR(
    traderId ? ['flow-markets', traderId] : null,
    () =>
      api.getFlowMarkets(selectedTrader?.ai_model, 'mainnet', '1h', 50, true),
    // paid x402 endpoint — poll slowly (5m) to conserve claw402 funds; the
    // topology beam animation is client-side and stays fast regardless
    { refreshInterval: 300000, shouldRetryOnError: false }
  )
  const { data: realSignalRank } = useSWR(
    traderId ? ['signal-rank', traderId] : null,
    () => api.getDirectionChangeLeaderboard(30, true),
    // paid x402 endpoint — poll slowly (5m) to conserve claw402 funds
    { refreshInterval: 300000, shouldRetryOnError: false }
  )

  // Demo / showcase mode for product walkthroughs. Toggle with Shift+D (or the
  // discreet corner dot). Generates a fast, profitable-looking US-equity dataset
  // entirely client-side — it never touches the backend or any real account.
  // When off, real data flows through unchanged.
  const [demo, setDemo] = useState(false)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.shiftKey && (e.key === 'D' || e.key === 'd')) {
        const el = document.activeElement
        if (el && /^(input|textarea|select)$/i.test(el.tagName)) return
        setDemo((v) => !v)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])
  const sim = useDemoEngine(demo)
  const on = demo && !!sim

  const status = on ? sim!.status : propStatus
  const account = on ? sim!.account : propAccount
  const positions = on ? sim!.positions : propPositions
  const decisions = on ? sim!.decisions : propDecisions
  const fullStats = on ? sim!.fullStats : realFullStats
  const history = on ? sim!.history : realHistory
  const config = on ? (sim!.config as unknown as typeof realConfig) : realConfig
  const flow = on ? sim!.flow : realFlow
  const signalRank = on ? sim!.signalRank : realSignalRank

  const latest = decisions && decisions.length > 0 ? decisions[0] : undefined
  const candidateCoins = latest?.candidate_coins ?? []
  const flowItems = flow?.data?.inflow ?? []

  // Both the cost/liq map and the order book follow this symbol so they stay in
  // sync. The heatmap only covers hip3_perp synthetic markets, so we pick a
  // synthetic (non-crypto) the bot trades — preferring the BUSIEST one (most
  // 1h trades, per flow-markets) so the shared order book ticks as fast as
  // possible. Falls back to any held synthetic, then SP500.
  const heatmapSymbol = useMemo(() => {
    const held = new Set(
      [...(positions ?? []).map((p) => p.symbol), ...candidateCoins]
        .map(baseLabel)
        .filter((b) => b && !CRYPTO_MAJORS.has(b))
    )
    const synthByActivity = flowItems
      .map((i) => ({ b: baseLabel(i.symbol), trades: i.trades || 0 }))
      .filter((x) => x.b && !CRYPTO_MAJORS.has(x.b))
      .sort((a, b) => b.trades - a.trades)
    const busiestHeld = synthByActivity.find((x) => held.has(x.b))
    if (busiestHeld) return busiestHeld.b
    if (held.size) return [...held][0]
    if (synthByActivity.length) return synthByActivity[0].b
    return 'SP500'
  }, [positions, candidateCoins, flowItems])

  // user can click a signal-matrix cell to drive both the cost/liq map and the
  // order book. Default to the instrument the bot is ACTUALLY holding (first
  // open position, else this cycle's first candidate) so the price panels match
  // the real traded symbol; only fall back to the busiest synthetic if the bot
  // holds nothing.
  const [selectedSym, setSelectedSym] = useState<string | null>(null)
  const defaultSym = useMemo(() => {
    // the bot's actual first open position (else this cycle's first candidate);
    // every market — synthetic or crypto — now has a cost/liq heatmap, so no
    // need to prefer one type. Falls back to the busiest synthetic if flat.
    const heldBases = [
      ...(positions ?? []).map((p) => p.symbol),
      ...candidateCoins,
    ]
      .map(baseLabel)
      .filter(Boolean)
    return heldBases[0] || heatmapSymbol || 'SP500'
  }, [positions, candidateCoins, heatmapSymbol])
  const activeSym = baseLabel(selectedSym || defaultSym)
  const activeSignal = signalRank?.items?.find(
    (item) => baseLabel(item.symbol) === activeSym
  )
  const sourceSymbol =
    activeSignal?.symbol ||
    [...(positions ?? []).map((p) => p.symbol), ...candidateCoins].find(
      (s) => baseLabel(s) === activeSym
    ) ||
    selectedSym ||
    defaultSym
  const activeMarketType =
    activeSignal?.market_type === 'core_perp' ||
    activeSignal?.market_type === 'hip3_perp'
      ? activeSignal.market_type
      : /^xyz:/i.test(sourceSymbol)
        ? 'hip3_perp'
        : /USDT$/i.test(sourceSymbol) || CRYPTO_MAJORS.has(activeSym)
          ? 'core_perp'
          : 'hip3_perp'
  const activeRequestSymbol =
    activeMarketType === 'hip3_perp' ? `xyz:${activeSym}` : activeSym

  const pnl = account?.total_pnl ?? 0
  const pnlPct = account?.total_pnl_pct ?? 0
  const up = pnl >= 0
  const running = status?.is_running

  // direction per symbol — priority: AI's actual decision > signal bias >
  // net flow > prevailing market majority (never blindly default to long).
  const dirFor = useMemo(() => {
    const dec = new Map<string, 'long' | 'short'>()
    ;(latest?.decisions ?? []).forEach((d) => {
      const b = baseLabel(d.symbol)
      if (d.action === 'open_long' || d.action === 'close_short')
        dec.set(b, 'long')
      else if (d.action === 'open_short' || d.action === 'close_long')
        dec.set(b, 'short')
    })
    const sig = new Map<string, 'long' | 'short'>()
    let bull = 0
    let bear = 0
    ;(signalRank?.items ?? []).forEach((s) => {
      const b = baseLabel(s.symbol)
      const bias = (s.bias || '').toLowerCase()
      if (bias === 'bearish') {
        sig.set(b, 'short')
        bear++
      } else if (bias === 'bullish') {
        sig.set(b, 'long')
        bull++
      }
    })
    const fl = new Map<string, 'long' | 'short'>()
    ;(flow?.data?.inflow ?? []).forEach((i) =>
      fl.set(baseLabel(i.symbol), 'long')
    )
    ;(flow?.data?.outflow ?? []).forEach((i) =>
      fl.set(baseLabel(i.symbol), 'short')
    )
    const majority: 'long' | 'short' = bear > bull ? 'short' : 'long'
    return (sym: string): 'long' | 'short' => {
      const b = baseLabel(sym)
      return dec.get(b) ?? sig.get(b) ?? fl.get(b) ?? majority
    }
  }, [latest, signalRank, flow])

  const scanMin =
    config?.scan_interval_minutes || parseScanMinutes(status?.scan_interval)
  const nextCycleMs = useMemo(() => {
    if (!latest?.timestamp) return null
    return new Date(latest.timestamp).getTime() + scanMin * 60_000
  }, [latest?.timestamp, scanMin])
  let countdown = '—'
  if (nextCycleMs) {
    const ms = nextCycleMs - Date.now()
    if (ms <= 0) countdown = 'due now'
    else {
      const s = Math.floor(ms / 1000)
      countdown = `${Math.floor(s / 60)}m ${s % 60}s`
    }
  }

  // Live hard-gate progress vs strategy limits (same math as Autopilot).
  const equityNow = account?.total_equity ?? 0
  const dayStartEq = status?.day_start_equity ?? 0
  const peakEq = status?.peak_equity ?? 0
  const dayLossPct =
    dayStartEq > 0 && equityNow > 0
      ? Math.max(0, ((dayStartEq - equityNow) / dayStartEq) * 100)
      : null
  const liveDrawdownPct =
    peakEq > 0 && equityNow > 0
      ? Math.max(0, ((peakEq - equityNow) / peakEq) * 100)
      : null
  const dailyLossLimit =
    config?.daily_loss_limit_pct ?? status?.daily_loss_limit_pct ?? 0
  const drawdownLimit =
    config?.max_drawdown_limit_pct ?? status?.max_drawdown_limit_pct ?? 0

  const recentTrades = (history?.positions ?? []).slice(0, 8)
  const [replayTradeId, setReplayTradeId] = useState<number | null>(null)
  const [showFullHistory, setShowFullHistory] = useState(false)
  const decisionByTrade = useMemo(() => {
    const list = decisions ?? []
    const map = new Map<number, DecisionRecord>()
    for (const trade of recentTrades) {
      const exitMs =
        trade.exit_time > 1e12 ? trade.exit_time : trade.exit_time * 1000
      const base = baseLabel(trade.symbol)
      let best: DecisionRecord | undefined
      let bestDelta = Number.POSITIVE_INFINITY
      for (const d of list) {
        const ts = new Date(d.timestamp).getTime()
        if (Number.isNaN(ts)) continue
        const delta = Math.abs(ts - exitMs)
        // Match closes that land in the same scan window (±90m).
        if (delta > 90 * 60_000) continue
        const hit = (d.decisions ?? []).some((a) => {
          if (baseLabel(a.symbol) !== base) return false
          const act = (a.action || '').toLowerCase()
          return (
            act.includes('close') ||
            act.includes('reduce') ||
            act === 'hold' ||
            act === 'wait'
          )
        })
        if (!hit && !(d.candidate_coins ?? []).some((c) => baseLabel(c) === base))
          continue
        if (delta < bestDelta) {
          bestDelta = delta
          best = d
        }
      }
      if (best) map.set(trade.id, best)
    }
    return map
  }, [decisions, recentTrades])
  const symbolStats = useMemo(
    () =>
      (history?.symbol_stats ?? [])
        .slice()
        .sort((a, b) => b.total_trades - a.total_trades)
        .slice(0, 6),
    [history]
  )
  const maxSymTrades = symbolStats.reduce(
    (m, s) => Math.max(m, s.total_trades),
    1
  )

  const sc: CSSProperties = { padding: '10px 14px' }
  const cellBorder = '1px solid var(--tm-hair)'

  // Portal the trader selector + run status into the global nav so the app has
  // a single top bar (no separate dashboard titlebar).
  const [navSlot, setNavSlot] = useState<HTMLElement | null>(null)
  useEffect(() => {
    setNavSlot(document.getElementById('dash-header-slot'))
  }, [])

  return (
    <div
      className="strand-terminal strand-dashboard"
      style={{ minHeight: '100vh', padding: 0 }}
    >
      {/* discreet showcase toggle (Shift+D also works) */}
      <button
        type="button"
        onClick={() => setDemo((v) => !v)}
        aria-label="toggle presentation mode"
        title="Shift+D · presentation demo"
        style={{
          position: 'fixed',
          right: 10,
          bottom: 10,
          zIndex: 9999,
          width: 12,
          height: 12,
          padding: 0,
          borderRadius: '50%',
          border: 'none',
          cursor: 'pointer',
          background: on ? 'var(--tm-up)' : 'color-mix(in srgb, var(--tm-ink) 18%, transparent)',
          opacity: on ? 0.55 : 0.18,
        }}
      />
      {on ? (
        <div
          className="tm-mono"
          role="status"
          style={{
            margin: '8px 14px 0',
            padding: '8px 12px',
            fontSize: 11,
            border: '1px dashed var(--tm-up)',
            color: 'var(--tm-up)',
            background: 'rgba(46,160,67,0.08)',
            display: 'flex',
            gap: 10,
            alignItems: 'center',
            flexWrap: 'wrap',
          }}
        >
          <strong>DEMO</strong>
          <span style={{ color: 'var(--tm-ink-2)' }}>
            Synthetic showcase data — not your live account. Press Shift+D to
            exit.
          </span>
          <button
            type="button"
            className="tm-mono"
            onClick={() => setDemo(false)}
            style={{
              marginLeft: 'auto',
              border: '1px solid var(--tm-up)',
              color: 'var(--tm-up)',
              background: 'transparent',
              padding: '2px 8px',
              cursor: 'pointer',
            }}
          >
            Exit demo
          </button>
        </div>
      ) : null}
      {/* centered, capped content column — no border (keeps it from feeling
          embedded) but bounded so the aspect-ratio SVGs don't balloon on wide screens */}
      {navSlot &&
        createPortal(
          <span
            className="strand-terminal"
            style={{
              background: 'transparent',
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              marginLeft: 16,
              paddingLeft: 16,
              borderLeft: '1px solid var(--tm-hair)',
              fontSize: 11,
            }}
          >
            <span className="tm-sc" style={{ color: 'var(--tm-muted)' }}>
              orchestration
            </span>
            {traders && traders.length > 0 && (
              <select
                value={traderId || ''}
                onChange={(e) => onTraderSelect(e.target.value)}
                className="tm-mono"
                style={{
                  background: 'var(--tm-panel)',
                  color: 'var(--tm-ink)',
                  border: '1px solid var(--tm-hair)',
                  borderRadius: 0,
                  fontSize: 11,
                  padding: '3px 6px',
                }}
              >
                {traders.map((t) => (
                  <option
                    key={t.trader_id}
                    value={t.trader_id}
                    style={{ color: 'var(--tm-ink)' }}
                  >
                    {t.trader_name}
                  </option>
                ))}
              </select>
            )}
            <span
              style={{
                color: status?.trading_blocked
                  ? 'var(--tm-dn)'
                  : running
                    ? 'var(--tm-up)'
                    : 'var(--tm-muted)',
              }}
            >
              {status?.trading_blocked
                ? '● trading blocked'
                : running
                  ? '● running'
                  : '○ stopped'}
            </span>
            <span className="tm-sc" style={{ color: 'var(--tm-muted)' }}>
              cycle
            </span>
            <span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>
              {status?.call_count ?? '—'}
            </span>
            <span
              className="tm-px"
              style={{ fontSize: 12, color: 'var(--tm-ink)' }}
            >
              {clock}
            </span>
          </span>,
          navSlot
        )}
      <div className="strand-dashboard-heading">
        <div>
          <span className="strand-dashboard-eyebrow">
            {t('pagePortfolioEyebrow', language)}
          </span>
          <h1>{t('pagePortfolioTitle', language)}</h1>
          <p>{t('pagePortfolioDesc', language)}</p>
        </div>
        <span>{t('pagePortfolioBadge', language)}</span>
      </div>
      <div
        className="tm-box strand-dashboard-body"
        style={{ maxWidth: 1280, margin: '0 auto', border: 'none' }}
      >
        {!on && (!traders || traders.length === 0) && (
          <div
            className="tm-mono"
            style={{
              margin: '8px 14px 0',
              padding: '16px 14px',
              border: '1px solid var(--tm-hair)',
              background: 'rgba(41,91,232,0.05)',
            }}
          >
            <div style={{ fontWeight: 700, marginBottom: 6, color: 'var(--tm-ink)' }}>
              {t('noAutopilotYet', language)}
            </div>
            <div className="tm-sc" style={{ marginBottom: 12, color: 'var(--tm-ink-2)' }}>
              {t('noAutopilotHint', language)}
            </div>
            <Link
              to={`${ROUTES.traders}?view=autopilot`}
              className="tm-mono"
              style={{
                display: 'inline-block',
                padding: '6px 12px',
                border: '1px solid var(--tm-up)',
                color: 'var(--tm-up)',
                textDecoration: 'none',
              }}
            >
              {t('openAutopilotSetup', language)}
            </Link>
          </div>
        )}
        {!on && (
          <FailureAttributionPanel
            status={status}
            decisions={decisions}
            language={language}
          />
        )}
        {accountFailed && (
          <div
            role="alert"
            className="tm-mono"
            style={{
              margin: '8px 14px 0',
              padding: '10px 12px',
              fontSize: 11,
              border: '1px solid var(--tm-dn)',
              color: 'var(--tm-dn)',
              background: 'rgba(200,60,40,0.06)',
              display: 'flex',
              gap: 10,
              alignItems: 'center',
              flexWrap: 'wrap',
            }}
          >
            <span>
              {t('accountApiStale', language)}
            </span>
            {onRetryAccount ? (
              <button
                type="button"
                className="tm-mono"
                onClick={onRetryAccount}
                style={{
                  marginLeft: 'auto',
                  border: '1px solid var(--tm-dn)',
                  color: 'var(--tm-dn)',
                  background: 'transparent',
                  padding: '3px 8px',
                  cursor: 'pointer',
                }}
              >
                {t('retry', language)}
              </button>
            ) : null}
          </div>
        )}
        {/* runtime health banner — AI fee wallet dry / safe mode would otherwise
            only be visible in server logs while the bot silently idles */}
        {!on &&
          status &&
          (status.safe_mode ||
            status.paused ||
            status.ai_wallet_status === 'empty' ||
            status.ai_wallet_status === 'low') && (
            <div
              className="tm-mono"
              style={{
                display: 'flex',
                gap: 10,
                alignItems: 'center',
                margin: '8px 14px 0',
                padding: '8px 12px',
                fontSize: 11,
                border: '1px solid var(--tm-down)',
                color: 'var(--tm-down)',
                background: 'rgba(200,60,40,0.06)',
                flexWrap: 'wrap',
              }}
            >
              <span style={{ fontWeight: 600 }}>
                {status.ai_wallet_status === 'empty'
                  ? language === 'zh'
                    ? 'AI 手续费钱包 USDC 不足 — 决策正在失败。'
                    : 'AI fee wallet is out of USDC — decisions are failing.'
                  : status.ai_wallet_status === 'low'
                    ? language === 'zh'
                      ? `AI 手续费钱包余额偏低（${(status.ai_wallet_balance_usdc ?? 0).toFixed(2)} USDC）— 请尽快充值。`
                      : `AI fee wallet is low (${(status.ai_wallet_balance_usdc ?? 0).toFixed(2)} USDC) — top up soon.`
                    : status.paused
                      ? language === 'zh'
                        ? '交易已暂停 — 恢复或到期前不开新仓。'
                        : 'Trading paused — no new positions until resume or stop_until.'
                      : language === 'zh'
                        ? '安全模式：AI 连续失败，当前不开新仓。'
                        : 'Safe mode: AI failed repeatedly, no new positions are being opened.'}
              </span>
              <span style={{ color: 'var(--tm-ink-2)' }}>
                {status.ai_wallet_status === 'empty' ||
                status.ai_wallet_status === 'low'
                  ? language === 'zh'
                    ? '向 Claw402 钱包充值 Base USDC 后，交易员会自动恢复。'
                    : 'Deposit Base USDC to the Claw402 wallet, the trader recovers automatically.'
                  : status.pause_reason || status.safe_mode_reason || ''}
              </span>
              {traderId && status.is_running && (
                <span style={{ display: 'flex', gap: 8, marginLeft: 'auto' }}>
                  {status.paused ? (
                    <button
                      type="button"
                      className="tm-mono"
                      style={{
                        border: '1px solid var(--tm-up)',
                        color: 'var(--tm-up)',
                        background: 'transparent',
                        padding: '4px 10px',
                        cursor: 'pointer',
                      }}
                      onClick={() => {
                        void api
                          .resumeTrader(traderId)
                          .then(() => {
                            notify.success(
                              t('tradingResumed', language)
                            )
                            void mutate(
                              (key) =>
                                typeof key === 'string' &&
                                key.includes('/status')
                            )
                          })
                          .catch((err) =>
                            notify.error(
                              err instanceof Error
                                ? err.message
                                : language === 'zh'
                                  ? '恢复失败'
                                  : 'Resume failed'
                            )
                          )
                      }}
                    >
                      {language === 'zh' ? '恢复' : 'Resume'}
                    </button>
                  ) : (
                    <button
                      type="button"
                      className="tm-mono"
                      style={{
                        border: '1px solid var(--tm-down)',
                        color: 'var(--tm-down)',
                        background: 'transparent',
                        padding: '4px 10px',
                        cursor: 'pointer',
                      }}
                      onClick={() => {
                        void api
                          .pauseTrader(traderId, {
                            hours: 24,
                            reason: 'Paused from Portfolio',
                          })
                          .then(() => {
                            notify.success(
                              language === 'zh'
                                ? '已暂停交易 24 小时'
                                : 'Trading paused for 24h'
                            )
                            void mutate(
                              (key) =>
                                typeof key === 'string' &&
                                key.includes('/status')
                            )
                          })
                          .catch((err) =>
                            notify.error(
                              err instanceof Error
                                ? err.message
                                : language === 'zh'
                                  ? '暂停失败'
                                  : 'Pause failed'
                            )
                          )
                      }}
                    >
                      {language === 'zh' ? '暂停 24h' : 'Pause 24h'}
                    </button>
                  )}
                </span>
              )}
            </div>
          )}
        {!on && status?.is_running && !status.paused && !status.safe_mode && traderId && (
          <div
            className="tm-mono"
            style={{
              display: 'flex',
              gap: 10,
              alignItems: 'center',
              justifyContent: 'flex-end',
              margin: '8px 14px 0',
              padding: '4px 0',
              fontSize: 11,
            }}
          >
            <button
              type="button"
              className="tm-mono"
              style={{
                border: '1px solid var(--tm-ink-2)',
                color: 'var(--tm-ink-2)',
                background: 'transparent',
                padding: '4px 10px',
                cursor: 'pointer',
              }}
              onClick={() => {
                void api
                  .pauseTrader(traderId, {
                    hours: 24,
                    reason: 'Paused from Portfolio',
                  })
                  .then(() => {
                    notify.success(
                      language === 'zh'
                        ? '已暂停交易 24 小时'
                        : 'Trading paused for 24h'
                    )
                    void mutate(
                      (key) =>
                        typeof key === 'string' && key.includes('/status')
                    )
                  })
                  .catch((err) =>
                    notify.error(
                      err instanceof Error
                        ? err.message
                        : language === 'zh'
                          ? '暂停失败'
                          : 'Pause failed'
                    )
                  )
              }}
            >
              {language === 'zh' ? '暂停交易 24h' : 'Pause trading 24h'}
            </button>
          </div>
        )}
        {/* first-run reassurance — a fresh autopilot looks idle for its first
            minute (the AI is reading the market); tell newcomers what to expect */}
        {!on &&
          status?.is_running &&
          (status.call_count ?? 0) <= 1 &&
          !status.safe_mode &&
          !status.trading_blocked &&
          status.trading_checked_at && (
            <div
              className="tm-mono"
              style={{
                display: 'flex',
                gap: 10,
                alignItems: 'center',
                margin: '8px 14px 0',
                padding: '8px 12px',
                fontSize: 11,
                border: '1px solid var(--tm-up)',
                color: 'var(--tm-ink)',
                background: 'rgba(40,140,80,0.06)',
                flexWrap: 'wrap',
              }}
            >
              <span style={{ fontWeight: 600, color: 'var(--tm-up)' }}>
                Your AI is live.
              </span>
              <span style={{ color: 'var(--tm-ink-2)' }}>
                It reads the whole market before acting — the first decision
                usually lands within a minute or two and will appear in the
                Execution Log below. You can stop or pause anytime from
                Portfolio or Autopilot.
              </span>
            </div>
          )}
        {/* config / identity strip — first row, flows directly under the global nav */}
        <div
          className="tm-mono"
          style={{
            display: 'flex',
            gap: 16,
            padding: '6px 14px',
            fontSize: 11,
            color: 'var(--tm-ink-2)',
            flexWrap: 'wrap',
          }}
        >
          <span style={{ fontWeight: 500 }}>
            {selectedTrader?.trader_name ?? 'Strand'}
          </span>
          <span>
            <span className="tm-sc">
              {language === 'zh' ? '模型 ' : 'model '}
            </span>
            {(() => {
              const raw = config?.ai_model || status?.ai_model || ''
              if (!raw) return '—'
              if (/claw402/i.test(raw)) return 'CLAW402'
              return raw.length > 16
                ? raw.slice(0, 16).toUpperCase()
                : raw.toUpperCase()
            })()}
          </span>
          <span>
            <span className="tm-sc">
              {language === 'zh' ? '交易所 ' : 'venue '}
            </span>
            {exchangeStateLoading && !exchangeState ? (
              '…'
            ) : exchangeState ? (
              <Link
                to={exchangeSetupHref}
                style={{
                  color:
                    exchangeState.status === 'ok'
                      ? 'var(--tm-up)'
                      : 'var(--tm-dn)',
                  textDecoration: 'underline',
                  textUnderlineOffset: 2,
                }}
                title={
                  exchangeState.error_message ||
                  exchangeState.status ||
                  'Exchange account'
                }
              >
                {exchangeState.status === 'ok'
                  ? exchangeState.display_balance ||
                    (exchangeState.total_equity != null
                      ? `${exchangeState.total_equity.toFixed(2)}${
                          exchangeState.asset ? ` ${exchangeState.asset}` : ''
                        }`
                      : 'ok')
                  : exchangeState.status.replace(/_/g, ' ')}
              </Link>
            ) : exchangeId ? (
              <Link
                to={exchangeSetupHref}
                style={{
                  color: 'var(--tm-muted)',
                  textDecoration: 'underline',
                  textUnderlineOffset: 2,
                }}
              >
                check
              </Link>
            ) : (
              '—'
            )}
          </span>
          <span>
            <span className="tm-sc">
              {language === 'zh' ? '策略 ' : 'strategy '}
            </span>
            {config?.strategy_name || selectedTrader?.strategy_name || '—'}
          </span>
          <span>
            <span className="tm-sc">
              {language === 'zh' ? '杠杆 ' : 'lev '}
            </span>
            {config?.btc_eth_leverage ?? status?.btc_eth_leverage ?? '—'}× /{' '}
            {config?.altcoin_leverage ?? status?.altcoin_leverage ?? '—'}×
          </span>
          <span>
            <span className="tm-sc">
              {language === 'zh' ? '保证金 ' : 'margin '}
            </span>
            {account?.margin_used_pct != null
              ? `${account.margin_used_pct.toFixed(0)}%`
              : '—'}
            {' / '}
            {(config?.max_margin_usage ?? 0) > 0
              ? `${(
                  (config!.max_margin_usage! <= 1.5
                    ? config!.max_margin_usage! * 100
                    : config!.max_margin_usage!) as number
                ).toFixed(0)}%`
              : 'off'}
          </span>
          <span>
            <span className="tm-sc">
              {language === 'zh' ? '日亏 ' : 'day loss '}
            </span>
            {dayLossPct != null ? `${dayLossPct.toFixed(1)}%` : '—'}
            {' / '}
            {(config?.daily_loss_limit_pct ??
              status?.daily_loss_limit_pct ??
              0) > 0
              ? `${(config?.daily_loss_limit_pct ?? status?.daily_loss_limit_pct)!.toFixed(0)}%`
              : 'off'}
          </span>
          <span>
            <span className="tm-sc">
              {language === 'zh' ? '回撤闸 ' : 'dd gate '}
            </span>
            {liveDrawdownPct != null ? `${liveDrawdownPct.toFixed(1)}%` : '—'}
            {' / '}
            {(config?.max_drawdown_limit_pct ??
              status?.max_drawdown_limit_pct ??
              0) > 0
              ? `${(config?.max_drawdown_limit_pct ?? status?.max_drawdown_limit_pct)!.toFixed(0)}%`
              : 'off'}
          </span>
          <span>
            <span className="tm-sc">scan </span>
            {scanMin}m
          </span>
          <span>
            <span className="tm-sc">universe </span>
            {candidateCoins.length}
          </span>
          <span>
            <span className="tm-sc">positions </span>
            {positions?.length ?? 0}
            {(config?.max_positions ?? status?.max_positions ?? 0) > 0
              ? ` / ${config?.max_positions ?? status?.max_positions}`
              : ''}
          </span>
          {config?.strategy_id ? (
            <Link
              to={`${ROUTES.strategy}?id=${encodeURIComponent(config.strategy_id)}&focus=risk`}
              className="tm-mono"
              style={{
                color: 'var(--tm-ink-2)',
                textDecoration: 'underline',
                textUnderlineOffset: 2,
              }}
              title={
                language === 'zh'
                  ? '在策略工作室编辑风控 — 下一扫描周期生效'
                  : 'Edit risk limits on Strategy — next scan cycle picks them up'
              }
            >
              {language === 'zh' ? '编辑风控' : 'edit risk'}
            </Link>
          ) : null}
          <span style={{ marginLeft: 'auto' }}>
            <span className="tm-sc">next cycle </span>
            {countdown}
          </span>
        </div>
        <div className="tm-rule" />

        {status?.strategy_type === 'grid_trading' && traderId ? (
          <div style={{ padding: '10px 14px 0' }}>
            <GridRiskPanel
              traderId={traderId}
              language={language}
              refreshInterval={5000}
            />
          </div>
        ) : null}

        {/* metric row — "Total P/L" is equity-based (includes unrealized);
            "Realized P/L" is closed-trades only and matches PF/win-rate/sharpe,
            so the two never read as contradicting each other */}
        <div
          className="strand-dashboard-metrics"
          style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)' }}
        >
          {[
            {
              l: language === 'zh' ? '权益' : 'Equity',
              v: fmtUsd(account?.total_equity),
              c: 'var(--tm-ink)',
            },
            {
              l:
                language === 'zh'
                  ? '总盈亏 · 含未实现'
                  : 'Total P/L · incl. unrealized',
              v: `${fmtUsd(pnl, true)} (${fmtPct(pnlPct)})`,
              c: up ? 'var(--tm-up)' : 'var(--tm-dn)',
            },
            {
              l:
                language === 'zh'
                  ? '已实现盈亏 · 平仓'
                  : 'Realized P/L · closed trades',
              v: fullStats != null ? fmtUsd(fullStats.total_pnl, true) : '—',
              c:
                fullStats != null && fullStats.total_pnl >= 0
                  ? 'var(--tm-up)'
                  : 'var(--tm-dn)',
            },
            {
              l: language === 'zh' ? '盈亏比' : 'Profit factor',
              v: fullStats != null ? fullStats.profit_factor.toFixed(2) : '—',
              c: 'var(--tm-ink)',
            },
            // max_drawdown_pct is already a percent (18.5 = -18.5%)
            {
              l: language === 'zh' ? '最大回撤' : 'Max drawdown',
              v:
                fullStats != null
                  ? `-${fullStats.max_drawdown_pct.toFixed(1)}%`
                  : '—',
              c: 'var(--tm-dn)',
            },
          ].map((m, i) => (
            <div
              key={m.l}
              style={{
                padding: '12px 14px',
                borderRight: i < 4 ? cellBorder : 'none',
              }}
            >
              <div className="tm-sc">{m.l}</div>
              <div
                className="tm-mono"
                style={{
                  fontSize: 17,
                  fontWeight: 500,
                  color: m.c,
                  marginTop: 3,
                }}
              >
                {m.v}
              </div>
            </div>
          ))}
        </div>
        <div className="tm-rule" />

        {/* trades summary */}
        {fullStats != null && (
          <>
            <div
              className="tm-mono"
              style={{
                display: 'flex',
                gap: 18,
                padding: '6px 14px',
                fontSize: 11,
                color: 'var(--tm-ink-2)',
                flexWrap: 'wrap',
              }}
            >
              <span className="tm-sc">
                trades{' '}
                <b style={{ color: 'var(--tm-ink)' }}>
                  {fullStats.total_trades}
                </b>
              </span>
              <span className="tm-sc tm-up">
                win {fullStats.win_trades} ({fullStats.win_rate.toFixed(1)}%)
              </span>
              <span className="tm-sc tm-dn">loss {fullStats.loss_trades}</span>
              {/* fee-drag chain: gross realized − fees = net realized */}
              <span className="tm-sc">
                gross{' '}
                <b style={{ color: 'var(--tm-ink)' }}>
                  {fmtUsd(fullStats.total_pnl + fullStats.total_fee, true)}
                </b>
              </span>
              <span className="tm-sc">
                fees{' '}
                <b style={{ color: 'var(--tm-ink)' }}>
                  -{fmtUsd(fullStats.total_fee)}
                </b>
              </span>
              <span className="tm-sc">
                net{' '}
                <b
                  style={{
                    color:
                      fullStats.total_pnl >= 0
                        ? 'var(--tm-up)'
                        : 'var(--tm-dn)',
                  }}
                >
                  {fmtUsd(fullStats.total_pnl, true)}
                </b>
              </span>
              <span className="tm-sc">
                sharpe/trade{' '}
                <b style={{ color: 'var(--tm-ink)' }}>
                  {fullStats.sharpe_ratio.toFixed(2)}
                </b>
              </span>
              <span className="tm-sc">
                avg win/loss{' '}
                <b style={{ color: 'var(--tm-ink)' }}>
                  {fullStats.avg_win.toFixed(2)}/{fullStats.avg_loss.toFixed(2)}
                </b>
              </span>
            </div>
            <div className="tm-rule" />
          </>
        )}

        {/* ── row 1: cost/liq map · live L2 order book · signal matrix (instrument selector)
              all three columns are locked to one fixed height so the row is always
              balanced; the K-line flexes to fill any remaining space ── */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns:
              'minmax(0,1.1fr) minmax(0,0.95fr) minmax(0,1.05fr)',
          }}
        >
          <div
            style={{
              ...sc,
              borderRight: cellBorder,
              height: ROW1_H,
              overflow: 'hidden',
            }}
          >
            {/* cost/liq heatmap works for both synthetic (hip3_perp) and crypto
                (perp) markets — pass the likely marketType; the component falls
                back to the other one if the guess is wrong */}
            <LiquidationMap
              symbol={activeSym}
              demo={on}
              marketType={activeMarketType}
              height={ROW1_H - 130}
            />
          </div>
          <div
            style={{
              ...sc,
              borderRight: cellBorder,
              height: ROW1_H,
              overflow: 'hidden',
            }}
          >
            <OrderBook
              symbol={activeRequestSymbol}
              demo={on}
              markPrice={
                positions?.find((p) => baseLabel(p.symbol) === activeSym)
                  ?.entry_price
              }
            />
          </div>
          <div
            style={{
              ...sc,
              height: ROW1_H,
              display: 'flex',
              flexDirection: 'column',
              minHeight: 0,
            }}
          >
            <SignalMatrix
              items={signalRank?.items}
              max={18}
              active={activeSym}
              onSelect={setSelectedSym}
            />
            {/* the live K-line always sits under the selector and flexes to fill */}
            <div className="tm-rule" style={{ margin: '10px 0 8px' }} />
            <div style={{ flex: 1, minHeight: 0 }}>
              <KlineChart symbol={activeRequestSymbol} fill demo={on} />
            </div>
          </div>
        </div>
        <div className="tm-rule" />

        {/* ── win-rate matrix (holder cost × win-rate grid, claw402 paid data).
              Full-width panel under row 1; long/short grids side by side with
              click-to-drill address lists. ── */}
        <div style={sc}>
          <WinrateMatrix
            symbol={activeSym}
            demo={on}
            marketType={activeMarketType}
          />
        </div>
        <div className="tm-rule" />

        {/* orchestration topology — second row, full width (the agent workflow) */}
        <div style={sc}>
          <div
            style={{
              display: 'flex',
              alignItems: 'baseline',
              gap: 10,
              marginBottom: 4,
            }}
          >
            <span className="tm-px" style={{ fontSize: 12 }}>
              Orchestration topology
            </span>
            <span className="tm-sc">
              Orchestration topology · net inflow → signal → execute → hold
            </span>
          </div>
          <OrchestrationTopology
            layers={[
              {
                key: 'flow',
                title: 'FLOW',
                zh: 'flow',
                items: [
                  ...(flow?.data?.inflow ?? []).map((i) => ({
                    symbol: i.symbol,
                    dir: 'long' as const,
                  })),
                  ...(flow?.data?.outflow ?? []).map((i) => ({
                    symbol: i.symbol,
                    dir: 'short' as const,
                  })),
                ],
              },
              {
                key: 'signal',
                title: 'SIGNAL',
                zh: 'signal',
                items: (signalRank?.items ?? []).map((s) => ({
                  symbol: s.symbol,
                  dir:
                    (s.bias || '').toLowerCase() === 'bearish'
                      ? ('short' as const)
                      : ('long' as const),
                })),
              },
              {
                // every candidate the AI actually judged this cycle (its full decision set)
                key: 'decision',
                title: 'DECISION',
                zh: 'decision',
                items: candidateCoins.map((c) => ({
                  symbol: c,
                  dir: dirFor(c),
                })),
              },
              {
                // executed & live: every open position is an executed order, so
                // EXECUTE mirrors the live book (this cycle's fills plus anything
                // still open from prior cycles) and flows straight into HOLD
                key: 'exec',
                title: 'EXECUTE',
                zh: 'execute',
                items: (positions ?? []).map((p) => ({
                  symbol: p.symbol,
                  dir: (p.side || '').toLowerCase().includes('short')
                    ? ('short' as const)
                    : ('long' as const),
                })),
              },
              {
                key: 'hold',
                title: 'HOLD',
                zh: 'hold',
                items: (positions ?? []).map((p) => ({
                  symbol: p.symbol,
                  dir: (p.side || '').toLowerCase().includes('short')
                    ? ('short' as const)
                    : ('long' as const),
                })),
              },
            ]}
          />
        </div>
        <div className="tm-rule" />

        {/* equity curve — first visual proof the Autopilot is accumulating history */}
        {traderId && !on && (
          <div style={{ ...sc, borderBottom: cellBorder }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'baseline',
                gap: 8,
                marginBottom: 8,
              }}
            >
              <span className="tm-px" style={{ fontSize: 11 }}>
                Equity
              </span>
              <span className="tm-sc">Account equity curve · live</span>
            </div>
            <EquityChart traderId={traderId} embedded />
            <div style={{ marginTop: 12 }}>
              <ChartToOrderPanel
                symbol={
                  (positions?.[0]?.symbol || candidateCoins?.[0] || 'BTC').replace(
                    /USDT$/i,
                    ''
                  )
                }
                exchange="hyperliquid"
                traderId={traderId}
                language={language}
              />
            </div>
          </div>
        )}

        {/* ── row 3: execution log · risk radar · recent trades ── */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'minmax(0,1.1fr) minmax(0,1fr) minmax(0,1fr)',
          }}
        >
          <div style={{ ...sc, borderRight: cellBorder }}>
            <ExecutionLog decisions={decisions} height={432} />
          </div>
          <div style={{ ...sc, borderRight: cellBorder }}>
            <RiskRadar
              positions={positions}
              account={account}
              config={{
                btc_eth_leverage:
                  config?.btc_eth_leverage ?? status?.btc_eth_leverage,
                altcoin_leverage:
                  config?.altcoin_leverage ?? status?.altcoin_leverage,
                max_positions: config?.max_positions ?? status?.max_positions,
                max_margin_usage: config?.max_margin_usage,
                daily_loss_limit_pct: dailyLossLimit,
                max_drawdown_limit_pct: drawdownLimit,
              }}
              fullStats={fullStats}
              dayLossPct={dayLossPct}
              liveDrawdownPct={liveDrawdownPct}
              strategyEditHref={
                config?.strategy_id
                  ? `${ROUTES.strategy}?id=${encodeURIComponent(config.strategy_id)}&focus=risk`
                  : undefined
              }
            />
          </div>
          <div style={sc}>
            {/* live open positions (the book right now) */}
            <div
              style={{
                display: 'flex',
                alignItems: 'baseline',
                gap: 8,
                marginBottom: 6,
              }}
            >
              <span className="tm-px" style={{ fontSize: 11 }}>
                Positions
              </span>
              <span className="tm-sc">Current positions · live</span>
              <span className="tm-sc" style={{ marginLeft: 'auto' }}>
                {positions?.length ?? 0} open
              </span>
              {traderId && !on && positions && positions.length > 0 && (
                <button
                  type="button"
                  onClick={() => void closeAllPositions(positions)}
                  disabled={closing !== null}
                  className="tm-mono"
                  style={{
                    background: 'transparent',
                    border: '1px solid var(--tm-dn)',
                    color: 'var(--tm-dn)',
                    borderRadius: 3,
                    fontSize: 9,
                    padding: '1px 6px',
                    cursor: closing ? 'not-allowed' : 'pointer',
                    opacity: closing ? 0.5 : 1,
                  }}
                >
                  {closing === '__all__' ? 'closing…' : 'close all'}
                </button>
              )}
            </div>
            {positions && positions.length > 0 ? (
              <table
                className="tm-mono"
                style={{
                  width: '100%',
                  borderCollapse: 'collapse',
                  fontSize: 11,
                }}
              >
                <thead>
                  <tr className="tm-sc" style={{ fontSize: 9 }}>
                    <td style={{ padding: '0 0 3px' }}>symbol</td>
                    <td style={{ padding: '0 0 3px' }}>side·lev</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>
                      entry
                    </td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>
                      size
                    </td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>
                      PnL
                    </td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>
                      return%
                    </td>
                    {traderId && !on && <td style={{ padding: '0 0 3px' }} />}
                  </tr>
                </thead>
                <tbody>
                  {positions.map((p, i) => {
                    const long = /long|buy/i.test(p.side)
                    const win = (p.unrealized_pnl ?? 0) >= 0
                    const notional =
                      Math.abs(p.quantity ?? 0) *
                      (p.mark_price || p.entry_price || 0)
                    return (
                      <tr
                        key={`${p.symbol}-${i}`}
                        style={{ borderTop: '1px solid var(--tm-hair)' }}
                      >
                        <td style={{ padding: '5px 0', fontWeight: 500 }}>
                          {baseLabel(p.symbol)}
                        </td>
                        <td
                          style={{ padding: '5px 0' }}
                          className={long ? 'tm-up' : 'tm-dn'}
                        >
                          {long ? 'long' : 'short'}{' '}
                          <span style={{ color: 'var(--tm-muted)' }}>
                            {p.leverage}×
                          </span>
                        </td>
                        <td
                          style={{
                            padding: '5px 0',
                            textAlign: 'right',
                            color: 'var(--tm-ink-2)',
                          }}
                        >
                          {fmtPx(p.entry_price)}
                        </td>
                        <td
                          style={{
                            padding: '5px 0',
                            textAlign: 'right',
                            color: 'var(--tm-ink-2)',
                          }}
                        >
                          {fmtUsd(notional)}
                        </td>
                        <td
                          style={{ padding: '5px 0', textAlign: 'right' }}
                          className={win ? 'tm-up' : 'tm-dn'}
                        >
                          {fmtUsd(p.unrealized_pnl, true)}
                        </td>
                        <td
                          style={{ padding: '5px 0', textAlign: 'right' }}
                          className={win ? 'tm-up' : 'tm-dn'}
                        >
                          {(p.unrealized_pnl_pct ?? 0).toFixed(2)}%
                        </td>
                        {traderId && !on && (
                          <td
                            style={{
                              padding: '5px 0 5px 8px',
                              textAlign: 'right',
                              width: 1,
                            }}
                          >
                            <button
                              type="button"
                              onClick={() =>
                                void closePositionRow(
                                  p.symbol,
                                  long ? 'LONG' : 'SHORT'
                                )
                              }
                              disabled={closing !== null}
                              title={`Close ${p.symbol}`}
                              className="tm-mono"
                              style={{
                                background: 'transparent',
                                border: '1px solid var(--tm-dn)',
                                color: 'var(--tm-dn)',
                                borderRadius: 3,
                                fontSize: 9,
                                padding: '1px 5px',
                                cursor: closing ? 'not-allowed' : 'pointer',
                                opacity: closing ? 0.5 : 1,
                              }}
                            >
                              {closing === p.symbol ? '…' : 'close'}
                            </button>
                          </td>
                        )}
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            ) : (
              <div className="tm-sc" style={{ padding: '8px 0' }}>
                {positionsFailed ? (
                  <span style={{ color: 'var(--tm-dn)' }}>
                    Positions API failed — book may be stale.{' '}
                    {onRetryPositions ? (
                      <button
                        type="button"
                        className="tm-mono"
                        onClick={onRetryPositions}
                        style={{
                          border: '1px solid var(--tm-dn)',
                          color: 'var(--tm-dn)',
                          background: 'transparent',
                          padding: '1px 6px',
                          cursor: 'pointer',
                        }}
                      >
                        Retry
                      </button>
                    ) : null}
                  </span>
                ) : status?.trading_blocked ? (
                  language === 'zh'
                    ? '交易已阻断，当前无法开仓。'
                    : 'Trading blocked — no new positions until re-authorized.'
                ) : status?.paused ? (
                  language === 'zh'
                    ? '交易已暂停，当前不开新仓。'
                    : 'Trading paused — no new positions until resume.'
                ) : status?.is_running ? (
                  language === 'zh'
                    ? '暂无持仓 — Autopilot 运行中，等待下一决策周期。'
                    : 'Flat — Autopilot is running; waiting for the next cycle.'
                ) : (
                  language === 'zh' ? '暂无持仓。' : 'No open positions.'
                )}
              </div>
            )}

            <div className="tm-rule" style={{ margin: '12px 0 10px' }} />

            <div
              style={{
                display: 'flex',
                alignItems: 'baseline',
                gap: 8,
                marginBottom: 6,
              }}
            >
              <span className="tm-px" style={{ fontSize: 11 }}>
                {language === 'zh' ? '最近成交' : 'Recent trades'}
              </span>
              <span className="tm-sc">
                {language === 'zh'
                  ? '最近平仓 · 点击查看 AI 周期'
                  : 'Recent closes · click row for AI cycle'}
              </span>
              {traderId ? (
                <button
                  type="button"
                  className="tm-mono"
                  onClick={() => setShowFullHistory((v) => !v)}
                  style={{
                    marginLeft: 'auto',
                    fontSize: 9,
                    padding: '2px 6px',
                    border: '1px solid var(--tm-hair)',
                    color: 'var(--tm-ink-2)',
                    background: 'transparent',
                    cursor: 'pointer',
                  }}
                >
                  {showFullHistory
                    ? language === 'zh'
                      ? '收起全部'
                      : 'Hide all'
                    : language === 'zh'
                      ? '全部平仓'
                      : 'All closes'}
                </button>
              ) : null}
            </div>
            {recentTrades.length > 0 ? (
              <>
                <table
                  className="tm-mono"
                  style={{
                    width: '100%',
                    borderCollapse: 'collapse',
                    fontSize: 11,
                  }}
                >
                  <thead>
                    <tr className="tm-sc" style={{ fontSize: 9 }}>
                      <td style={{ padding: '0 0 3px' }}>
                        {language === 'zh' ? '标的' : 'symbol'}
                      </td>
                      <td style={{ padding: '0 0 3px' }}>
                        {language === 'zh' ? '方向' : 'side'}
                      </td>
                      <td style={{ padding: '0 0 3px', textAlign: 'right' }}>
                        {language === 'zh' ? '持仓' : 'hold'}
                      </td>
                      <td style={{ padding: '0 0 3px' }}>
                        {language === 'zh' ? '平仓' : 'closed'}
                      </td>
                      <td style={{ padding: '0 0 3px', textAlign: 'right' }}>
                        PnL
                      </td>
                    </tr>
                  </thead>
                  <tbody>
                    {recentTrades.map((p) => {
                      const win = p.realized_pnl >= 0
                      const linked = decisionByTrade.has(p.id)
                      const open = replayTradeId === p.id
                      return (
                        <tr
                          key={p.id}
                          onClick={() =>
                            setReplayTradeId((cur) =>
                              cur === p.id ? null : p.id
                            )
                          }
                          style={{
                            borderTop: '1px solid var(--tm-hair)',
                            cursor: linked ? 'pointer' : 'default',
                            background: open
                              ? 'color-mix(in srgb, var(--tm-ink) 6%, transparent)'
                              : 'transparent',
                          }}
                          title={
                            linked
                              ? 'Show matching AI decision cycle'
                              : 'No nearby decision cycle in loaded history'
                          }
                        >
                          <td style={{ padding: '5px 0', fontWeight: 500 }}>
                            {baseLabel(p.symbol)}
                            {linked ? (
                              <span
                                className="tm-sc"
                                style={{ marginLeft: 4, fontSize: 8 }}
                              >
                                AI
                              </span>
                            ) : null}
                          </td>
                          <td
                            style={{ padding: '5px 0' }}
                            className={
                              p.side === 'long' || p.side === 'LONG'
                                ? 'tm-up'
                                : 'tm-dn'
                            }
                          >
                            {p.side.toLowerCase()}
                          </td>
                          <td
                            style={{
                              padding: '5px 0',
                              textAlign: 'right',
                              color: 'var(--tm-ink-2)',
                            }}
                          >
                            {fmtHold(p.entry_time, p.exit_time)}
                          </td>
                          <td
                            style={{
                              padding: '5px 0 5px 6px',
                              color: 'var(--tm-muted)',
                            }}
                          >
                            {fmtTime(p.exit_time)}
                          </td>
                          <td
                            style={{ padding: '5px 0', textAlign: 'right' }}
                            className={win ? 'tm-up' : 'tm-dn'}
                          >
                            {fmtUsd(p.realized_pnl, true)}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
                {replayTradeId != null &&
                decisionByTrade.get(replayTradeId) ? (
                  <TradeDecisionReplay
                    record={decisionByTrade.get(replayTradeId)!}
                    symbol={
                      recentTrades.find((t) => t.id === replayTradeId)
                        ?.symbol
                    }
                  />
                ) : null}
              </>
            ) : (
              <div className="tm-sc" style={{ padding: '8px 0' }}>
                {language === 'zh' ? '暂无平仓记录。' : 'No closed trades yet.'}
              </div>
            )}
          </div>
        </div>

        {showFullHistory && traderId ? (
          <>
            <div className="tm-rule" />
            <div
              id="terminal-position-history"
              style={{ padding: '12px 14px 16px' }}
            >
              <div
                className="tm-mono"
                style={{
                  display: 'flex',
                  alignItems: 'baseline',
                  gap: 8,
                  marginBottom: 10,
                  fontSize: 11,
                }}
              >
                <span className="tm-px">
                  {language === 'zh' ? '完整平仓历史' : 'Full close history'}
                </span>
                <span className="tm-sc">
                  {language === 'zh'
                    ? '筛选 · 分页 · 统计'
                    : 'filter · page · stats'}
                </span>
              </div>
              <PositionHistory traderId={traderId} />
            </div>
          </>
        ) : null}

        <div className="tm-rule" />

        {/* market net inflow (Vergex) · by-symbol history · edge profile — footer */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns:
              'minmax(0,1.2fr) minmax(0,0.9fr) minmax(0,0.9fr)',
          }}
        >
          <div style={{ ...sc, borderRight: cellBorder }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'baseline',
                gap: 10,
                marginBottom: 8,
              }}
            >
              <span className="tm-px" style={{ fontSize: 12 }}>
                Market net inflow
              </span>
              <span className="tm-sc">
                Market net inflow · {flow?.data?.window || '1h'} · Vergex
              </span>
              <span className="tm-sc" style={{ marginLeft: 'auto' }}>
                {flowItems.length} markets
              </span>
            </div>
            <FlowMarkets items={flowItems} window={flow?.data?.window} />
          </div>
          <div style={{ ...sc, borderRight: cellBorder }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'baseline',
                gap: 8,
                marginBottom: 8,
              }}
            >
              <span className="tm-px" style={{ fontSize: 11 }}>
                By symbol
              </span>
              <span className="tm-sc">By-symbol history · trades/win/pnl</span>
            </div>
            {symbolStats.length > 0 ? (
              symbolStats.map((s) => (
                <div key={s.symbol} style={{ marginBottom: 7 }}>
                  <div
                    className="tm-mono"
                    style={{ display: 'flex', fontSize: 11, marginBottom: 2 }}
                  >
                    <span style={{ fontWeight: 500 }}>
                      {baseLabel(s.symbol)}
                    </span>
                    <span className="tm-sc" style={{ marginLeft: 8 }}>
                      {s.total_trades} trades · {s.win_rate.toFixed(0)}% win
                    </span>
                    <span
                      className={s.total_pnl >= 0 ? 'tm-up' : 'tm-dn'}
                      style={{ marginLeft: 'auto' }}
                    >
                      {fmtUsd(s.total_pnl, true)}
                    </span>
                  </div>
                  <div style={{ height: 4, background: 'var(--tm-hair)' }}>
                    <div
                      style={{
                        height: 4,
                        width: `${(s.total_trades / maxSymTrades) * 100}%`,
                        background:
                          s.total_pnl >= 0 ? 'var(--tm-up)' : 'var(--tm-dn)',
                      }}
                    />
                  </div>
                </div>
              ))
            ) : (
              <div className="tm-sc">No symbol history.</div>
            )}
          </div>
          <div style={sc}>
            <div
              style={{
                display: 'flex',
                alignItems: 'baseline',
                gap: 8,
                marginBottom: 8,
              }}
            >
              <span className="tm-px" style={{ fontSize: 11 }}>
                Edge profile
              </span>
              <span className="tm-sc">
                Net by hold time &amp; side · after fees
              </span>
            </div>
            <EdgeProfile positions={history?.positions} />
          </div>
        </div>
      </div>
    </div>
  )
}

function TradeDecisionReplay({
  record,
  symbol,
}: {
  record: DecisionRecord
  symbol?: string
}) {
  const base = symbol ? baseLabel(symbol) : ''
  const actions = (record.decisions ?? []).filter((a) =>
    base ? baseLabel(a.symbol) === base : true
  )
  const shown = actions.length > 0 ? actions : (record.decisions ?? []).slice(0, 3)
  return (
    <div
      className="tm-mono"
      style={{
        marginTop: 8,
        padding: '8px 10px',
        border: '1px solid var(--tm-hair)',
        background: 'color-mix(in srgb, var(--tm-ink) 4%, transparent)',
        fontSize: 10,
        color: 'var(--tm-ink-2)',
        maxHeight: 160,
        overflowY: 'auto',
        whiteSpace: 'pre-wrap',
        overflowWrap: 'anywhere',
      }}
    >
      <div style={{ marginBottom: 6, color: 'var(--tm-ink)' }}>
        CYCLE {record.cycle_number}
        <span style={{ color: 'var(--tm-muted)', marginLeft: 8 }}>
          {record.timestamp}
        </span>
      </div>
      {record.error_message ? (
        <div style={{ color: 'var(--tm-dn)', marginBottom: 6 }}>
          {record.error_message}
        </div>
      ) : null}
      {shown.map((a, i) => (
        <div key={`replay-${i}`} style={{ marginBottom: 6 }}>
          <strong>
            {a.action} {baseLabel(a.symbol)}
          </strong>
          {a.reasoning ? <div>{a.reasoning}</div> : null}
          {a.error ? (
            <div style={{ color: 'var(--tm-dn)' }}>{a.error}</div>
          ) : null}
        </div>
      ))}
      {record.cot_trace ? (
        <div>
          <strong>Chain of thought</strong>
          {'\n'}
          {record.cot_trace.length > 600
            ? `${record.cot_trace.slice(0, 600)}…`
            : record.cot_trace}
        </div>
      ) : null}
      {!shown.length && !record.cot_trace && !record.error_message ? (
        <div className="tm-sc">Matched cycle has no action detail.</div>
      ) : null}
    </div>
  )
}

export default TerminalDashboard
