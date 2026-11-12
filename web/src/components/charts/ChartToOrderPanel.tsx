import { useState } from 'react'
import { useNavigate } from 'react-router'
import { ArrowRight, Crosshair } from 'lucide-react'
import { ROUTES, buildDashboardPath } from '../../router/paths'

/**
 * Chart → order bridge.
 * - Autopilot: seeds symbol bias via query (AI still decides entries).
 * - Frontier / Trade: opens planning with side + notional and requests compile.
 */
export function ChartToOrderPanel({
  symbol,
  exchange = 'hyperliquid',
  traderId,
  language = 'en',
}: {
  symbol: string
  exchange?: string
  traderId?: string
  language?: string
}) {
  const navigate = useNavigate()
  const isZh = language === 'zh'
  const [side, setSide] = useState<'long' | 'short'>('long')
  const [notional, setNotional] = useState('50')

  const cleanSymbol = (symbol || 'BTC').replace(/USDT$/i, '')

  return (
    <div className="rounded-lg border border-strand-gold/20 bg-strand-bg-lighter p-3">
      <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-strand-text">
        <Crosshair className="h-4 w-4 text-strand-gold" />
        {isZh ? '图到单' : 'Chart to order'}
      </div>
      <p className="mb-3 text-xs leading-5 text-strand-text-muted">
        {isZh
          ? `把 ${cleanSymbol} 的偏向发给自动驾驶，或打开交易页直接编译计划（仍需你确认）。`
          : `Seed Autopilot with a ${cleanSymbol} bias, or open Trade to compile a plan (you still approve).`}
      </p>
      <div className="mb-3 flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => setSide('long')}
          className={`rounded-md px-3 py-1.5 text-xs font-semibold ${
            side === 'long'
              ? 'bg-strand-success/20 text-strand-success'
              : 'bg-strand-bg text-strand-text-muted'
          }`}
        >
          {isZh ? '做多' : 'Long'}
        </button>
        <button
          type="button"
          onClick={() => setSide('short')}
          className={`rounded-md px-3 py-1.5 text-xs font-semibold ${
            side === 'short'
              ? 'bg-strand-danger/20 text-strand-danger'
              : 'bg-strand-bg text-strand-text-muted'
          }`}
        >
          {isZh ? '做空' : 'Short'}
        </button>
        <label className="inline-flex items-center gap-1 text-xs text-strand-text-muted">
          $
          <input
            value={notional}
            onChange={(e) => setNotional(e.target.value)}
            className="w-16 rounded border border-strand-gold/20 bg-strand-bg px-2 py-1 text-strand-text"
          />
        </label>
      </div>
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => {
            const q = new URLSearchParams({
              symbol: cleanSymbol,
              side,
              notional,
              exchange,
              chart_order: '1',
            })
            if (traderId) {
              navigate(`${buildDashboardPath(traderId)}?${q.toString()}`)
            } else {
              navigate(`${ROUTES.traders}?view=autopilot&${q.toString()}`)
            }
          }}
          className="inline-flex items-center gap-1.5 rounded-lg bg-strand-gold px-3 py-2 text-xs font-bold text-white"
        >
          {isZh ? '发给自动驾驶' : 'Send to Autopilot'}
          <ArrowRight className="h-3.5 w-3.5" />
        </button>
        <button
          type="button"
          onClick={() => {
            const q = new URLSearchParams({
              symbol: cleanSymbol,
              side,
              notional,
              venue: exchange,
              compile: '1',
              chart_order: '1',
            })
            navigate(`${ROUTES.planning}?${q.toString()}`)
          }}
          className="inline-flex items-center gap-1.5 rounded-lg border border-strand-gold/25 px-3 py-2 text-xs font-semibold text-strand-gold"
        >
          {isZh ? '编译到交易页' : 'Compile in Trade'}
        </button>
      </div>
    </div>
  )
}
