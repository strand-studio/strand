import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import useSWR from 'swr'
import {
  ArrowLeft,
  ArrowLeftRight,
  Copy,
  ExternalLink,
  MessageSquare,
  RefreshCw,
  TrendingUp,
} from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import { t } from '../i18n/translations'
import { api } from '../lib/api'
import type { Kline, MarketSymbol } from '../lib/api/data'
import {
  buildMarketSymbolPath,
  ROUTES,
} from '../router/paths'
import {
  MarketDetailChart,
  type ChartMode,
} from '../components/markets/MarketDetailChart'
import './MarketSymbolPage.css'

const INTERVALS = [
  { id: '5m', label: '5m', limit: 288 },
  { id: '15m', label: '15m', limit: 192 },
  { id: '1h', label: '1H', limit: 168 },
  { id: '4h', label: '4H', limit: 180 },
  { id: '1d', label: '1D', limit: 120 },
] as const

function text(lang: string, zh: string, en: string, id?: string) {
  if (lang === 'zh') return zh
  if (lang === 'id' && id) return id
  return en
}

function formatPrice(value?: number) {
  if (value == null || !Number.isFinite(value)) return '—'
  if (Math.abs(value) >= 1000)
    return value.toLocaleString(undefined, { maximumFractionDigits: 2 })
  if (Math.abs(value) >= 1)
    return value.toLocaleString(undefined, { maximumFractionDigits: 4 })
  return value.toLocaleString(undefined, {
    maximumFractionDigits: 6,
    minimumFractionDigits: 2,
  })
}

