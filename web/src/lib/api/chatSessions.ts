import { API_BASE, getAuthHeaders, handleJSONResponse } from './helpers'

export interface ChatSessionRow {
  id: string
  title: string
  model_id?: string
  created_at?: string
  updated_at?: string
}

export interface ChatSessionMessage {
  id: string
  role: string
  content: string
  tool_name?: string
  tool_call_id?: string
  created_at?: string
}

export async function listChatSessions(q?: string): Promise<ChatSessionRow[]> {
  const params = new URLSearchParams()
  if (q?.trim()) params.set('q', q.trim())
  const suffix = params.toString() ? `?${params}` : ''
  const r = await fetch(`${API_BASE}/chat/sessions${suffix}`, {
    headers: getAuthHeaders(),
  })
  const data = await handleJSONResponse<{ sessions: ChatSessionRow[] }>(r)
  return data.sessions || []
}

export async function createChatSession(input?: {
  title?: string
  model_id?: string
}): Promise<ChatSessionRow> {
  const r = await fetch(`${API_BASE}/chat/sessions`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(input || {}),
  })
  return handleJSONResponse(r)
}

export async function getChatSession(id: string): Promise<{
  session: ChatSessionRow
  messages: ChatSessionMessage[]
}> {
  const r = await fetch(`${API_BASE}/chat/sessions/${id}`, {
    headers: getAuthHeaders(),
  })
  return handleJSONResponse(r)
}

export async function renameChatSession(id: string, title: string): Promise<void> {
  const r = await fetch(`${API_BASE}/chat/sessions/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify({ title }),
  })
  await handleJSONResponse(r)
}

export async function deleteChatSession(id: string): Promise<void> {
  const r = await fetch(`${API_BASE}/chat/sessions/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  await handleJSONResponse(r)
}
