import { API_BASE, httpClient } from './helpers'

export type ResourceHealthFields = {
  last_test_ok?: boolean
  tested?: boolean
  latency_ms?: number
  last_error?: string
  last_test_at?: string
}

export type ResourceDefaults = {
  default_model_id: string
  default_agent_id: string
  default_exchange_id: string
  alert_platform_ids: string[]
  updated_at?: string
}

export type ResourceAgent = ResourceHealthFields & {
  id: string
  name: string
  kind: string
  configured?: boolean
  agent_type?: string
}

export type ResourcesUsageClaw402 = {
  wallet_address?: string
  balance_usdc?: string
  balance_status?: 'ready' | 'empty' | 'unknown' | string
  est_daily_usd?: number
  est_runway_days?: number
  model_hint?: string
}

export type ResourcesUsageBucket = {
  usd?: number
  calls?: number
}

export type ResourcesUsageSpend = {
  today_usd?: number
  today_calls?: number
  week_usd?: number
  week_calls?: number
  by_model_today?: Record<string, ResourcesUsageBucket | number>
  by_model_week?: Record<string, ResourcesUsageBucket | number>
  by_provider_today?: Record<string, ResourcesUsageBucket | number>
  by_provider_week?: Record<string, ResourcesUsageBucket | number>
}

export type ResourcesUsage = {
  claw402?: ResourcesUsageClaw402
  spend?: ResourcesUsageSpend
}

export type ResourcesOverview = {
  models: Array<
    Record<string, unknown> &
      ResourceHealthFields & {
        id: string
        name?: string
        enabled?: boolean
        provider?: string
        customModelName?: string
        balance_usdc?: string
        balance_status?: string
        spend_today_usd?: number
        spend_today_calls?: number
        spend_week_usd?: number
        spend_week_calls?: number
      }
  >
  agents: ResourceAgent[]
  platforms: Array<
    Record<string, unknown> &
      ResourceHealthFields & {
        id: string
        name?: string
        platform_type?: string
        enabled?: boolean
        alerts_enabled?: boolean
      }
  >
  exchanges: Array<
    Record<string, unknown> &
      ResourceHealthFields & {
        id: string
        name?: string
        account_name?: string
        exchange_type?: string
        enabled?: boolean
      }
  >
  defaults: ResourceDefaults
  usage?: ResourcesUsage
}

export type ResourceTestResult = {
  ok: boolean
  error?: string
  latency_ms?: number
  tested_at?: string
  kind?: string
  id?: string
}

export const resourcesApi = {
  async overview(): Promise<ResourcesOverview> {
    const result = await httpClient.get<ResourcesOverview>(`${API_BASE}/resources`)
    if (!result.success) throw new Error('Failed to load resources')
    return (
      result.data || {
        models: [],
        agents: [],
        platforms: [],
        exchanges: [],
        defaults: {
          default_model_id: '',
          default_agent_id: '',
          default_exchange_id: '',
          alert_platform_ids: [],
        },
      }
    )
  },

  async getDefaults(): Promise<ResourceDefaults> {
    const result = await httpClient.get<ResourceDefaults>(
      `${API_BASE}/resources/defaults`
    )
    if (!result.success) throw new Error('Failed to load resource defaults')
    return (
      result.data || {
        default_model_id: '',
        default_agent_id: '',
        default_exchange_id: '',
        alert_platform_ids: [],
      }
    )
  },

  async saveDefaults(body: Partial<ResourceDefaults>): Promise<ResourceDefaults> {
    const result = await httpClient.request<ResourceDefaults>(
      `${API_BASE}/resources/defaults`,
      { method: 'PUT', data: body }
    )
    if (!result.success) throw new Error(result.error || 'Failed to save defaults')
    return result.data!
  },

  async test(
    kind: 'model' | 'agent' | 'platform' | 'exchange',
    id: string
  ): Promise<ResourceTestResult> {
    const result = await httpClient.request<ResourceTestResult>(
      `${API_BASE}/resources/${encodeURIComponent(kind)}/${encodeURIComponent(id)}/test`,
      { method: 'POST', data: {}, silent: true }
    )
    if (!result.success) {
      const errData = (result as { errorData?: ResourceTestResult }).errorData
      return {
        ok: false,
        error: result.message || errData?.error || 'test failed',
        latency_ms: errData?.latency_ms,
        tested_at: errData?.tested_at,
        kind,
        id,
      }
    }
    return result.data || { ok: true }
  },
}
