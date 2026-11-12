import { useMemo, useState } from 'react'
import type { DecisionRecord } from '../../types'
import { formatRequestError } from '../../lib/requestErrors'
import { useLanguage } from '../../contexts/LanguageContext'

/**
 * ExecutionLog renders the AI trading agent's real decisions and order results
 * as a high-density, newest-first terminal stream — Bloomberg-style log on the
 * cream paper theme. Each cycle is a clearly-delimited block: a header bar
 * (cycle no · time · action count), one badge line per AI action, and one
 * indented sub-line per execution-log entry, color-coded by
 * success / throttle / risk.
 *
 * Real data only — renders each DecisionRecord without changing stored history;
 * verbose throttle and transport errors are summarized without inventing results.
 */

const C_AMBER = '#c8860b' // throttle / block warnings

// Strip dex prefix + quote suffix to the bare base ticker.
function baseSymbol(raw: string): string {
  return raw
    .toUpperCase()
    .replace(/^XYZ:/, '')
    .replace(/(USDT|USDC|USD)$/, '')
}

// Include the local date: old sessions must not appear to be today's trades.
function fmtTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '--:--:--'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

type Side = 'long' | 'short' | 'flat'

// Trade side of an action token — drives badge color.
function actionSide(action: string): Side {
  const a = action.toLowerCase()
  if (a === 'open_long' || a === 'close_short') return 'long'
  if (a === 'open_short' || a === 'close_long') return 'short'
  return 'flat'
}

function sideColor(side: Side): string {
  if (side === 'long') return 'var(--tm-up)'
  if (side === 'short') return 'var(--tm-dn)'
  return 'var(--tm-muted)'
}

type LogTone = 'ok' | 'warn' | 'risk' | 'info'

// Classify an execution-log line into a tone for color-coding.
function logTone(line: string): LogTone {
  const s = line.toLowerCase()
  if (s.includes('succeed') || s.includes('success') || line.includes('✓'))
    return 'ok'
  if (
    s.includes('throttle') ||
    s.includes('re-entry') ||
    s.includes('cooldown') ||
    s.includes('blocked')
  )
    return 'warn'
  if (
    s.includes('risk') ||
    s.includes('fail') ||
    s.includes('reject') ||
    s.includes('error') ||
    s.includes('denied') ||
    line.includes('✗') ||
    line.includes('❌')
  )
    return 'risk'
  return 'info'
}

const TONE_COLOR: Record<LogTone, string> = {
  ok: 'var(--tm-up)',
  warn: C_AMBER,
  risk: 'var(--tm-dn)',
  info: 'var(--tm-ink-2)',
}
const TONE_GLYPH: Record<LogTone, string> = {
  ok: '✓',
  warn: '⚠',
  risk: '❌',
  info: '·',
}

/**
 * Tidy a verbose execution-log string to its gist without fabricating data.
 * Strips a leading "└"/glyph the backend may already include (we render our
 * own), the symbol (already shown), and collapses the wordy throttle phrasing
 * to "throttle · closed 28m ago, wait 2m". Falls back to the original line.
 */
function cleanLog(
  raw: string,
  language: 'en' | 'zh' | 'id' = 'en'
): string {
  let s = raw.replace(/^[\s└>•·]*[✓✗⚠❌]?\s*/, '').trim()

  const throttle = s.match(
    /closed\s+([0-9smhd.]+)\s+ago;\s*wait\s+([0-9smhd.]+)/i
  )
  if (throttle) {
    const ago = throttle[1].replace(/(\d)0s$/, '$1').replace(/0s$/, '')
    const wait = throttle[2].replace(/(\d)0s$/, '$1').replace(/0s$/, '')
    return `throttle · closed ${ago} ago, wait ${wait}`
  }
  // drop a redundant leading "SYMBOL action" prefix when present
  s = s.replace(
    /^[A-Z0-9:_-]{2,12}\s+(open_long|open_short|close_long|close_short)\s+/i,
    ''
  )
  return formatRequestError(s, language === 'zh' ? 'zh' : 'en')
}

