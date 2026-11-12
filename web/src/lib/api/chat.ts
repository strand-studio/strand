import { getAuthHeaders, handleJSONResponse } from './helpers'

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
}

export interface ChatToolDef {
  name: string
  description: string
  parameters?: Record<string, unknown>
}

export interface ChatEvent {
  type: 'start' | 'delta' | 'tool_start' | 'tool_result' | 'error' | 'done' | string
  data: Record<string, unknown>
}

export interface ChatCompletionResponse {
  content: string
  model?: { id: string; name: string; provider: string }
  tool_trace?: Array<{ name: string; arguments?: string; result?: unknown }>
}

export async function listChatTools(): Promise<ChatToolDef[]> {
  const r = await fetch('/api/chat/tools', { headers: getAuthHeaders() })
  const data = await handleJSONResponse<{ tools: ChatToolDef[] }>(r)
  return data.tools || []
}

export async function testChatTool(
  name: string,
  args: Record<string, unknown> = {}
): Promise<{ name: string; result: unknown }> {
  const r = await fetch('/api/chat/tools/test', {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ name, args }),
  })
  return handleJSONResponse(r)
}

export async function chatCompletions(body: {
  messages: ChatMessage[]
  model_id?: string
  tools_enabled?: boolean
  trader_id?: string
}): Promise<ChatCompletionResponse> {
  const r = await fetch('/api/chat/completions', {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(body),
  })
  return handleJSONResponse(r)
}

export async function streamChat(
  body: {
    messages: ChatMessage[]
    model_id?: string
    tools_enabled?: boolean
    trader_id?: string
    session_id?: string
  },
  signal: AbortSignal,
  onEvent: (event: ChatEvent) => void
): Promise<void> {
  const r = await fetch('/api/chat/stream', {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(body),
    signal,
  })
  if (!r.ok) {
    await handleJSONResponse(r)
    return
  }
  if (!r.body) throw new Error('Server returned no chat stream')
  const reader = r.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let finished = false
  let serverError = false
  try {
    for (;;) {
      const { done, value } = await reader.read()
      buffer += decoder.decode(value, { stream: !done })
      buffer = buffer.replace(/\r\n/g, '\n')
      let at: number
      while ((at = buffer.indexOf('\n\n')) >= 0) {
        const frame = buffer.slice(0, at)
        buffer = buffer.slice(at + 2)
        for (const line of frame.split('\n')) {
          if (!line.startsWith('data: ')) continue
          const event = JSON.parse(line.slice(6)) as ChatEvent
          if (event.type === 'done') finished = true
          if (event.type === 'error') serverError = true
          onEvent(event)
        }
      }
      if (done) {
        if (!finished && !serverError) {
          throw new Error('Chat stream ended without a result')
        }
        break
      }
    }
  } finally {
    await reader.cancel().catch(() => {})
  }
}
