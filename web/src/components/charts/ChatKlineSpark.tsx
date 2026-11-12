import { useEffect, useMemo, useRef } from 'react'
import {
  createChart,
  type IChartApi,
  type UTCTimestamp,
  AreaSeries,
} from 'lightweight-charts'

type Bar = {
  openTime?: number
  time?: number
  close?: number | string
}

export function ChatKlineSpark({
  symbol,
  bars,
  height = 160,
}: {
  symbol: string
  bars: Bar[]
  height?: number
}) {
  const ref = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)

  const points = useMemo(() => {
    return (bars || [])
      .map((b) => {
        const t = Number(b.openTime ?? b.time ?? 0)
        const c = Number(b.close)
        if (!Number.isFinite(t) || !Number.isFinite(c) || t <= 0) return null
        const sec = t > 1e12 ? Math.floor(t / 1000) : Math.floor(t)
        return { time: sec as UTCTimestamp, value: c }
      })
      .filter(Boolean) as { time: UTCTimestamp; value: number }[]
  }, [bars])

  useEffect(() => {
    if (!ref.current || points.length < 2) return
    const chart = createChart(ref.current, {
      height,
      layout: {
        background: { color: 'transparent' },
        textColor: '#94a3b8',
      },
      grid: {
        vertLines: { visible: false },
        horzLines: { color: 'rgba(148,163,184,0.12)' },
      },
      rightPriceScale: { borderVisible: false },
      timeScale: { borderVisible: false },
      width: ref.current.clientWidth,
    })
    const series = chart.addSeries(AreaSeries, {
      lineColor: '#d4a017',
      topColor: 'rgba(212,160,23,0.28)',
      bottomColor: 'rgba(212,160,23,0.02)',
      lineWidth: 2,
    })
    series.setData(points)
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
  }, [points, height])

  if (points.length < 2) return null

  return (
    <div className="ai-chat-kline">
      <div className="ai-chat-kline-label">{symbol || 'Chart'}</div>
      <div ref={ref} style={{ width: '100%', height }} />
    </div>
  )
}
