import { API_BASE, httpClient } from '../api/helpers'
import type { LaunchCheck, LaunchPreflightResult, SetupTarget } from './types'

export interface LaunchPreflightRequest {
  ai_model_id: string
  exchange_id: string
  strategy_id?: string
}

/**
 * Server-side launch readiness checks. This is the single source of truth for
 * balances and minimums — never gate the UI on cached client-side values.
 */
export async function runLaunchPreflight(
  request: LaunchPreflightRequest
): Promise<LaunchPreflightResult> {
  const result = await httpClient.post<LaunchPreflightResult>(
    `${API_BASE}/launch/preflight`,
    request
  )
  if (!result.success || !result.data) {
    throw new Error(result.message || 'Failed to run launch preflight')
  }
  return result.data
}

export async function getTraderPreflight(
  traderId: string
): Promise<LaunchPreflightResult> {
  const result = await httpClient.get<LaunchPreflightResult>(
    `${API_BASE}/traders/${traderId}/preflight`
  )
  if (!result.success || !result.data) {
    throw new Error(result.message || 'Failed to run launch preflight')
  }
  return result.data
}

export function failedLaunchChecks(
  result: LaunchPreflightResult
): LaunchCheck[] {
  return result.checks.filter((check) => check.status === 'failed')
}

export function launchWarnings(result: LaunchPreflightResult): LaunchCheck[] {
  return result.checks.filter((check) => check.status === 'warning')
}

/** Human-readable one-liner combining every failing check. */
export function describeLaunchFailures(result: LaunchPreflightResult): string {
  return failedLaunchChecks(result)
    .map((check) => check.message)
    .filter(Boolean)
    .join(' ')
}

function isHyperliquidVenue(check: LaunchCheck): boolean {
  const t = (check.exchange_type || '').toLowerCase()
  const id = (check.exchange_id || '').toLowerCase()
  return t === 'hyperliquid' || id === 'hyperliquid' || id.includes('hyperliquid')
}

/**
 * Maps a failing check to the guided-setup anchor that fixes it. The traders
 * page opens the matching modal/section from the `?setup=` query param.
 */
export function setupTargetForCheck(check: LaunchCheck): SetupTarget | null {
  switch (check.id) {
    case 'ai_model':
    case 'ai_wallet':
    case 'ai_wallet_funds':
      return 'claw402'
    case 'exchange_config':
    case 'exchange_account':
    case 'exchange_trade_auth':
      // HL agent / builder auth → reconnect; CEX → credentials / IP whitelist.
      return isHyperliquidVenue(check) ? 'hyperliquid' : 'exchange'
    case 'exchange_funds':
      return isHyperliquidVenue(check) ? 'hyperliquid-funds' : 'exchange'
    default:
      if (
        check.code === 'permission_denied' ||
        check.code === 'missing_credentials' ||
        check.code === 'invalid_credentials'
      ) {
        return isHyperliquidVenue(check) ? 'hyperliquid' : 'exchange'
      }
      return null
  }
}

/** The setup anchor for the first (highest-priority) failing check. */
export function primarySetupTarget(
  result: LaunchPreflightResult
): SetupTarget | null {
  for (const check of failedLaunchChecks(result)) {
    const target = setupTargetForCheck(check)
    if (target) return target
  }
  return null
}

/** Exchange id from the first failing exchange-related check (for `?exchange_id=`). */
export function primaryFailingExchangeId(
  result: LaunchPreflightResult
): string | null {
  for (const check of failedLaunchChecks(result)) {
    if (
      check.id === 'exchange_config' ||
      check.id === 'exchange_account' ||
      check.id === 'exchange_funds' ||
      check.id === 'exchange_trade_auth'
    ) {
      if (check.exchange_id) return check.exchange_id
    }
  }
  return null
}

/** Build `?setup=…&exchange_id=…` query fragment for traders-page deep links. */
export function setupQueryForPreflight(
  result: LaunchPreflightResult
): string | null {
  const target = primarySetupTarget(result)
  if (!target) return null
  const params = new URLSearchParams()
  params.set('setup', target)
  if (target === 'exchange') {
    const exchangeId = primaryFailingExchangeId(result)
    if (exchangeId) params.set('exchange_id', exchangeId)
  }
  return params.toString()
}