function formatVolume(value?: number) {
  if (value == null || !Number.isFinite(value) || value <= 0) return '—'
  return new Intl.NumberFormat(undefined, {
    notation: 'compact',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatChange(value?: number) {
  if (value == null || !Number.isFinite(value)) return '—'
  const sign = value > 0 ? '+' : ''
  return `${sign}${value.toFixed(2)}%`
}

function displayOf(row?: MarketSymbol | null, fallback = '') {
  return row?.display || row?.symbol || fallback
}

function matchSymbol(list: MarketSymbol[], raw: string) {
  const q = raw.trim().toUpperCase()
  return (
    list.find((s) => s.symbol.toUpperCase() === q) ||
    list.find((s) => (s.display || '').toUpperCase() === q) ||
    list.find((s) => s.symbol.toUpperCase().includes(q)) ||
    null
  )
}

function periodChange(bars: Kline[], lookback: number) {
  if (!bars.length) return undefined
  const end = bars[bars.length - 1]
  const start = bars[Math.max(0, bars.length - 1 - lookback)]
  const a = Number(start.open ?? start.close)
  const b = Number(end.close)
  if (!Number.isFinite(a) || !Number.isFinite(b) || a === 0) return undefined
  return ((b - a) / a) * 100
}

function rangeHiLo(bars: Kline[]) {
  let hi = -Infinity
  let lo = Infinity
  for (const b of bars) {
    const h = Number(b.high)
    const l = Number(b.low)
    if (Number.isFinite(h)) hi = Math.max(hi, h)
    if (Number.isFinite(l)) lo = Math.min(lo, l)
  }
  return {
    high: Number.isFinite(hi) ? hi : undefined,
    low: Number.isFinite(lo) ? lo : undefined,
  }
}

function aboutBlurb(
  lang: string,
  exchange: string,
  display: string,
  category: string
) {
  const venue = exchange.charAt(0).toUpperCase() + exchange.slice(1)
  if (lang === 'zh') {
    return `${display} 是 ${venue} 上的 ${category || 'crypto'} 永续／合约行情。Strand Markets 通过公开接口汇总最新价、24h 涨跌、成交量与 K 线，便于研究后再跳转 Trade 规划或 Chat 简报——不托管资金。`
  }
  if (lang === 'id') {
    return `${display} adalah pasar ${category || 'crypto'} di ${venue}. Strand Markets menampilkan harga, perubahan 24j, volume, dan K-line dari API publik agar Anda bisa riset sebelum Trade atau Chat — tanpa kustodi dana.`
  }
  return `${display} is a ${category || 'crypto'} perpetual market on ${venue}. Strand Markets aggregates live mark price, 24h change, volume and candles from public APIs so you can research before jumping to Trade or Chat — no custody.`
}

export function MarketSymbolPage() {
  const { language } = useLanguage()
  const { token } = useAuth()
  const isLoggedIn = Boolean(token)
  const navigate = useNavigate()
  const params = useParams<{ exchange: string; symbol: string }>()
  const exchange = decodeURIComponent(params.exchange || 'hyperliquid').toLowerCase()
  const symbolParam = decodeURIComponent(params.symbol || '')

  const [interval, setInterval] = useState<(typeof INTERVALS)[number]['id']>('1h')
  const [chartMode, setChartMode] = useState<ChartMode>('candle')
  const [quoteAmount, setQuoteAmount] = useState('1000')
  const [baseAmount, setBaseAmount] = useState('')
  const [copied, setCopied] = useState(false)

  const intervalMeta =
    INTERVALS.find((i) => i.id === interval) || INTERVALS[2]

  const { data: board, isLoading: boardLoading, mutate: mutateBoard } = useSWR(
    ['market-symbol-board', exchange],
    () => api.getSymbols(exchange, { limit: 500, silent: true }),
    { refreshInterval: 30_000, revalidateOnFocus: true }
  )

  const row = useMemo(
    () => matchSymbol(board?.symbols || [], symbolParam),
    [board?.symbols, symbolParam]
  )

  const symbol = row?.symbol || symbolParam
  const display = displayOf(row, symbolParam)

  const { data: klines, isLoading: klineLoading, mutate: mutateKlines, error: klineError } =
    useSWR(
      symbol
        ? ['market-symbol-klines', exchange, symbol, interval, intervalMeta.limit]
        : null,
      () =>
        api.getKlines(symbol, interval, exchange, intervalMeta.limit, true),
      { refreshInterval: 30_000, revalidateOnFocus: true }
    )

  const { data: direction } = useSWR(
    isLoggedIn && display
      ? ['market-symbol-direction', display]
      : null,
    async () => {
      try {
        return await api.getVergexDirectionChangeCurrent(display)
      } catch {
        return null
      }
    },
    { revalidateOnFocus: false }
  )

  const bars = klines || []
  const range = useMemo(() => rangeHiLo(bars), [bars])
  const perf = useMemo(
    () => ({
      h1: periodChange(bars, interval === '5m' ? 12 : interval === '15m' ? 4 : 1),
      h24: periodChange(
        bars,
        interval === '5m' ? 288 : interval === '15m' ? 96 : interval === '1h' ? 24 : interval === '4h' ? 6 : 1
      ),
      d7: periodChange(
        bars,
        interval === '1h' ? 168 : interval === '4h' ? 42 : interval === '1d' ? 7 : 0
      ),
      d30: periodChange(bars, interval === '1d' ? 30 : interval === '4h' ? 180 : 0),
    }),
    [bars, interval]
  )

  const related = useMemo(() => {
    const list = [...(board?.symbols || [])]
      .filter((s) => s.symbol !== symbol)
      .sort((a, b) => (b.volume_24h || 0) - (a.volume_24h || 0))
      .slice(0, 8)
    return list
  }, [board?.symbols, symbol])

  const price = row?.mark_price
  const change = row?.change_24h_pct
  const changeClass =
    change == null ? '' : change > 0 ? 'up' : change < 0 ? 'down' : ''

  const tradeHref = `${ROUTES.planning}?symbol=${encodeURIComponent(symbol)}`
  const chatHref = `${ROUTES.chat}?q=${encodeURIComponent(display)}`

  const syncFromQuote = (q: string) => {
    setQuoteAmount(q)
    const n = Number(q)
    if (!Number.isFinite(n) || !price || price <= 0) {
      setBaseAmount('')
      return
    }
    setBaseAmount((n / price).toFixed(6))
  }

  const syncFromBase = (b: string) => {
    setBaseAmount(b)
    const n = Number(b)
    if (!Number.isFinite(n) || !price || price <= 0) {
      setQuoteAmount('')
      return
    }
    setQuoteAmount((n * price).toFixed(2))
  }

  const copyLink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href)
      setCopied(true)
      setTimeout(() => setCopied(false), 1600)
    } catch {
      /* ignore */
    }
  }

  const refresh = () => {
    void mutateBoard()
    void mutateKlines()
  }

  useEffect(() => {
    if (!price || price <= 0) return
    const n = Number(quoteAmount)
    if (!Number.isFinite(n)) return
    setBaseAmount((n / price).toFixed(6))
    // Only reseed when mark price first becomes available / changes materially.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [price])

  return (
    <div className="msd-page">
      <div className="msd-shell">
        <nav className="msd-breadcrumb" aria-label="Breadcrumb">
          <Link to={ROUTES.markets}>{t('marketsNav', language)}</Link>
          <span>/</span>
          <Link to={`${ROUTES.markets}?exchange=${encodeURIComponent(exchange)}`}>
            {exchange}
          </Link>
          <span>/</span>
          <strong>{display}</strong>
        </nav>

        <header className="msd-hero">
          <div className="msd-hero-main">
            <button
              type="button"
              className="msd-back"
              onClick={() => navigate(ROUTES.markets)}
            >
              <ArrowLeft size={16} />
              {text(language, '返回市场', 'Back to Markets', 'Kembali ke Pasar')}
            </button>

            <div className="msd-title-row">
              <div className="msd-avatar" aria-hidden>
                {(display || '?').slice(0, 1)}
              </div>
              <div>
                <div className="msd-title-line">
                  <h1>{display}</h1>
                  <span className="msd-chip">{exchange}</span>
                  <span className="msd-chip">{row?.category || 'crypto'}</span>
                  {row?.maxLeverage ? (
                    <span className="msd-chip">{row.maxLeverage}x</span>
                  ) : null}
                </div>
                <p className="msd-subsym">{symbol}</p>
              </div>
            </div>

            <div className="msd-price-block">
              <div className="msd-price">{formatPrice(price)}</div>
              <div className={`msd-change ${changeClass}`}>
                {formatChange(change)}
                <small>
                  {text(language, '24 小时', '24h', '24j')}
                </small>
              </div>
            </div>
          </div>

          <div className="msd-hero-actions">
            {isLoggedIn ? (
              <>
                <Link className="msd-btn primary" to={tradeHref}>
                  <TrendingUp size={16} />
                  {t('tradeNav', language)}
                </Link>
                <Link className="msd-btn" to={chatHref}>
                  <MessageSquare size={16} />
                  {t('chatNav', language)}
                </Link>
              </>
            ) : (
              <Link
                className="msd-btn primary"
                to={ROUTES.login}
                onClick={() => {
                  try {
                    sessionStorage.setItem('returnUrl', tradeHref)
                  } catch {
                    /* ignore */
                  }
                }}
              >
                {text(language, '登录后交易', 'Sign in to trade', 'Masuk untuk trade')}
              </Link>
            )}
            <button type="button" className="msd-btn" onClick={copyLink}>
              <Copy size={16} />
              {copied
                ? text(language, '已复制', 'Copied', 'Disalin')
                : text(language, '复制链接', 'Copy link', 'Salin tautan')}
            </button>
            <button
              type="button"
              className="msd-btn"
              onClick={refresh}
              disabled={boardLoading || klineLoading}
            >
              <RefreshCw size={16} />
              {text(language, '刷新', 'Refresh', 'Muat ulang')}
            </button>
          </div>
        </header>

        <div className="msd-stats-strip" aria-label="Key stats">
          <div>
            <span>{text(language, '24h 成交量', '24h volume', 'Volume 24j')}</span>
            <strong>{formatVolume(row?.volume_24h)}</strong>
          </div>
          <div>
            <span>{text(language, '周期最高', 'Period high', 'Tertinggi')}</span>
            <strong>{formatPrice(range.high)}</strong>
          </div>
          <div>
            <span>{text(language, '周期最低', 'Period low', 'Terendah')}</span>
            <strong>{formatPrice(range.low)}</strong>
          </div>
          <div>
            <span>{text(language, '昨收 / 参考', 'Prev / ref', 'Sebelumnya')}</span>
            <strong>{formatPrice(row?.prev_day_price)}</strong>
          </div>
          <div>
            <span>{text(language, '数据源', 'Source', 'Sumber')}</span>
            <strong>{board?.source || '—'}</strong>
          </div>
        </div>

        <div className="msd-layout">
          <section className="msd-main">
            <div className="msd-toolbar">
              <div className="msd-intervals" role="tablist" aria-label="Interval">
                {INTERVALS.map((iv) => (
                  <button
                    key={iv.id}
                    type="button"
                    role="tab"
                    aria-selected={interval === iv.id}
                    onClick={() => setInterval(iv.id)}
                  >
                    {iv.label}
                  </button>
                ))}
              </div>
              <div className="msd-modes" role="group" aria-label="Chart mode">
                <button
                  type="button"
                  aria-pressed={chartMode === 'candle'}
                  onClick={() => setChartMode('candle')}
                >
                  {text(language, 'K 线', 'Candles', 'Candle')}
                </button>
                <button
                  type="button"
                  aria-pressed={chartMode === 'area'}
                  onClick={() => setChartMode('area')}
                >
                  {text(language, '走势', 'Area', 'Area')}
                </button>
              </div>
            </div>

            <div className="msd-chart-panel">
              {(boardLoading || klineLoading) && bars.length === 0 ? (
                <p className="msd-status">
                  {text(language, '加载行情…', 'Loading market…', 'Memuat pasar…')}
                </p>
              ) : klineError || bars.length < 2 ? (
                <p className="msd-status">
                  {text(
                    language,
                    '暂无 K 线（该所可能不可达或标的不支持）。价格统计仍可参考上方行情。',
                    'No candles yet (venue unreachable or unsupported). Stats above may still update.',
                    'Belum ada candle. Statistik di atas mungkin tetap tersedia.'
                  )}
                </p>
              ) : (
                <MarketDetailChart bars={bars} mode={chartMode} />
              )}
            </div>

            <section className="msd-perf">
              <h2>{text(language, '表现', 'Performance', 'Performa')}</h2>
              <div className="msd-perf-grid">
                {(
                  [
                    ['1h', perf.h1],
                    ['24h', perf.h24],
                    ['7d', perf.d7],
                    ['30d', perf.d30],
                  ] as const
                ).map(([label, value]) => (
                  <div key={label}>
                    <span>{label}</span>
                    <strong
                      className={
                        value == null ? '' : value > 0 ? 'up' : value < 0 ? 'down' : ''
                      }
                    >
                      {formatChange(value)}
                    </strong>
                  </div>
                ))}
              </div>
              <p className="msd-note">
                {text(
                  language,
                  '表现由当前周期 K 线推算，短周期下 7d/30d 可能显示为 —。',
                  'Performance is derived from the loaded candles; 7d/30d may be — on short intervals.',
                  'Performa dihitung dari candle yang dimuat; 7d/30d bisa — pada interval pendek.'
                )}
              </p>
            </section>

            <section className="msd-about">
              <h2>{text(language, '关于', 'About', 'Tentang')} {display}</h2>
              <p>{aboutBlurb(language, exchange, display, row?.category || 'crypto')}</p>
              <dl className="msd-meta">
                <div>
                  <dt>{text(language, '交易所', 'Exchange', 'Bursa')}</dt>
                  <dd>{exchange}</dd>
                </div>
                <div>
                  <dt>{text(language, '合约符号', 'Contract', 'Kontrak')}</dt>
                  <dd>{symbol}</dd>
                </div>
                <div>
                  <dt>{text(language, '显示名', 'Display', 'Tampilan')}</dt>
                  <dd>{display}</dd>
                </div>
                <div>
                  <dt>{text(language, '类别', 'Category', 'Kategori')}</dt>
                  <dd>{row?.category || 'crypto'}</dd>
                </div>
                {row?.sz_decimals != null && (
                  <div>
                    <dt>sz decimals</dt>
                    <dd>{row.sz_decimals}</dd>
                  </div>
                )}
              </dl>
            </section>
          </section>

          <aside className="msd-side">
            <section className="msd-card">
              <h2>
                <ArrowLeftRight size={16} />
                {text(language, '换算', 'Converter', 'Konverter')}
              </h2>
              <label className="msd-field">
                <span>USDT</span>
                <input
                  value={quoteAmount}
                  onChange={(e) => syncFromQuote(e.target.value)}
                  inputMode="decimal"
                />
              </label>
              <label className="msd-field">
                <span>{display}</span>
                <input
                  value={baseAmount}
                  onChange={(e) => syncFromBase(e.target.value)}
                  inputMode="decimal"
                />
              </label>
              <p className="msd-note">
                {text(
                  language,
                  '按最新标记价估算，非成交保证。',
                  'Estimated at latest mark — not an execution quote.',
                  'Estimasi mark terbaru — bukan kuotasi eksekusi.'
                )}
              </p>
            </section>

            {isLoggedIn && direction && (
              <section className="msd-card">
                <h2>{text(language, '方向信号', 'Direction', 'Arah')}</h2>
                <p className="msd-signal">
                  <strong
                    className={
                      direction.direction === 'bullish' ||
                      direction.new_bias === 'bullish'
                        ? 'up'
                        : direction.direction === 'bearish' ||
                            direction.new_bias === 'bearish'
                          ? 'down'
                          : ''
                    }
                  >
                    {direction.direction || direction.new_bias || '—'}
                  </strong>
                  {direction.mark_price != null && (
                    <em>{formatPrice(direction.mark_price)}</em>
                  )}
                </p>
                <p className="msd-note">
                  {text(
                    language,
                    '来自 Claw402 / Vergex（需配置）。',
                    'From Claw402 / Vergex when configured.',
                    'Dari Claw402 / Vergex jika dikonfigurasi.'
                  )}
                </p>
              </section>
            )}

            <section className="msd-card">
              <h2>{text(language, '同所热门', 'Related on venue', 'Terkait di venue')}</h2>
              <ul className="msd-related">
                {related.map((item) => (
                  <li key={item.symbol}>
                    <Link
                      to={buildMarketSymbolPath(exchange, item.symbol)}
                    >
                      <strong>{displayOf(item)}</strong>
                      <span className={
                        (item.change_24h_pct || 0) > 0
                          ? 'up'
                          : (item.change_24h_pct || 0) < 0
                            ? 'down'
                            : ''
                      }>
                        {formatChange(item.change_24h_pct)}
                      </span>
                      <em>{formatPrice(item.mark_price)}</em>
                    </Link>
                  </li>
                ))}
                {related.length === 0 && (
                  <li className="msd-note">—</li>
                )}
              </ul>
            </section>

            <section className="msd-card">
              <h2>{text(language, '快捷入口', 'Shortcuts', 'Pintasan')}</h2>
              <div className="msd-shortcuts">
                <Link to={ROUTES.markets}>
                  {t('marketsNav', language)}
                </Link>
                <Link to={isLoggedIn ? tradeHref : ROUTES.login}>
                  {t('tradeNav', language)}
                </Link>
                <a
                  href={`https://www.coingecko.com/en/search?query=${encodeURIComponent(display)}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  CoinGecko <ExternalLink size={12} />
                </a>
              </div>
            </section>
          </aside>
        </div>
      </div>
    </div>
  )
}
