import { describe, expect, it } from 'vitest'
import type { LaunchPreflightResult } from './types'
import {
  primaryFailingExchangeId,
  primarySetupTarget,
  setupQueryForPreflight,
  setupTargetForCheck,
} from './preflight'

function result(
  checks: LaunchPreflightResult['checks']
): LaunchPreflightResult {
  return {
    ready: false,
    checks,
    min_ai_fee_usdc: 1,
    min_trading_usdc: 12,
    checked_at: new Date().toISOString(),
  }
}

describe('setupTargetForCheck', () => {
  it('routes Hyperliquid trade-auth to hyperliquid', () => {
    expect(
      setupTargetForCheck({
        id: 'exchange_trade_auth',
        status: 'failed',
        exchange_id: 'hl-1',
        exchange_type: 'hyperliquid',
      })
    ).toBe('hyperliquid')
  })

  it('routes CEX account failure to exchange', () => {
    expect(
      setupTargetForCheck({
        id: 'exchange_account',
        status: 'failed',
        code: 'permission_denied',
        exchange_id: 'bn-1',
        exchange_type: 'binance',
      })
    ).toBe('exchange')
  })

  it('routes HL funds shortfall to hyperliquid-funds', () => {
    expect(
      setupTargetForCheck({
        id: 'exchange_funds',
        status: 'failed',
        exchange_type: 'hyperliquid',
      })
    ).toBe('hyperliquid-funds')
  })

  it('routes CEX funds shortfall to exchange', () => {
    expect(
      setupTargetForCheck({
        id: 'exchange_funds',
        status: 'failed',
        exchange_id: 'by-1',
        exchange_type: 'bybit',
      })
    ).toBe('exchange')
  })
})

describe('setupQueryForPreflight', () => {
  it('builds exchange deep-link with exchange_id', () => {
    const q = setupQueryForPreflight(
      result([
        {
          id: 'exchange_account',
          status: 'failed',
          exchange_id: 'ex-okx-9',
          exchange_type: 'okx',
          message: 'IP blocked',
        },
      ])
    )
    expect(q).toBe('setup=exchange&exchange_id=ex-okx-9')
    expect(primaryFailingExchangeId(result([{
      id: 'exchange_account',
      status: 'failed',
      exchange_id: 'ex-okx-9',
      exchange_type: 'okx',
    }]))).toBe('ex-okx-9')
  })

  it('prefers claw402 when AI wallet fails first', () => {
    expect(
      primarySetupTarget(
        result([
          { id: 'ai_wallet_funds', status: 'failed', message: 'empty' },
          {
            id: 'exchange_account',
            status: 'failed',
            exchange_type: 'binance',
            exchange_id: 'bn-1',
          },
        ])
      )
    ).toBe('claw402')
  })
})
