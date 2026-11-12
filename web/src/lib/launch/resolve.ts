import { api } from '../api'
import type { AIModel, Exchange } from '../../types'
import type { SetupTarget } from './types'

export function modelHasCredential(model: AIModel) {
  return Boolean(
    model.has_api_key ||
    model.apiKey ||
    (model.provider === 'claw402' && model.walletAddress)
  )
}

export function exchangeHasKey(exchange: Exchange) {
  return Boolean(exchange.has_api_key || exchange.apiKey)
}

export function isHyperliquidExchange(exchange: Exchange) {
  return exchange.exchange_type === 'hyperliquid'
}

/** Prefers the claw402 model, falls back to any enabled model with a credential. */
export function pickTradingModel(models: AIModel[]) {
  return (
    models.find(
      (model) =>
        model.provider === 'claw402' &&
        model.enabled &&
        modelHasCredential(model)
    ) ||
    models.find((model) => model.enabled && modelHasCredential(model)) ||
    null
  )
}

function isReadyHyperliquid(exchange: Exchange) {
  return (
    isHyperliquidExchange(exchange) &&
    exchange.enabled &&
    exchangeHasKey(exchange) &&
    Boolean(exchange.hyperliquidBuilderApproved) &&
    (exchange.hyperliquidWalletAddr || '').trim() !== ''
  )
}

function isReadyCex(exchange: Exchange) {
  if (isHyperliquidExchange(exchange)) return false
  // Aster / Lighter need venue-specific fields; treat key presence as enough
  // for launch resolution — server preflight does the deep check.
  if (exchange.exchange_type === 'aster') {
    return (
      exchange.enabled &&
      Boolean(exchange.asterUser && exchange.asterSigner && exchange.has_api_key)
    )
  }
  if (exchange.exchange_type === 'lighter') {
    return (
      exchange.enabled &&
      Boolean(exchange.lighterWalletAddr && exchange.has_api_key)
    )
  }
  return exchange.enabled && exchangeHasKey(exchange)
}

/**
 * Prefer a fully-authorized Hyperliquid account; otherwise the first enabled
 * CEX / venue with credentials so Binance/Bybit users are not blocked by HL-only
 * Autopilot resolution.
 */
export function pickTradingExchange(exchanges: Exchange[]) {
  return (
    exchanges.find(isReadyHyperliquid) ||
    exchanges.find(isReadyCex) ||
    null
  )
}

/**
 * Resolves a launch-capable AI model, auto-provisioning the beginner claw402
 * wallet when none is configured yet. Returns null when nothing could be
 * resolved — the caller routes the user into claw402 setup.
 */
export async function resolveLaunchModel(): Promise<AIModel | null> {
  let models = await api.getModelConfigs()
  let model = pickTradingModel(models)
  if (model) return model

  const onboarding = await api.prepareBeginnerOnboarding()
  models = await api.getModelConfigs()
  model =
    models.find(
      (item) =>
        item.id === onboarding.configured_model_id &&
        item.enabled &&
        modelHasCredential(item)
    ) || pickTradingModel(models)
  if (model) return model

  return model
}

export type ResolveLaunchExchangeResult =
  | { exchange: Exchange }
  | { exchange: null; reason: string; setupTarget: SetupTarget }

/**
 * Resolves a launch-capable exchange. Returns the exchange or a message
 * explaining the most specific missing prerequisite (+ setup deep-link target).
 */
export async function resolveLaunchExchange(): Promise<ResolveLaunchExchangeResult> {
  const exchanges = await api.getExchangeConfigs()
  const ready = pickTradingExchange(exchanges)
  if (ready) return { exchange: ready }

  const hyperliquid = exchanges.find(isHyperliquidExchange)
  const cexCandidate = exchanges.find(
    (ex) => !isHyperliquidExchange(ex) && ex.enabled
  )

  // Prefer guiding the user toward a partially-configured CEX they already added.
  if (cexCandidate) {
    if (!exchangeHasKey(cexCandidate) && cexCandidate.exchange_type !== 'aster') {
      return {
        exchange: null,
        setupTarget: 'exchange',
        reason: `Finish ${cexCandidate.account_name || cexCandidate.exchange_type} API credentials (and IP whitelist if required).`,
      }
    }
    return {
      exchange: null,
      setupTarget: 'exchange',
      reason: `Exchange ${cexCandidate.account_name || cexCandidate.exchange_type} is not ready for trading yet. Open its settings to finish setup.`,
    }
  }

  if (!hyperliquid) {
    return {
      exchange: null,
      setupTarget: 'hyperliquid',
      reason:
        'No trading exchange is connected. Connect Hyperliquid or add a CEX API account first.',
    }
  }
  if (!hyperliquid.enabled) {
    return {
      exchange: null,
      setupTarget: 'hyperliquid',
      reason: 'The Hyperliquid account is disabled. Enable it first.',
    }
  }
  if (!exchangeHasKey(hyperliquid)) {
    return {
      exchange: null,
      setupTarget: 'hyperliquid',
      reason:
        'The Hyperliquid agent key is missing. Reconnect Hyperliquid and save the agent wallet.',
    }
  }
  if (!hyperliquid.hyperliquidBuilderApproved) {
    return {
      exchange: null,
      setupTarget: 'hyperliquid',
      reason:
        'Hyperliquid builder authorization is not complete. Finish wallet authorization first.',
    }
  }
  return {
    exchange: null,
    setupTarget: 'hyperliquid',
    reason:
      'The Hyperliquid wallet address is missing. Reconnect Hyperliquid first.',
  }
}
