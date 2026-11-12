import type { DecisionRecord, SystemStatus } from '../../types'
import type { Language } from '../../i18n/translations'
import { formatRequestError } from '../../lib/requestErrors'

type FailureKind = 'lifecycle' | 'order' | 'ai' | 'wallet' | 'risk'

interface FailureItem {
  kind: FailureKind
  title: string
  detail: string
  at?: string
  hint?: string
}

function kindLabel(kind: FailureKind, language: Language): string {
  if (language === 'zh') {
    switch (kind) {
      case 'lifecycle':
        return '启停'
      case 'order':
        return '下单'
      case 'ai':
        return 'AI'
      case 'wallet':
        return '手续费钱包'
      case 'risk':
        return '风控'
    }
  }
  if (language === 'id') {
    switch (kind) {
      case 'lifecycle':
        return 'Start/Stop'
      case 'order':
        return 'Order'
      case 'ai':
        return 'AI'
      case 'wallet':
        return 'Dompet biaya'
      case 'risk':
        return 'Risiko'
    }
  }
  switch (kind) {
    case 'lifecycle':
      return 'Lifecycle'
    case 'order':
      return 'Order'
    case 'ai':
      return 'AI'
    case 'wallet':
      return 'Fee wallet'
    case 'risk':
      return 'Risk'
  }
}

function collectFailures(
  status: SystemStatus | undefined,
  decisions: DecisionRecord[] | undefined,
  language: Language
): FailureItem[] {
  const items: FailureItem[] = []
  if (!status) return items

  if (status.last_runtime_error) {
    items.push({
      kind: 'lifecycle',
      title: language === 'zh' ? '运行时崩溃 / 停止' : 'Runtime crash / stop',
      detail: formatRequestError(status.last_runtime_error, language),
      at: status.last_runtime_error_at,
      hint:
        language === 'zh'
          ? '检查交易所凭证、网络与策略配置后重新启动。'
          : 'Check exchange credentials, network, and strategy, then restart.',
    })
  }

  if (status.trading_blocked) {
    items.push({
      kind: 'order',
      title: language === 'zh' ? '交易授权阻断' : 'Trading authorization blocked',
      detail: formatRequestError(status.trading_error, language),
      at: status.trading_checked_at,
      hint:
        language === 'zh'
          ? '用主钱包重新签名 Agent / builder 授权。'
          : 'Re-authorize the agent / builder fee with the main wallet.',
    })
  }

  if (status.last_order_error) {
    const sym = status.last_order_error_symbol
      ? ` · ${status.last_order_error_symbol}`
      : ''
    const act = status.last_order_error_action
      ? ` (${status.last_order_error_action})`
      : ''
    items.push({
      kind: 'order',
      title:
        language === 'zh'
          ? `最近下单失败${sym}${act}`
          : `Latest order failure${sym}${act}`,
      detail: formatRequestError(status.last_order_error, language),
      at: status.last_order_error_at,
      hint:
        language === 'zh'
          ? '常见原因：杠杆/保证金不足、符号不可用、最小名义金额。'
          : 'Common causes: leverage/margin, unavailable symbol, min notional.',
    })
  }

  if (status.last_ai_error) {
    items.push({
      kind: 'ai',
      title:
        language === 'zh'
          ? `AI 决策失败${status.last_ai_error_cycle ? ` · 周期 #${status.last_ai_error_cycle}` : ''}`
          : `AI decision failed${status.last_ai_error_cycle ? ` · cycle #${status.last_ai_error_cycle}` : ''}`,
      detail: formatRequestError(status.last_ai_error, language),
      at: status.last_ai_error_at,
      hint:
        language === 'zh'
          ? '检查 AI 模型密钥、Base URL，以及 Claw402 手续费钱包余额。'
          : 'Check AI keys / Base URL and Claw402 fee-wallet balance.',
    })
  } else if (decisions?.length) {
    const failed = [...decisions]
      .reverse()
      .find((d) => !d.success && d.error_message)
    if (failed?.error_message) {
      items.push({
        kind: 'ai',
        title:
          language === 'zh'
            ? `AI 决策失败 · 周期 #${failed.cycle_number}`
            : `AI decision failed · cycle #${failed.cycle_number}`,
        detail: formatRequestError(failed.error_message, language),
        at: failed.timestamp,
      })
    }
    const orderFail = [...decisions]
      .flatMap((d) => d.decisions || [])
      .reverse()
      .find((a) => !a.success && a.error)
    if (orderFail?.error && !status.last_order_error) {
      items.push({
        kind: 'order',
        title:
          language === 'zh'
            ? `下单失败 · ${orderFail.symbol}`
            : `Order failed · ${orderFail.symbol}`,
        detail: formatRequestError(orderFail.error, language),
        at: orderFail.timestamp,
      })
    }
  }

  if (
    status.ai_wallet_status === 'empty' ||
    status.ai_wallet_status === 'low'
  ) {
    items.push({
      kind: 'wallet',
      title:
        status.ai_wallet_status === 'empty'
          ? language === 'zh'
            ? 'AI 手续费钱包余额不足'
            : 'AI fee wallet empty'
          : language === 'zh'
            ? 'AI 手续费钱包余额偏低'
            : 'AI fee wallet low',
      detail: `${(status.ai_wallet_balance_usdc ?? 0).toFixed(2)} USDC`,
      at: status.ai_wallet_checked_at,
      hint:
        language === 'zh'
          ? '向 Base 上的手续费钱包充值 USDC。'
          : 'Top up USDC on the Base fee wallet.',
    })
  }

  if (status.safe_mode) {
    items.push({
      kind: 'risk',
      title: language === 'zh' ? '安全模式' : 'Safe mode',
      detail: formatRequestError(status.safe_mode_reason, language),
      hint:
        language === 'zh'
          ? '连续 AI 失败后暂停开仓；修复根因后会自动恢复。'
          : 'New entries paused after repeated AI failures; recovers after root cause is fixed.',
    })
  }

  if (status.paused) {
    items.push({
      kind: 'risk',
      title: language === 'zh' ? '交易已暂停' : 'Trading paused',
      detail: formatRequestError(status.pause_reason, language),
      hint:
        language === 'zh'
          ? '日亏/回撤硬闸或手动暂停；可在「资产」页恢复。'
          : 'Daily-loss / drawdown gate or manual pause — resume from Portfolio.',
    })
  }

  return items
}

