import type {
  TraderInfo,
  TraderConfigData,
  CreateTraderRequest,
} from '../../types'
import { API_BASE, httpClient } from './helpers'
import { ApiError } from '../httpClient'

const TRADER_LIFECYCLE_TIMEOUT_MS = 120_000

function throwApiError(
  message: string,
  errorKey?: string,
  errorParams?: Record<string, string>,
  statusCode?: number,
  errorData?: Record<string, any>
): never {
  throw new ApiError(message, errorKey, errorParams, statusCode, errorData)
}

export const traderApi = {
  async getTraders(silent?: boolean): Promise<TraderInfo[]> {
    const result = await httpClient.request<TraderInfo[]>(
      `${API_BASE}/my-traders`,
      { silent }
    )
    if (!result.success) throw new Error('Failed to fetch trader list')
    return Array.isArray(result.data) ? result.data : []
  },

  async getPublicTraders(): Promise<any[]> {
    const result = await httpClient.get<any[]>(`${API_BASE}/traders`)
    if (!result.success) throw new Error('Failed to fetch public trader list')
    return result.data!
  },

  async createTrader(request: CreateTraderRequest): Promise<TraderInfo> {
    const result = await httpClient.request<TraderInfo>(`${API_BASE}/traders`, {
      method: 'POST',
      data: request,
      timeout: TRADER_LIFECYCLE_TIMEOUT_MS,
    })
    if (!result.success) {
      throwApiError(
        result.message || 'Failed to create trader',
        result.errorKey,
        result.errorParams,
        result.statusCode
      )
    }
    return result.data!
  },

  async deleteTrader(traderId: string): Promise<void> {
    const result = await httpClient.delete(`${API_BASE}/traders/${traderId}`)
    if (!result.success) throw new Error('Failed to delete trader')
  },

  async startTrader(traderId: string): Promise<void> {
    const result = await httpClient.request(
      `${API_BASE}/traders/${traderId}/start`,
      { method: 'POST', timeout: TRADER_LIFECYCLE_TIMEOUT_MS }
    )
    if (!result.success) {
      throwApiError(
        result.message || 'Failed to start trader',
        result.errorKey,
        result.errorParams,
        result.statusCode,
        result.errorData
      )
    }
  },

  async stopTrader(traderId: string): Promise<void> {
    const result = await httpClient.request(
      `${API_BASE}/traders/${traderId}/stop`,
      { method: 'POST', timeout: TRADER_LIFECYCLE_TIMEOUT_MS }
    )
    if (!result.success) {
      throwApiError(
        result.message || 'Failed to stop trader',
        result.errorKey,
        result.errorParams,
        result.statusCode,
        result.errorData
      )
    }
  },

  async pauseTrader(
    traderId: string,
    opts?: { hours?: number; reason?: string }
  ): Promise<void> {
    const result = await httpClient.request(
      `${API_BASE}/traders/${traderId}/pause`,
      {
        method: 'POST',
        data: {
          hours: opts?.hours ?? 24,
          reason: opts?.reason || 'Paused from Portfolio',
        },
        timeout: TRADER_LIFECYCLE_TIMEOUT_MS,
      }
    )
    if (!result.success) {
      throwApiError(
        result.message || 'Failed to pause trader',
        result.errorKey,
        result.errorParams,
        result.statusCode,
        result.errorData
      )
    }
  },

  async resumeTrader(traderId: string): Promise<void> {
    const result = await httpClient.request(
      `${API_BASE}/traders/${traderId}/resume`,
      { method: 'POST', timeout: TRADER_LIFECYCLE_TIMEOUT_MS }
    )
    if (!result.success) {
      throwApiError(
        result.message || 'Failed to resume trader',
        result.errorKey,
        result.errorParams,
        result.statusCode,
        result.errorData
      )
    }
  },

  async updateTraderRuntime(
    traderId: string,
    scanIntervalMinutes?: number,
    leverage?: { btcEth?: number; altcoin?: number }
  ): Promise<{ applied_live?: boolean }> {
    const body: Record<string, number> = {}
    if (scanIntervalMinutes != null && scanIntervalMinutes > 0) {
      body.scan_interval_minutes = scanIntervalMinutes
    }
    if (leverage?.btcEth != null && leverage.btcEth > 0) {
      body.btc_eth_leverage = leverage.btcEth
    }
    if (leverage?.altcoin != null && leverage.altcoin > 0) {
      body.altcoin_leverage = leverage.altcoin
    }
    const result = await httpClient.put<{ applied_live?: boolean }>(
      `${API_BASE}/traders/${traderId}/runtime`,
      body
    )
    if (!result.success) {
      throwApiError(
        result.message || 'Failed to update runtime',
        result.errorKey,
        result.errorParams,
        result.statusCode,
        result.errorData
      )
    }
    return result.data || {}
  },

  async toggleCompetition(
    traderId: string,
    showInCompetition: boolean
  ): Promise<void> {
    const result = await httpClient.put(
      `${API_BASE}/traders/${traderId}/competition`,
      { show_in_competition: showInCompetition }
    )
    if (!result.success)
      throw new Error('Failed to update competition visibility')
  },

  async closePosition(
    traderId: string,
    symbol: string,
    side: string
  ): Promise<{ message: string }> {
    const result = await httpClient.post<{ message: string }>(
      `${API_BASE}/traders/${traderId}/close-position`,
      { symbol, side }
    )
    if (!result.success) throw new Error('Failed to close position')
    return result.data!
  },

  async updateTraderPrompt(
    traderId: string,
    customPrompt: string
  ): Promise<void> {
    const result = await httpClient.put(
      `${API_BASE}/traders/${traderId}/prompt`,
      { custom_prompt: customPrompt }
    )
    if (!result.success) throw new Error('Failed to update custom prompt')
  },

  async getTraderConfig(
    traderId: string,
    silent?: boolean
  ): Promise<TraderConfigData> {
    const result = await httpClient.request<TraderConfigData>(
      `${API_BASE}/traders/${traderId}/config`,
      { silent }
    )
    if (!result.success) throw new Error('Failed to fetch trader config')
    return result.data!
  },

  async updateTrader(
    traderId: string,
    request: CreateTraderRequest
  ): Promise<TraderInfo> {
    const result = await httpClient.request<TraderInfo>(
      `${API_BASE}/traders/${traderId}`,
      { method: 'PUT', data: request, timeout: TRADER_LIFECYCLE_TIMEOUT_MS }
    )
    if (!result.success) {
      throwApiError(
        result.message || 'Failed to update trader',
        result.errorKey,
        result.errorParams,
        result.statusCode
      )
    }
    return result.data!
  },
}
