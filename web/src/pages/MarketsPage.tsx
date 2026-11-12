import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router'
import useSWR from 'swr'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import { t } from '../i18n/translations'
import { api } from '../lib/api'
import type { MarketSymbol } from '../lib/api/data'
import { buildMarketSymbolPath, ROUTES } from '../router/paths'
import './MarketsPage.css'

type SortKey = 'volume' | 'change' | 'symbol' | 'price'
type CategoryFilter = 'all' | 'crypto' | 'stock' | 'forex' | 'commodity' | 'index'

const EXCHANGES = [
  { id: 'hyperliquid', label: 'Hyperliquid' },
  { id: 'binance', label: 'Binance' },
  { id: 'bybit', label: 'Bybit' },
  { id: 'okx', label: 'OKX' },
  { id: 'gate', label: 'Gate' },
  { id: 'bitget', label: 'Bitget' },
] as const

const CATEGORIES: CategoryFilter[] = [
  'all',
  'crypto',
  'stock',
  'forex',
  'commodity',
  'index',
]

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
  return value.toLocaleString(undefined, { maximumFractionDigits: 6 })
}

function formatVolume(value?: number) {
  if (value == null || !Number.isFinite(value) || value <= 0) return '—'
  return new Intl.NumberFormat(undefined, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value)
}

function formatChange(value?: number) {
  if (value == null || !Number.isFinite(value)) return '—'
  const sign = value > 0 ? '+' : ''
  return `${sign}${value.toFixed(2)}%`
}

function displaySymbol(row: MarketSymbol) {
  return row.display || row.symbol
}

