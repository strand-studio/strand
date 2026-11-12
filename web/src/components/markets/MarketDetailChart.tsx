import { useEffect, useMemo, useRef } from 'react'
import {
  createChart,
  type IChartApi,
  type ISeriesApi,
  type UTCTimestamp,
  CandlestickSeries,
  AreaSeries,
  HistogramSeries,
} from 'lightweight-charts'
import type { Kline } from '../../lib/api/data'

export type ChartMode = 'candle' | 'area'

function toSec(msOrSec: number) {
  return (msOrSec > 1e12 ? Math.floor(msOrSec / 1000) : Math.floor(msOrSec)) as UTCTimestamp
}

export function MarketDetailChart({
  bars,
  mode = 'candle',
  height = 420,
  upColor = '#1f7a4d',
  downColor = '#b42318',
}: {
  bars: Kline[]
  mode?: ChartMode
  height?: number
  upColor?: string
  downColor?: string
}) {
  const ref = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)

  const candleData = useMemo(() => {
    const rows = (bars || [])
      .map((b) => {
        const t = Number(b.openTime)
        const o = Number(b.open)
        const h = Number(b.high)
        const l = Number(b.low)
        const c = Number(b.close)
        const v = Number(b.volume)
        if (![t, o, h, l, c].every(Number.isFinite) || t <= 0) return null
        return {
          time: toSec(t),
          open: o,
          high: h,
          low: l,
          close: c,
          volume: Number.isFinite(v) ? v : 0,
        }
      })
      .filter(Boolean) as Array<{
      time: UTCTimestamp
      open: number
      high: number
      low: number
      close: number
      volume: number
    }>

    // Deduplicate ascending times
    const byTime = new Map<number, (typeof rows)[number]>()
    for (const row of rows) byTime.set(row.time as number, row)
    return [...byTime.values()].sort((a, b) => (a.time as number) - (b.time as number))
  }, [bars])

  useEffect(() => {
    if (!ref.current || candleData.length < 2) return

    const el = ref.current
    const chart = createChart(el, {
      height,
      layout: {
        background: { color: 'transparent' },
        textColor: getComputedStyle(el).getPropertyValue('--mk-muted').trim() || '#637087',
      },
      grid: {
        vertLines: { color: 'rgba(99,112,135,0.12)' },
        horzLines: { color: 'rgba(99,112,135,0.12)' },
      },
      rightPriceScale: { borderVisible: false },
      timeScale: { borderVisible: false, timeVisible: true, secondsVisible: false },
      crosshair: { mode: 1 },
      width: el.clientWidth,
    })

    let priceSeries: ISeriesApi<'Candlestick'> | ISeriesApi<'Area'>
    if (mode === 'area') {
      const last = candleData[candleData.length - 1]
      const first = candleData[0]
      const up = last.close >= first.open
      priceSeries = chart.addSeries(AreaSeries, {
        lineColor: up ? upColor : downColor,
        topColor: up ? `${upColor}55` : `${downColor}55`,
        bottomColor: 'transparent',
        lineWidth: 2,
      })
      priceSeries.setData(candleData.map((d) => ({ time: d.time, value: d.close })))
    } else {
      priceSeries = chart.addSeries(CandlestickSeries, {
        upColor,
        downColor,
        borderUpColor: upColor,
        borderDownColor: downColor,
        wickUpColor: upColor,
        wickDownColor: downColor,
      })
      priceSeries.setData(
        candleData.map((d) => ({
          time: d.time,
          open: d.open,
          high: d.high,
          low: d.low,
          close: d.close,
        }))
      )
    }

    const volumeSeries = chart.addSeries(HistogramSeries, {
      priceFormat: { type: 'volume' },
      priceScaleId: 'vol',
    })
    chart.priceScale('vol').applyOptions({
      scaleMargins: { top: 0.78, bottom: 0 },
    })
    volumeSeries.setData(
      candleData.map((d) => ({
        time: d.time,
        value: d.volume,
        color: d.close >= d.open ? `${upColor}66` : `${downColor}66`,
      }))
    )

    chart.timeScale().fitContent()
    chartRef.current = chart

    const onResize = () => {
      if (ref.current) chart.applyOptions({ width: ref.current.clientWidth })
    }
    window.addEventListener('resize', onResize)
    return () => {
      window.removeEventListener('resize', onResize)
      chart.remove()
      chartRef.current = null
    }
  }, [candleData, mode, height, upColor, downColor])

  if (candleData.length < 2) {
    return <div className="msd-chart-empty">—</div>
  }

  return <div ref={ref} className="msd-chart-canvas" style={{ height }} />
}
