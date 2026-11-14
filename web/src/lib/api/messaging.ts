import { API_BASE, httpClient } from './helpers'

export type MessagingField = {
  key: string
  label: string
  secret?: boolean
  required?: boolean
  placeholder?: string
  help?: string
}

export type MessagingPlatformSchema = {
  type: string
  label: string
  description: string
  fields: MessagingField[]
  chat: boolean
  alerts: boolean
  bound_peer_placeholder?: string
  bound_peer_help?: string
}

export type MessagingWorkspace = {
  user_id: string
  agent_type: string
  agent_model: string
  work_dir: string
  language: string
}

export type MessagingChannel = {
  id: string
  name: string
  platform_type: string
  chat_enabled: boolean
  alerts_enabled: boolean
  enabled: boolean
  allow_from?: string
  bound_peer?: string
  status: string
  last_error?: string
  credentials?: Record<string, string>
  last_test_ok?: boolean
  tested?: boolean
  latency_ms?: number
  last_test_at?: string
}

export const messagingApi = {
  async getSupported(): Promise<{
    platforms: MessagingPlatformSchema[]
    agents: string[]
  }> {
    const result = await httpClient.get<{
      platforms: MessagingPlatformSchema[]
      agents: string[]
    }>(`${API_BASE}/messaging/supported`)
    if (!result.success) throw new Error('Failed to load messaging schema')
    return result.data || { platforms: [], agents: [] }
  },

  async getWorkspace(): Promise<MessagingWorkspace> {
    const result = await httpClient.get<MessagingWorkspace>(
      `${API_BASE}/messaging/workspace`
    )
    if (!result.success) throw new Error('Failed to load messaging workspace')
    return result.data!
  },

  async saveWorkspace(
    body: Partial<MessagingWorkspace>
  ): Promise<MessagingWorkspace> {
    const result = await httpClient.request<MessagingWorkspace>(
      `${API_BASE}/messaging/workspace`,
      { method: 'PUT', data: body }
    )
    if (!result.success) throw new Error('Failed to save messaging workspace')
    return result.data!
  },

  async listChannels(): Promise<MessagingChannel[]> {
    const result = await httpClient.get<{ channels: MessagingChannel[] }>(
      `${API_BASE}/messaging/channels`
    )
    if (!result.success) throw new Error('Failed to list messaging channels')
    return result.data?.channels || []
  },

  async createChannel(body: {
    name: string
    platform_type: string
    credentials: Record<string, string>
    chat_enabled?: boolean
    alerts_enabled?: boolean
    enabled?: boolean
    allow_from?: string
    bound_peer?: string
  }): Promise<MessagingChannel> {
    const result = await httpClient.request<MessagingChannel>(
      `${API_BASE}/messaging/channels`,
      { method: 'POST', data: body }
    )
    if (!result.success) throw new Error('Failed to create channel')
    return result.data!
  },

  async updateChannel(
    id: string,
    body: Partial<{
      name: string
      credentials: Record<string, string>
      chat_enabled: boolean
      alerts_enabled: boolean
      enabled: boolean
      allow_from: string
      bound_peer: string
    }>
  ): Promise<MessagingChannel> {
    const result = await httpClient.request<MessagingChannel>(
      `${API_BASE}/messaging/channels/${id}`,
      { method: 'PUT', data: body }
    )
    if (!result.success) throw new Error('Failed to update channel')
    return result.data!
  },

  async deleteChannel(id: string): Promise<void> {
    const result = await httpClient.request(`${API_BASE}/messaging/channels/${id}`, {
      method: 'DELETE',
    })
    if (!result.success) throw new Error('Failed to delete channel')
  },

  async reload(): Promise<void> {
    const result = await httpClient.request(`${API_BASE}/messaging/reload`, {
      method: 'POST',
    })
    if (!result.success) throw new Error('Failed to reload messaging')
  },
}
