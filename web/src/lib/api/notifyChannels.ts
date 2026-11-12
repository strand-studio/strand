import { API_BASE, getAuthHeaders, handleJSONResponse } from './helpers'

export type NotifyChannelType = 'discord' | 'slack'

export interface NotifyChannelConfig {
  channel_type: NotifyChannelType
  configured: boolean
  token_masked?: string
  webhook_masked?: string
  is_bound?: boolean
  bound_guild_id?: string
  bound_channel_id?: string
  username?: string
  model_id?: string
}

export async function getNotifyChannel(
  type: NotifyChannelType
): Promise<NotifyChannelConfig> {
  const r = await fetch(`${API_BASE}/notify/${type}`, {
    headers: getAuthHeaders(),
  })
  return handleJSONResponse(r)
}

export async function saveNotifyChannel(
  type: NotifyChannelType,
  body: {
    bot_token?: string
    webhook_url?: string
    model_id?: string
    guild_id?: string
    channel_id?: string
    username?: string
  }
): Promise<{ ok: boolean; hint?: string }> {
  const r = await fetch(`${API_BASE}/notify/${type}`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(body),
  })
  return handleJSONResponse(r)
}

export async function unbindNotifyChannel(
  type: NotifyChannelType
): Promise<void> {
  const r = await fetch(`${API_BASE}/notify/${type}/binding`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  await handleJSONResponse(r)
}

export async function clearNotifyChannel(
  type: NotifyChannelType
): Promise<void> {
  const r = await fetch(`${API_BASE}/notify/${type}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  await handleJSONResponse(r)
}