function isFaultCycle(record: DecisionRecord): boolean {
  const actions = record.decisions ?? []
  const failedTrades = actions.filter(
    (a) => a.action !== 'hold' && a.action !== 'wait' && !a.success
  ).length
  return !record.success || failedTrades > 0 || Boolean(record.error_message)
}

interface ExecutionLogProps {
  decisions?: DecisionRecord[]
  height?: number
}

export function ExecutionLog({ decisions, height = 440 }: ExecutionLogProps) {
  const { language } = useLanguage()
  const [failuresOnly, setFailuresOnly] = useState(false)

  // Fault cycles first, then newest cycle number.
  const cycles = useMemo(() => {
    const list = decisions ?? []
    const sorted = [...list].sort((a, b) => {
      const af = isFaultCycle(a) ? 1 : 0
      const bf = isFaultCycle(b) ? 1 : 0
      if (af !== bf) return bf - af
      return b.cycle_number - a.cycle_number
    })
    return failuresOnly ? sorted.filter(isFaultCycle) : sorted
  }, [decisions, failuresOnly])

  const faultCount = useMemo(
    () => (decisions ?? []).filter(isFaultCycle).length,
    [decisions]
  )

  return (
    <div style={{ fontFamily: 'var(--tm-mono)' }}>
      {/* header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'baseline',
          gap: 8,
          marginBottom: 2,
        }}
      >
        <span className="tm-px" style={{ fontSize: 11 }}>
          {language === 'zh' ? '执行日志' : 'Execution log'}
        </span>
        <button
          type="button"
          className="tm-mono"
          onClick={() => setFailuresOnly((v) => !v)}
          style={{
            marginLeft: 'auto',
            fontSize: 9,
            padding: '2px 6px',
            border: `1px solid ${failuresOnly ? 'var(--tm-dn)' : 'var(--tm-hair)'}`,
            color: failuresOnly ? 'var(--tm-dn)' : 'var(--tm-muted)',
            background: 'transparent',
            cursor: 'pointer',
          }}
        >
          {failuresOnly
            ? language === 'zh'
              ? `仅故障 · ${faultCount}`
              : `Faults only · ${faultCount}`
            : language === 'zh'
              ? `全部 · ${faultCount} 故障`
              : `All · ${faultCount} fault`}
        </button>
        <span
          className="tm-sc"
          style={{
            color: cycles.length ? 'var(--tm-up)' : 'var(--tm-muted)',
          }}
        >
          {cycles.length ? `${cycles.length} cyc` : '—'}
        </span>
      </div>
      <div className="tm-sc" style={{ fontSize: 9, marginBottom: 5 }}>
        {language === 'zh'
          ? '执行日志 · 每周期 AI 决策与成交 · 故障优先'
          : 'Execution log · AI decisions & fills per cycle · faults first'}
      </div>

      {/* legend */}
      <div
        className="tm-sc"
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: 12,
          marginBottom: 6,
          fontSize: 9,
        }}
      >
        <Legend
          glyph="✓"
          c="var(--tm-up)"
          label={language === 'zh' ? '成功' : 'ok'}
        />
        <Legend
          glyph="⚠"
          c={C_AMBER}
          label={language === 'zh' ? '节流' : 'throttle'}
        />
        <Legend
          glyph="❌"
          c="var(--tm-dn)"
          label={language === 'zh' ? '风险' : 'risk'}
        />
      </div>

      <div className="tm-hair" style={{ marginBottom: 0 }} />

      {!cycles.length ? (
        <div className="tm-sc" style={{ padding: '16px 0' }}>
          {failuresOnly
            ? language === 'zh'
              ? '当前窗口无故障周期。'
              : 'No failed cycles in this window.'
            : language === 'zh'
              ? '暂无执行事件 — 启动 Autopilot 后可见首个周期。'
              : 'No execution events yet — start Autopilot to see the first cycle.'}
        </div>
      ) : (
        <div
          style={{
            height,
            overflowY: 'auto',
            fontSize: 10,
            lineHeight: 1.5,
            fontVariantNumeric: 'tabular-nums',
          }}
        >
          {cycles.map((c) => (
            <Cycle
              key={`${c.cycle_number}-${c.timestamp}`}
              record={c}
              language={language}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function Legend({
  glyph,
  c,
  label,
}: {
  glyph: string
  c: string
  label: string
}) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
      <span style={{ color: c, fontSize: 10 }}>{glyph}</span>
      <span>{label}</span>
    </span>
  )
}

interface CycleProps {
  record: DecisionRecord
  language: 'en' | 'zh' | 'id'
}

function Cycle({ record, language }: CycleProps) {
  const [openDetail, setOpenDetail] = useState(false)
  const time = fmtTime(record.timestamp)
  const actions = record.decisions ?? []
  const logs = record.execution_log ?? []
  const count = actions.length
  const trades = actions.filter(
    (a) => a.action !== 'hold' && a.action !== 'wait'
  )
  const failed = trades.filter((a) => !a.success).length
  // Also correct the display of older records whose aggregate flag was wrong.
  const fault = isFaultCycle(record)
  const equity =
    record.account_state?.total_balance != null &&
    record.account_state.total_balance > 0
      ? record.account_state.total_balance
      : null
  const hasDetail = Boolean(
    record.cot_trace ||
      record.error_message ||
      actions.some((a) => a.reasoning || a.error)
  )

  return (
    <div style={{ borderBottom: '1px solid var(--tm-hair)', padding: '5px 0' }}>
      {/* cycle header bar */}
      <div
        className="tm-sc"
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 7,
          fontSize: 9,
          padding: '2px 6px',
          marginBottom:
            actions.length || logs.length || record.error_message ? 4 : 0,
          background: fault
            ? 'rgba(214,67,58,0.06)'
            : 'rgba(25,36,52,0.045)',
          borderLeft: `2px solid ${fault ? 'var(--tm-dn)' : 'var(--tm-hair)'}`,
          color: 'var(--tm-ink-2)',
        }}
      >
        <span style={{ color: 'var(--tm-ink)', fontWeight: 700 }}>
          CYCLE {record.cycle_number}
        </span>
        <span style={{ color: 'var(--tm-muted)' }}>·</span>
        <span style={{ color: 'var(--tm-muted)' }} title="Local time">
          {time}
        </span>
        {equity != null ? (
          <>
            <span style={{ color: 'var(--tm-muted)' }}>·</span>
            <span style={{ color: 'var(--tm-ink-2)' }}>
              eq {equity.toFixed(2)}
            </span>
          </>
        ) : null}
        <span style={{ marginLeft: 'auto', color: 'var(--tm-muted)' }}>
          {count === 0 ? 'no action' : `${count} action${count > 1 ? 's' : ''}`}
        </span>
        {fault ? (
          <span style={{ color: 'var(--tm-dn)', fontWeight: 700 }}>
            FAULT
            {failed > 0
              ? ` · ${failed}/${trades.length} trades failed/blocked`
              : ''}
          </span>
        ) : null}
        {hasDetail ? (
          <button
            type="button"
            className="tm-mono"
            onClick={() => setOpenDetail((v) => !v)}
            style={{
              fontSize: 9,
              padding: '0 5px',
              border: '1px solid var(--tm-hair)',
              color: 'var(--tm-muted)',
              background: 'transparent',
              cursor: 'pointer',
            }}
          >
            {openDetail
              ? language === 'zh'
                ? '收起'
                : 'hide'
              : language === 'zh'
                ? '原因'
                : 'why'}
          </button>
        ) : null}
      </div>

      {/* AI action lines */}
      {actions.map((a, i) => {
        const side = actionSide(a.action)
        const aTime = fmtTime(a.timestamp)
        return (
          <div key={`act-${i}`}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '1px 6px',
                color: 'var(--tm-ink-2)',
              }}
            >
              <span
                style={{
                  color: 'var(--tm-muted)',
                  flex: '0 0 auto',
                  minWidth: 52,
                }}
              >
                {aTime !== '--:--:--' ? aTime : time}
              </span>
              <ActionBadge action={a.action} side={side} />
              <span
                style={{
                  color: 'var(--tm-ink)',
                  fontWeight: 600,
                  flex: '0 0 auto',
                  minWidth: 48,
                }}
              >
                {baseSymbol(a.symbol)}
              </span>
              {a.confidence != null ? (
                <span style={{ color: 'var(--tm-muted)', flex: '0 0 auto' }}>
                  conf
                  <span style={{ color: 'var(--tm-ink-2)' }}>
                    {Math.round(a.confidence)}
                  </span>
                </span>
              ) : null}
              {!a.success && a.action !== 'hold' && a.action !== 'wait' ? (
                <span style={{ color: 'var(--tm-dn)', fontWeight: 700 }}>
                  FAIL
                </span>
              ) : null}
            </div>
            {a.error ? (
              <SubLine tone="risk" text={cleanLog(a.error, language)} />
            ) : null}
          </div>
        )
      })}

      {/* execution-log result sub-lines */}
      {logs.map((line, i) => (
        <SubLine
          key={`log-${i}`}
          tone={logTone(line)}
          text={cleanLog(line, language)}
        />
      ))}

      {/* fault message */}
      {record.error_message ? (
        <SubLine
          tone="risk"
          text={cleanLog(record.error_message, language)}
        />
      ) : null}

      {openDetail && hasDetail ? (
        <div
          style={{
            margin: '4px 6px 2px',
            padding: '6px 8px',
            border: '1px solid var(--tm-hair)',
            background: 'rgba(25,36,52,0.03)',
            fontSize: 9,
            color: 'var(--tm-ink-2)',
            whiteSpace: 'pre-wrap',
            overflowWrap: 'anywhere',
            maxHeight: 180,
            overflowY: 'auto',
          }}
        >
          {record.error_message ? (
            <div style={{ marginBottom: 6, color: 'var(--tm-dn)' }}>
              <strong>Error</strong>
              {'\n'}
              {cleanLog(record.error_message, language)}
            </div>
          ) : null}
          {actions
            .filter((a) => a.reasoning || a.error)
            .map((a, i) => (
              <div key={`why-${i}`} style={{ marginBottom: 6 }}>
                <strong>
                  {a.action} {baseSymbol(a.symbol)}
                </strong>
                {a.error ? (
                  <div style={{ color: 'var(--tm-dn)' }}>
                    {cleanLog(a.error, language)}
                  </div>
                ) : null}
                {a.reasoning ? <div>{a.reasoning}</div> : null}
              </div>
            ))}
          {record.cot_trace ? (
            <div>
              <strong>Chain of thought</strong>
              {'\n'}
              {record.cot_trace}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

function ActionBadge({ action, side }: { action: string; side: Side }) {
  const c = sideColor(side)
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        flex: '0 0 auto',
        padding: '0 5px',
        height: 14,
        fontSize: 9,
        letterSpacing: '0.04em',
        color: c,
        border: `1px solid ${c}`,
        background:
          side === 'long'
            ? 'rgba(46,139,87,0.08)'
            : side === 'short'
              ? 'rgba(214,67,58,0.08)'
              : 'transparent',
        borderRadius: 2,
        whiteSpace: 'nowrap',
      }}
    >
      {action.toLowerCase()}
    </span>
  )
}

function SubLine({ tone, text }: { tone: LogTone; text: string }) {
  const color = TONE_COLOR[tone]
  const glyph = TONE_GLYPH[tone]
  return (
    <div
      style={{
        display: 'flex',
        gap: 6,
        padding: '0 6px 0 10px',
        marginLeft: 6,
        borderLeft: '1px solid var(--tm-hair)',
        color,
      }}
    >
      <span style={{ flex: '0 0 auto', width: 10, textAlign: 'center' }}>
        {glyph}
      </span>
      <span style={{ minWidth: 0, overflowWrap: 'anywhere' }}>{text}</span>
    </div>
  )
}

export default ExecutionLog