export function MarketsPage() {
  const { language } = useLanguage()
  const { token } = useAuth()
  const isLoggedIn = Boolean(token)
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const initialExchange = (() => {
    const q = (searchParams.get('exchange') || '').toLowerCase()
    return EXCHANGES.some((e) => e.id === q) ? q : 'hyperliquid'
  })()

  const [exchange, setExchange] = useState<string>(initialExchange)
  const [category, setCategory] = useState<CategoryFilter>('all')
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('volume')
  const [sortDir, setSortDir] = useState<'desc' | 'asc'>('desc')

  useEffect(() => {
    const next = new URLSearchParams(searchParams)
    if (exchange === 'hyperliquid') next.delete('exchange')
    else next.set('exchange', exchange)
    setSearchParams(next, { replace: true })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [exchange])

  const { data, error, isLoading, isValidating, mutate } = useSWR(
    ['markets-symbols', exchange],
    () => api.getSymbols(exchange, { limit: 500, silent: true }),
    { refreshInterval: 30_000, revalidateOnFocus: true }
  )

  const { data: direction } = useSWR(
    isLoggedIn ? ['markets-direction', 12] : null,
    () => api.getDirectionChangeLeaderboard(12, true),
    { refreshInterval: 60_000 }
  )

  const { data: flow } = useSWR(
    isLoggedIn ? ['markets-flow', '1h'] : null,
    () => api.getFlowMarkets(undefined, 'mainnet', '1h', 8, true),
    { refreshInterval: 60_000 }
  )

  const rows = useMemo(() => {
    const list = [...(data?.symbols || [])]
    const q = search.trim().toLowerCase()
    const filtered = list.filter((row) => {
      if (category !== 'all' && (row.category || 'crypto') !== category) {
        return false
      }
      if (!q) return true
      const hay = `${row.symbol} ${row.display || ''} ${row.name || ''}`.toLowerCase()
      return hay.includes(q)
    })

    filtered.sort((a, b) => {
      const dir = sortDir === 'asc' ? 1 : -1
      switch (sortKey) {
        case 'price':
          return ((a.mark_price || 0) - (b.mark_price || 0)) * dir
        case 'change':
          return ((a.change_24h_pct || 0) - (b.change_24h_pct || 0)) * dir
        case 'symbol':
          return displaySymbol(a).localeCompare(displaySymbol(b)) * dir
        case 'volume':
        default:
          return ((a.volume_24h || 0) - (b.volume_24h || 0)) * dir
      }
    })
    return filtered
  }, [data?.symbols, category, search, sortKey, sortDir])

  const stats = useMemo(() => {
    const list = data?.symbols || []
    const withPrice = list.filter((r) => Number.isFinite(r.mark_price))
    const gainers = list.filter((r) => (r.change_24h_pct || 0) > 0).length
    const losers = list.filter((r) => (r.change_24h_pct || 0) < 0).length
    const topVol = [...list].sort(
      (a, b) => (b.volume_24h || 0) - (a.volume_24h || 0)
    )[0]
    return {
      count: list.length,
      priced: withPrice.length,
      gainers,
      losers,
      topVol,
    }
  }, [data?.symbols])

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === 'desc' ? 'asc' : 'desc'))
      return
    }
    setSortKey(key)
    setSortDir(key === 'symbol' ? 'asc' : 'desc')
  }

  const sortMark = (key: SortKey) =>
    sortKey === key ? (sortDir === 'desc' ? ' ↓' : ' ↑') : ''

  const frontierHref = (symbol: string) =>
    `${ROUTES.planning}?symbol=${encodeURIComponent(symbol)}`
  const chatHref = (symbol: string) =>
    `${ROUTES.chat}?q=${encodeURIComponent(symbol)}`

  return (
    <div className="markets-page">
      <div className="markets-shell">
        <header className="markets-header">
          <div>
            <span className="markets-eyebrow">
              {t('pageMarketsEyebrow', language)}
            </span>
            <h1>{t('pageMarketsTitle', language)}</h1>
            <p>{t('pageMarketsDesc', language)}</p>
          </div>
          <div className="markets-header-actions">
            {isLoggedIn ? (
              <Link
                to={ROUTES.strategy}
                className="markets-refresh"
                style={{ textDecoration: 'none' }}
              >
                {text(
                  language,
                  '策略工作室 · 方向板',
                  'Strategy · direction board',
                  'Strategi · papan arah'
                )}
              </Link>
            ) : null}
            <button
              type="button"
              className="markets-refresh"
              disabled={isValidating}
              onClick={() => mutate()}
            >
              {isValidating
                ? text(language, '刷新中…', 'Refreshing…', 'Memuat…')
                : text(language, '刷新', 'Refresh', 'Muat ulang')}
            </button>
            <span className="markets-meta">
              {text(language, '每 30 秒自动更新', 'Auto-refresh 30s', 'Auto 30 dtk')}
            </span>
          </div>
        </header>

        <div className="markets-summary">
          <div className="markets-stat">
            <span>{text(language, '标的数', 'Markets', 'Pasar')}</span>
            <strong>{stats.count || '—'}</strong>
          </div>
          <div className="markets-stat">
            <span>{text(language, '上涨', 'Gainers', 'Naik')}</span>
            <strong className="up">{stats.gainers || '—'}</strong>
          </div>
          <div className="markets-stat">
            <span>{text(language, '下跌', 'Losers', 'Turun')}</span>
            <strong className="down">{stats.losers || '—'}</strong>
          </div>
          <div className="markets-stat">
            <span>{text(language, '成交量王', 'Top volume', 'Volume tertinggi')}</span>
            <strong>
              {stats.topVol ? displaySymbol(stats.topVol) : '—'}
            </strong>
          </div>
        </div>

        <div className="markets-toolbar">
          <div className="markets-exchange" role="tablist" aria-label="Exchange">
            {EXCHANGES.map((ex) => (
              <button
                key={ex.id}
                type="button"
                role="tab"
                aria-selected={exchange === ex.id}
                onClick={() => {
                  setExchange(ex.id)
                  // CEX boards are crypto-only; keep filters from emptying the table.
                  if (ex.id !== 'hyperliquid' && category !== 'all' && category !== 'crypto') {
                    setCategory('all')
                  }
                }}
              >
                {ex.label}
              </button>
            ))}
          </div>

          <div className="markets-filters">
            <div className="markets-categories" role="group" aria-label="Category">
              {(exchange === 'hyperliquid'
                ? CATEGORIES
                : (['all', 'crypto'] as CategoryFilter[])
              ).map((cat) => (
                <button
                  key={cat}
                  type="button"
                  aria-pressed={category === cat}
                  onClick={() => setCategory(cat)}
                >
                  {cat === 'all'
                    ? text(language, '全部', 'All', 'Semua')
                    : cat}
                </button>
              ))}
            </div>
            <input
              className="markets-search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={text(
                language,
                '搜索标的，如 BTC / TSLA',
                'Search symbol, e.g. BTC / TSLA',
                'Cari simbol, mis. BTC / TSLA'
              )}
              aria-label={text(language, '搜索市场', 'Search markets', 'Cari pasar')}
            />
          </div>
        </div>

        <div className="markets-layout">
          <section className="markets-table-panel" aria-label="Market table">
            {isLoading && (
              <p className="markets-status" role="status">
                {text(language, '加载行情…', 'Loading markets…', 'Memuat pasar…')}
              </p>
            )}
            {error && (
              <p className="markets-error" role="alert">
                {text(
                  language,
                  '无法加载市场数据，请确认后端已启动后重试。',
                  'Could not load markets. Check that the API is running, then retry.',
                  'Gagal memuat pasar. Pastikan API berjalan, lalu coba lagi.'
                )}
              </p>
            )}
            {!isLoading && !error && rows.length === 0 && (
              <p className="markets-status">
                {text(
                  language,
                  '没有匹配的标的。',
                  'No markets match your filters.',
                  'Tidak ada pasar yang cocok.'
                )}
              </p>
            )}

            {rows.length > 0 && (
              <div className="markets-table-wrap">
                <table className="markets-table">
                  <thead>
                    <tr>
                      <th>
                        <button type="button" onClick={() => toggleSort('symbol')}>
                          {text(language, '标的', 'Symbol', 'Simbol')}
                          {sortMark('symbol')}
                        </button>
                      </th>
                      <th>
                        <button type="button" onClick={() => toggleSort('price')}>
                          {text(language, '最新价', 'Price', 'Harga')}
                          {sortMark('price')}
                        </button>
                      </th>
                      <th>
                        <button type="button" onClick={() => toggleSort('change')}>
                          {text(language, '24h', '24h', '24j')}
                          {sortMark('change')}
                        </button>
                      </th>
                      <th>
                        <button type="button" onClick={() => toggleSort('volume')}>
                          {text(language, '24h 量', '24h vol', 'Vol 24j')}
                          {sortMark('volume')}
                        </button>
                      </th>
                      <th>{text(language, '类别', 'Category', 'Kategori')}</th>
                      <th>{text(language, '杠杆', 'Lev', 'Lev')}</th>
                      <th>{text(language, '操作', 'Actions', 'Aksi')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((row) => {
                      const change = row.change_24h_pct
                      const changeClass =
                        change == null
                          ? ''
                          : change > 0
                            ? 'up'
                            : change < 0
                              ? 'down'
                              : ''
                      const detailHref = buildMarketSymbolPath(
                        row.exchange || exchange,
                        row.symbol
                      )
                      return (
                        <tr
                          key={`${row.exchange || exchange}:${row.symbol}`}
                          className="markets-row-clickable"
                          tabIndex={0}
                          onClick={() => navigate(detailHref)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                              e.preventDefault()
                              navigate(detailHref)
                            }
                          }}
                        >
                          <td>
                            <div className="markets-sym">
                              <Link
                                to={detailHref}
                                onClick={(e) => e.stopPropagation()}
                              >
                                <strong>{displaySymbol(row)}</strong>
                              </Link>
                              {row.symbol !== displaySymbol(row) && (
                                <small>{row.symbol}</small>
                              )}
                            </div>
                          </td>
                          <td>{formatPrice(row.mark_price)}</td>
                          <td className={changeClass}>{formatChange(change)}</td>
                          <td>{formatVolume(row.volume_24h)}</td>
                          <td>
                            <span className="markets-chip">
                              {row.category || 'crypto'}
                            </span>
                          </td>
                          <td>
                            {row.maxLeverage ? `${row.maxLeverage}x` : '—'}
                          </td>
                          <td>
                            <div
                              className="markets-row-actions"
                              onClick={(e) => e.stopPropagation()}
                            >
                              <Link to={detailHref}>
                                {text(language, '详情', 'Detail', 'Detail')}
                              </Link>
                              {isLoggedIn ? (
                                <>
                                  <Link to={frontierHref(row.symbol)}>
                                    {t('tradeNav', language)}
                                  </Link>
                                  <Link to={chatHref(displaySymbol(row))}>
                                    {t('chatNav', language)}
                                  </Link>
                                </>
                              ) : (
                                <Link to={ROUTES.login}>
                                  {text(language, '登录后交易', 'Sign in', 'Masuk')}
                                </Link>
                              )}
                            </div>
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            )}

            {exchange !== 'hyperliquid' &&
              data?.source === 'curated' &&
              rows.length > 0 && (
              <p className="markets-note">
                {text(
                  language,
                  '暂未能拉取该所实时行情，已回退到常用合约清单（无价格）。请稍后刷新。',
                  'Live quotes from this venue are temporarily unavailable; showing a curated catalog without prices. Try refresh shortly.',
                  'Kutipan live dari venue ini sementara tidak tersedia; menampilkan katalog tanpa harga. Coba muat ulang sebentar lagi.'
                )}
              </p>
            )}
            {exchange !== 'hyperliquid' &&
              data?.source === 'live' &&
              stats.priced > 0 && (
              <p className="markets-note">
                {text(
                  language,
                  `已加载 ${stats.priced} 个 USDT 永续合约实时行情（公开接口，约每 30 秒刷新）。`,
                  `Loaded ${stats.priced} live USDT perpetual markets (public API, ~30s refresh).`,
                  `Memuat ${stats.priced} pasar perpetual USDT live (API publik, refresh ~30 dtk).`
                )}
              </p>
            )}
          </section>

          <aside className="markets-side" aria-label="Market signals">
            <section className="markets-side-card">
              <h2>{text(language, '方向板', 'Direction board', 'Papan arah')}</h2>
              {!isLoggedIn ? (
                <p className="markets-side-empty">
                  {text(
                    language,
                    '登录后可查看 Claw402 / Vergex 方向排行。',
                    'Sign in to load Claw402 / Vergex direction ranks.',
                    'Masuk untuk melihat peringkat arah Claw402 / Vergex.'
                  )}
                </p>
              ) : (direction?.items || []).length === 0 ? (
                <p className="markets-side-empty">
                  {text(
                    language,
                    '暂无方向数据（需配置 Claw402）。',
                    'No direction data yet (Claw402 required).',
                    'Belum ada data arah (perlu Claw402).'
                  )}
                </p>
              ) : (
                <ul className="markets-signal-list">
                  {(direction?.items || []).map((item) => (
                    <li key={`${item.rank}-${item.symbol}`}>
                      <span className="markets-rank">#{item.rank}</span>
                      <strong>{item.symbol}</strong>
                      <span
                        className={
                          item.bias === 'bullish'
                            ? 'up'
                            : item.bias === 'bearish'
                              ? 'down'
                              : ''
                        }
                      >
                        {item.bias}
                      </span>
                      <em>{Number(item.score || 0).toFixed(2)}</em>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <section className="markets-side-card">
              <h2>{text(language, '资金流 (1h)', 'Net flow (1h)', 'Arus bersih (1j)')}</h2>
              {!isLoggedIn ? (
                <p className="markets-side-empty">
                  {text(
                    language,
                    '登录后显示净流入 / 流出榜。',
                    'Sign in to see inflow / outflow leaders.',
                    'Masuk untuk melihat inflow / outflow.'
                  )}
                </p>
              ) : (
                <div className="markets-flow-grid">
                  <div>
                    <h3 className="up">
                      {text(language, '流入', 'Inflow', 'Masuk')}
                    </h3>
                    <ul className="markets-signal-list">
                      {(flow?.data?.inflow || []).slice(0, 5).map((item) => (
                        <li key={`in-${item.key || item.symbol}`}>
                          <strong>{item.symbol}</strong>
                          <em>{item.latestPrice || '—'}</em>
                        </li>
                      ))}
                      {(flow?.data?.inflow || []).length === 0 && (
                        <li className="markets-side-empty">—</li>
                      )}
                    </ul>
                  </div>
                  <div>
                    <h3 className="down">
                      {text(language, '流出', 'Outflow', 'Keluar')}
                    </h3>
                    <ul className="markets-signal-list">
                      {(flow?.data?.outflow || []).slice(0, 5).map((item) => (
                        <li key={`out-${item.key || item.symbol}`}>
                          <strong>{item.symbol}</strong>
                          <em>{item.latestPrice || '—'}</em>
                        </li>
                      ))}
                      {(flow?.data?.outflow || []).length === 0 && (
                        <li className="markets-side-empty">—</li>
                      )}
                    </ul>
                  </div>
                </div>
              )}
            </section>
          </aside>
        </div>
      </div>
    </div>
  )
}