export function FailureAttributionPanel({
  status,
  decisions,
  language,
}: {
  status?: SystemStatus
  decisions?: DecisionRecord[]
  language: Language
}) {
  const items = collectFailures(status, decisions, language)
  if (items.length === 0) return null

  return (
    <div
      className="tm-mono"
      style={{
        margin: '8px 14px 0',
        padding: '12px',
        fontSize: 12,
        border: '1px solid var(--tm-dn)',
        background: 'rgba(200,60,40,0.05)',
      }}
      role="region"
      aria-label={
        language === 'zh' ? '失败归因' : 'Failure attribution'
      }
    >
      <div
        style={{
          fontWeight: 700,
          color: 'var(--tm-dn)',
          marginBottom: 8,
          letterSpacing: '0.04em',
        }}
      >
        {language === 'zh' ? '失败归因' : 'Failure attribution'}
      </div>
      <div style={{ display: 'grid', gap: 8 }}>
        {items.map((item, idx) => (
          <div
            key={`${item.kind}-${idx}`}
            style={{
              display: 'grid',
              gridTemplateColumns: '88px 1fr',
              gap: 8,
              alignItems: 'start',
            }}
          >
            <span
              style={{
                fontSize: 10,
                fontWeight: 700,
                letterSpacing: '0.06em',
                textTransform: 'uppercase',
                color: 'var(--tm-dn)',
                opacity: 0.85,
              }}
            >
              {kindLabel(item.kind, language)}
            </span>
            <div>
              <div style={{ color: 'var(--tm-ink)', fontWeight: 600 }}>
                {item.title}
              </div>
              <div className="tm-error" style={{ marginTop: 2 }}>
                {item.detail || '—'}
              </div>
              {item.hint ? (
                <div style={{ marginTop: 2, opacity: 0.75, fontSize: 11 }}>
                  {item.hint}
                </div>
              ) : null}
              {item.at ? (
                <div style={{ marginTop: 2, opacity: 0.55, fontSize: 10 }}>
                  {item.at}
                </div>
              ) : null}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
