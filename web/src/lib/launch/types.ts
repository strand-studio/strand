export type LaunchCheckStatus = 'ok' | 'failed' | 'warning' | 'skipped'

export type LaunchCheckId =
  | 'ai_model'
  | 'ai_wallet'
  | 'ai_wallet_funds'
  | 'strategy'
  | 'exchange_config'
  | 'exchange_account'
  | 'exchange_funds'
  | 'exchange_trade_auth'

export interface LaunchCheck {
  id: LaunchCheckId | string
  status: LaunchCheckStatus
  code?: string
  message?: string
  required?: number
  actual?: number
  asset?: string
  address?: string
  /** Present on exchange-related checks so the UI can deep-link the right venue. */
  exchange_id?: string
  exchange_type?: string
}

export interface LaunchPreflightResult {
  ready: boolean
  checks: LaunchCheck[]
  min_ai_fee_usdc: number
  min_trading_usdc: number
  checked_at: string
}

/**
 * Guided-setup anchor consumed by the traders page (`?setup=`).
 * - claw402: AI fee wallet
 * - hyperliquid / hyperliquid-funds: HL agent / deposit
 * - exchange: CEX / non-HL venue credentials (optional `?exchange_id=`)
 */
export type SetupTarget =
  | 'claw402'
  | 'hyperliquid'
  | 'hyperliquid-funds'
  | 'exchange'

export type LaunchOutcome =
  | { ok: true; traderId: string; warning?: string }
  | {
      ok: false
      kind: 'preflight'
      message: string
      preflight: LaunchPreflightResult
      setupTarget: SetupTarget | null
    }
  | { ok: false; kind: 'setup'; message: string; setupTarget: SetupTarget }
  | { ok: false; kind: 'error'; message: string }
