import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router'
import useSWR from 'swr'
import { api } from '../lib/api'
import {
  listChatTools,
  streamChat,
  testChatTool,
  type ChatMessage,
  type ChatToolDef,
} from '../lib/api/chat'
import type { ChatSessionRow } from '../lib/api/chatSessions'
import { ChatKlineSpark } from '../components/charts/ChatKlineSpark'
import { useLanguage } from '../contexts/LanguageContext'
import { t } from '../i18n/translations'
import { ROUTES } from '../router/paths'
import { pickPreferredModel } from '../lib/preferredModel'
import './AIChatPage.css'

type Bubble =
  | { kind: 'message'; role: 'user' | 'assistant'; content: string }
  | { kind: 'tool'; label: string }
  | { kind: 'error'; content: string }
  | {
      kind: 'chart'
      symbol: string
      bars: Array<{ openTime?: number; time?: number; close?: number | string }>
    }

const LAST_SESSION_KEY = 'strand.chat.last_session'

export function AIChatPage() {
  const { language } = useLanguage()
  const prompts = [
    t('aiChatPrompt1', language),
    t('aiChatPrompt2', language),
    t('aiChatPrompt3', language),
    t('aiChatPrompt4', language),
  ]
  const [bubbles, setBubbles] = useState<Bubble[]>([])
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const [toolsEnabled, setToolsEnabled] = useState(true)
  const [modelId, setModelId] = useState('')
  const [tools, setTools] = useState<ChatToolDef[]>([])
  const [testResult, setTestResult] = useState('')
  const [showTest, setShowTest] = useState(false)
  const [tgToken, setTgToken] = useState('')
  const [tgBusy, setTgBusy] = useState(false)
  const [tgMsg, setTgMsg] = useState('')
  const [sessionId, setSessionId] = useState(
    () => localStorage.getItem(LAST_SESSION_KEY) || ''
  )
  const [sessions, setSessions] = useState<ChatSessionRow[]>([])
  const [discordToken, setDiscordToken] = useState('')
  const [slackWebhook, setSlackWebhook] = useState('')
  const [slackChannel, setSlackChannel] = useState('')
  const [notifyBusy, setNotifyBusy] = useState(false)
  const [notifyMsg, setNotifyMsg] = useState('')
  const [sessionQuery, setSessionQuery] = useState('')
  const [renamingId, setRenamingId] = useState('')
  const [renameDraft, setRenameDraft] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  const { data: models, mutate: mutateModels } = useSWR(
    'ai-chat-models',
    () => api.getModelConfigs()
  )
  const { data: telegram, mutate: mutateTelegram } = useSWR(
    'ai-chat-telegram',
    () => api.getTelegramConfig(),
    {
      // Poll while waiting for the user to send /start in Telegram.
      refreshInterval: (latest) =>
        latest && latest.token_masked && !latest.is_bound ? 4000 : 30000,
      revalidateOnFocus: true,
    }
  )
  const { data: discord, mutate: mutateDiscord } = useSWR(
    'ai-chat-discord',
    () => api.getNotifyChannel('discord')
  )
  const { data: slack, mutate: mutateSlack } = useSWR('ai-chat-slack', () =>
    api.getNotifyChannel('slack')
  )

  async function refreshSessions(q = sessionQuery) {
    try {
      const rows = await api.listChatSessions(q)
      setSessions(rows)
    } catch {
      setSessions([])
    }
  }

  useEffect(() => {
    void refreshSessions()
  }, [])

  useEffect(() => {
    const handle = window.setTimeout(() => {
      void refreshSessions(sessionQuery)
    }, 250)
    return () => window.clearTimeout(handle)
  }, [sessionQuery])

  useEffect(() => {
    if (!sessionId) return
    localStorage.setItem(LAST_SESSION_KEY, sessionId)
    void (async () => {
      try {
        const data = await api.getChatSession(sessionId)
        const restored: Bubble[] = []
        for (const m of data.messages || []) {
          if (m.role === 'tool') {
            restored.push({
              kind: 'tool',
              label: `tool · ${m.tool_name || 'tool'}`,
            })
            continue
          }
          restored.push({
            kind: 'message',
            role: m.role === 'assistant' ? 'assistant' : 'user',
            content: m.content,
          })
        }
        setBubbles(restored)
        if (data.session.model_id) setModelId(data.session.model_id)
      } catch {
        /* stale id */
      }
    })()
  }, [sessionId])

  useEffect(() => {
    if (telegram?.is_bound) {
      setTgMsg((prev) =>
        prev && /\/start|待 \/start|waiting \/start|menunggu \/start/i.test(prev)
          ? t('telegram.boundAs', language, {
              name:
                telegram.username || String(telegram.bound_chat_id || ''),
            })
          : prev
      )
    }
  }, [telegram?.is_bound, telegram?.username, telegram?.bound_chat_id, language])
  const enabledModels = useMemo(
    () => (models || []).filter((m) => m.enabled && (m.has_api_key || m.apiKey)),
    [models]
  )
  const hasModel = enabledModels.length > 0

  useEffect(() => {
    if (modelId) return
    const preferred = pickPreferredModel(enabledModels)
    if (preferred?.id) setModelId(preferred.id)
  }, [enabledModels, modelId])

  useEffect(() => {
    listChatTools()
      .then(setTools)
      .catch(() => setTools([]))
  }, [])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [bubbles, sending])

  const history = useMemo(() => {
    const msgs: ChatMessage[] = []
    for (const b of bubbles) {
      if (b.kind === 'message') msgs.push({ role: b.role, content: b.content })
    }
    return msgs
  }, [bubbles])

  async function send(text: string) {
    const content = text.trim()
    if (!content || sending) return
    if (!hasModel) {
      setBubbles((prev) => [
        ...prev,
        {
          kind: 'error',
          content: t('aiChatNoModelError', language),
        },
      ])
      return
    }
    setInput('')
    setBubbles((prev) => [...prev, { kind: 'message', role: 'user', content }])
    setSending(true)
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller

    let assistant = ''
    setBubbles((prev) => [
      ...prev,
      { kind: 'message', role: 'assistant', content: '' },
    ])

    try {
      let activeSession = sessionId
      if (!activeSession) {
        try {
          const created = await api.createChatSession({
            model_id: modelId || undefined,
            title: content.slice(0, 40),
          })
          activeSession = created.id
          setSessionId(created.id)
          await refreshSessions()
        } catch {
          /* offline persistence optional */
        }
      }
      await streamChat(
        {
          messages: [...history, { role: 'user', content }],
          model_id: modelId || undefined,
          tools_enabled: toolsEnabled,
          session_id: activeSession || undefined,
        },
        controller.signal,
        (event) => {
          if (event.type === 'tool_start') {
            const name = String(event.data.name || 'tool')
            setBubbles((prev) => [
              ...prev.slice(0, -1),
              { kind: 'tool', label: `tool · ${name}` },
              { kind: 'message', role: 'assistant', content: assistant },
            ])
            return
          }
          if (event.type === 'tool_result') {
            const name = String(event.data.name || '')
            if (name === 'get_klines') {
              const raw = event.data.result
              const payload =
                typeof raw === 'string'
                  ? (() => {
                      try {
                        return JSON.parse(raw)
                      } catch {
                        return null
                      }
                    })()
                  : raw
              const obj = payload as {
                candles?: unknown[]
                klines?: unknown[]
                symbol?: string
              } | null
              const list = Array.isArray(payload)
                ? payload
                : Array.isArray(obj?.candles)
                  ? obj!.candles!
                  : Array.isArray(obj?.klines)
                    ? obj!.klines!
                    : []
              const symbol = String(obj?.symbol || 'Chart')
              if (list.length >= 2) {
                setBubbles((prev) => [
                  ...prev,
                  {
                    kind: 'chart',
                    symbol,
                    bars: list as Array<{
                      openTime?: number
                      time?: number
                      close?: number | string
                    }>,
                  },
                ])
              }
            }
            return
          }
          if (event.type === 'delta') {
            assistant += String(event.data.content || '')
            setBubbles((prev) => {
              const next = [...prev]
              const last = next[next.length - 1]
              if (last?.kind === 'message' && last.role === 'assistant') {
                next[next.length - 1] = {
                  kind: 'message',
                  role: 'assistant',
                  content: assistant,
                }
              } else {
                next.push({
                  kind: 'message',
                  role: 'assistant',
                  content: assistant,
                })
              }
              return next
            })
            return
          }
          if (event.type === 'error') {
            setBubbles((prev) => [
              ...prev,
              {
                kind: 'error',
                content: String(
                  event.data.message || t('aiChatFailed', language)
                ),
              },
            ])
          }
        }
      )
      if (!assistant.trim()) {
        setBubbles((prev) => {
          const next = [...prev]
          const last = next[next.length - 1]
          if (
            last?.kind === 'message' &&
            last.role === 'assistant' &&
            !last.content
          ) {
            next[next.length - 1] = {
              kind: 'message',
              role: 'assistant',
              content: t('aiChatEmptyModelReply', language),
            }
          }
          return next
        })
      }
    } catch (err) {
      if ((err as Error).name === 'AbortError') return
      setBubbles((prev) => [
        ...prev,
        {
          kind: 'error',
          content: err instanceof Error ? err.message : String(err),
        },
      ])
    } finally {
      setSending(false)
    }
  }

  async function runToolTest(name: string) {
    setShowTest(true)
    setTestResult(t('aiChatTesting', language))
    try {
      const defaults: Record<string, Record<string, unknown>> = {
        list_supported_chains: {},
        list_markets: { exchange: 'hyperliquid', limit: 10 },
        get_klines: {
          symbol: 'BTC',
          exchange: 'hyperliquid',
          interval: '1h',
          limit: 12,
        },
        get_direction_leaderboard: { limit: 8 },
        market_brief: { symbol: 'BTC', exchange: 'hyperliquid' },
        get_account_snapshot: {},
      }
      const result = await testChatTool(name, defaults[name] || {})
      setTestResult(JSON.stringify(result, null, 2))
    } catch (err) {
      setTestResult(err instanceof Error ? err.message : String(err))
    }
  }

  async function saveTelegram() {
    setTgBusy(true)
    setTgMsg('')
    try {
      await api.updateTelegramConfig(tgToken.trim(), modelId || undefined)
      setTgToken('')
      setTgMsg(t('telegram.tokenSavedOpenStart', language))
      await mutateTelegram()
    } catch (err) {
      setTgMsg(err instanceof Error ? err.message : String(err))
    } finally {
      setTgBusy(false)
    }
  }

  return (
    <div className="ai-chat-page">
      <div className="ai-chat-shell">
        <aside className="ai-chat-side">
          <div className="ai-chat-intro">
            <h1 className="ai-chat-brand">{t('aiChatTitle', language)}</h1>
            <p>{t('aiChatIntro', language)}</p>
          </div>

          <section className="ai-chat-section">
            <div className="ai-chat-section-label">
              {t('telegram.channels', language)}
            </div>
            <div className="ai-chat-channels">
              <div className="ai-chat-channel is-ready">
                <strong>{t('telegram.webChat', language)}</strong>
                <span>{t('telegram.webChatReady', language)}</span>
              </div>
              <div
                className={
                  'ai-chat-channel ' +
                  (telegram?.is_bound
                    ? 'is-ready'
                    : telegram?.token_masked
                      ? 'is-pending'
                      : 'is-todo')
                }
              >
                <strong>Telegram</strong>
                <span>
                  {telegram?.is_bound
                    ? t('telegram.boundAs', language, {
                        name:
                          telegram.username ||
                          String(telegram.bound_chat_id || ''),
                      })
                    : telegram?.token_masked
                      ? t('telegram.tokenSavedWaiting', language, {
                          masked: telegram.token_masked,
                        })
                      : t('telegram.notBoundHint', language)}
                </span>
                {!telegram?.is_bound && (
                  <div className="ai-chat-tg-bind">
                    <ol>
                      <li>{t('telegram.botFatherStep', language)}</li>
                      <li>{t('telegram.pasteTokenStep', language)}</li>
                      <li>{t('telegram.sendStartStep', language)}</li>
                    </ol>
                    <input
                      value={tgToken}
                      onChange={(e) => setTgToken(e.target.value)}
                      placeholder="123456:ABC..."
                      aria-label="Telegram bot token"
                    />
                    <button
                      type="button"
                      disabled={tgBusy || !tgToken.trim()}
                      onClick={() => void saveTelegram()}
                    >
                      {tgBusy
                        ? t('telegram.savingToken', language)
                        : t('telegram.saveToken', language)}
                    </button>
                    {tgMsg && <small>{tgMsg}</small>}
                  </div>
                )}
                {telegram?.is_bound && (
                  <div className="ai-chat-tg-bind">
                    <button
                      type="button"
                      disabled={tgBusy}
                      onClick={() => {
                        void (async () => {
                          setTgBusy(true)
                          try {
                            await api.unbindTelegram()
                            setTgMsg(t('telegram.unboundResendStart', language))
                            await mutateTelegram()
                          } catch (err) {
                            setTgMsg(
                              err instanceof Error ? err.message : String(err)
                            )
                          } finally {
                            setTgBusy(false)
                          }
                        })()
                      }}
                    >
                      {t('telegram.unbindAccount', language)}
                    </button>
                    {tgMsg && <small>{tgMsg}</small>}
                  </div>
                )}
              </div>
              <div
                className={
                  'ai-chat-channel ' +
                  (discord?.is_bound
                    ? 'is-ready'
                    : discord?.configured
                      ? 'is-pending'
                      : 'is-todo')
                }
              >
                <strong>Discord</strong>
                <span>
                  {discord?.is_bound
                    ? t('aiChatDiscordBound', language, {
                        id:
                          discord.bound_channel_id ||
                          discord.username ||
                          '',
                      })
                    : discord?.configured
                      ? t('aiChatDiscordPending', language, {
                          masked: discord.token_masked || '',
                        })
                      : t('aiChatDiscordHint', language)}
                </span>
                <div className="ai-chat-tg-bind">
                  <input
                    value={discordToken}
                    onChange={(e) => setDiscordToken(e.target.value)}
                    placeholder={t('aiChatDiscordTokenPlaceholder', language)}
                    aria-label={t('aiChatDiscordTokenPlaceholder', language)}
                  />
                  <button
                    type="button"
                    disabled={notifyBusy || !discordToken.trim()}
                    onClick={() => {
                      void (async () => {
                        setNotifyBusy(true)
                        setNotifyMsg('')
                        try {
                          const res = await api.saveNotifyChannel('discord', {
                            bot_token: discordToken.trim(),
                            model_id: modelId || undefined,
                          })
                          setDiscordToken('')
                          setNotifyMsg(
                            res.hint || t('aiChatDiscordSaved', language)
                          )
                          await mutateDiscord()
                        } catch (err) {
                          setNotifyMsg(
                            err instanceof Error ? err.message : String(err)
                          )
                        } finally {
                          setNotifyBusy(false)
                        }
                      })()
                    }}
                  >
                    {t('aiChatDiscordSave', language)}
                  </button>
                </div>
              </div>
              <div
                className={
                  'ai-chat-channel ' +
                  (slack?.is_bound
                    ? 'is-ready'
                    : slack?.configured
                      ? 'is-pending'
                      : 'is-todo')
                }
              >
                <strong>Slack</strong>
                <span>
                  {slack?.is_bound
                    ? t('aiChatSlackBound', language, {
                        id: slack.bound_channel_id || '',
                      })
                    : slack?.configured
                      ? t('aiChatSlackPending', language, {
                          masked: slack.webhook_masked || '',
                        })
                      : t('aiChatSlackHint', language)}
                </span>
                <div className="ai-chat-tg-bind">
                  <input
                    value={slackWebhook}
                    onChange={(e) => setSlackWebhook(e.target.value)}
                    placeholder={t('aiChatSlackWebhookPlaceholder', language)}
                    aria-label={t('aiChatSlackWebhookPlaceholder', language)}
                  />
                  <input
                    value={slackChannel}
                    onChange={(e) => setSlackChannel(e.target.value)}
                    placeholder={t('aiChatSlackChannelPlaceholder', language)}
                    aria-label={t('aiChatSlackChannelPlaceholder', language)}
                  />
                  <button
                    type="button"
                    disabled={notifyBusy || !slackWebhook.trim()}
                    onClick={() => {
                      void (async () => {
                        setNotifyBusy(true)
                        setNotifyMsg('')
                        try {
                          const res = await api.saveNotifyChannel('slack', {
                            webhook_url: slackWebhook.trim(),
                            channel_id: slackChannel.trim() || undefined,
                            model_id: modelId || undefined,
                          })
                          setSlackWebhook('')
                          setNotifyMsg(
                            res.hint || t('aiChatSlackSaved', language)
                          )
                          await mutateSlack()
                        } catch (err) {
                          setNotifyMsg(
                            err instanceof Error ? err.message : String(err)
                          )
                        } finally {
                          setNotifyBusy(false)
                        }
                      })()
                    }}
                  >
                    {t('aiChatSlackSave', language)}
                  </button>
                  {notifyMsg && <small>{notifyMsg}</small>}
                </div>
              </div>
            </div>
          </section>

          <section className="ai-chat-section">
            <div className="ai-chat-section-label">
              {t('aiChatSessions', language)}
            </div>
            <div className="ai-chat-sessions">
              <input
                className="ai-chat-session-search"
                value={sessionQuery}
                onChange={(e) => setSessionQuery(e.target.value)}
                placeholder={t('aiChatSearchPlaceholder', language)}
                aria-label={t('aiChatSearchAria', language)}
              />
              <button
                type="button"
                onClick={() => {
                  void (async () => {
                    const created = await api.createChatSession({
                      model_id: modelId || undefined,
                    })
                    setSessionId(created.id)
                    setBubbles([])
                    await refreshSessions()
                  })()
                }}
              >
                {t('aiChatNewSession', language)}
              </button>
              {sessions.map((s) => (
                <div
                  key={s.id}
                  className={
                    s.id === sessionId
                      ? 'ai-chat-session is-active'
                      : 'ai-chat-session'
                  }
                >
                  {renamingId === s.id ? (
                    <form
                      className="ai-chat-session-rename"
                      onSubmit={(e) => {
                        e.preventDefault()
                        void (async () => {
                          const title = renameDraft.trim()
                          if (!title) return
                          await api.renameChatSession(s.id, title)
                          setRenamingId('')
                          await refreshSessions()
                        })()
                      }}
                    >
                      <input
                        value={renameDraft}
                        onChange={(e) => setRenameDraft(e.target.value)}
                        autoFocus
                      />
                      <button type="submit">
                        {t('save', language)}
                      </button>
                    </form>
                  ) : (
                    <button
                      type="button"
                      className="ai-chat-session-main"
                      onClick={() => setSessionId(s.id)}
                      onDoubleClick={() => {
                        setRenamingId(s.id)
                        setRenameDraft(s.title || '')
                      }}
                      title={t('aiChatRenameHint', language)}
                    >
                      <span>
                        {s.title || t('aiChatDefaultTitle', language)}
                      </span>
                    </button>
                  )}
                  <small
                    onClick={(e) => {
                      e.stopPropagation()
                      void (async () => {
                        await api.deleteChatSession(s.id)
                        if (sessionId === s.id) {
                          setSessionId('')
                          setBubbles([])
                          localStorage.removeItem(LAST_SESSION_KEY)
                        }
                        await refreshSessions()
                      })()
                    }}
                  >
                    ×
                  </small>
                </div>
              ))}
            </div>
          </section>

          <section className="ai-chat-section">
            <div className="ai-chat-section-label">
              {t('aiChatModelLabel', language)}
            </div>
            {!hasModel ? (
              <div className="ai-chat-bind-card">
                <p>{t('aiChatModelNeeded', language)}</p>
                <ol>
                  <li>
                    {t('aiChatOpen', language)}{' '}
                    <Link to={`${ROUTES.settings}?tab=models`}>
                      {t('aiChatOpenConfig', language)}
                    </Link>
                  </li>
                  <li>{t('aiChatAddClaw402', language)}</li>
                  <li>{t('aiChatReturnRefresh', language)}</li>
                </ol>
                <button type="button" onClick={() => void mutateModels()}>
                  {t('aiChatRefreshModels', language)}
                </button>
              </div>
            ) : (
              <p className="ai-chat-model-ok">
                {t('aiChatReady', language)} {enabledModels.length}{' '}
                {t('aiChatModelsCount', language)}
              </p>
            )}
          </section>

          <section className="ai-chat-section">
            <div className="ai-chat-section-label">
              {t('aiChatSuggested', language)}
            </div>
            <div className="ai-chat-prompts">
              {prompts.map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => send(p)}
                  disabled={sending}
                >
                  {p}
                </button>
              ))}
            </div>
          </section>

          <section className="ai-chat-section">
            <details className="ai-chat-tools-fold">
              <summary>{t('aiChatToolTests', language)}</summary>
              <div className="ai-chat-tools">
                {tools.map((tool) => (
                  <button
                    key={tool.name}
                    type="button"
                    className="ai-chat-tool-btn"
                    onClick={() => runToolTest(tool.name)}
                  >
                    {tool.name}
                    <small>{tool.description}</small>
                  </button>
                ))}
              </div>
            </details>
          </section>
        </aside>

        <section className="ai-chat-main">
          <div className="ai-chat-toolbar">
            <select
              value={modelId}
              onChange={(e) => setModelId(e.target.value)}
              aria-label="AI model"
            >
              {enabledModels.length === 0 ? (
                <option value="">{t('aiChatEnableModelFirst', language)}</option>
              ) : (
                enabledModels.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name || m.provider} · {m.provider}
                  </option>
                ))
              )}
            </select>
            <div className="ai-chat-toolbar-meta">
              <label className="ai-chat-toggle">
                <input
                  type="checkbox"
                  checked={toolsEnabled}
                  onChange={(e) => setToolsEnabled(e.target.checked)}
                />
                {t('aiChatTools', language)}
              </label>
              <button
                type="button"
                className="ai-chat-tool-btn"
                onClick={() => {
                  setBubbles([])
                  setTestResult('')
                }}
              >
                {t('aiChatClear', language)}
              </button>
            </div>
          </div>

          <div className="ai-chat-messages">
            {bubbles.length === 0 ? (
              <div className="ai-chat-empty">
                <h2>{t('chatEmptyTitle', language)}</h2>
                <p>{t('chatEmptyDesc', language)}</p>
              </div>
            ) : (
              bubbles.map((b, i) => {
                if (b.kind === 'tool') {
                  return (
                    <div key={`t-${i}`} className="ai-chat-tool-chip">
                      {b.label}
                    </div>
                  )
                }
                if (b.kind === 'chart') {
                  return (
                    <div key={`c-${i}`} className="ai-chat-bubble assistant">
                      <ChatKlineSpark symbol={b.symbol} bars={b.bars} />
                    </div>
                  )
                }
                if (b.kind === 'error') {
                  return (
                    <div key={`e-${i}`} className="ai-chat-bubble error">
                      {b.content}
                    </div>
                  )
                }
                return (
                  <div key={`m-${i}`} className={`ai-chat-bubble ${b.role}`}>
                    {b.content ||
                      (sending && b.role === 'assistant' ? '…' : '')}
                  </div>
                )
              })
            )}
            <div ref={bottomRef} />
          </div>

          <form
            className="ai-chat-composer"
            onSubmit={(e) => {
              e.preventDefault()
              void send(input)
            }}
          >
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={t('aiChatPlaceholder', language)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  void send(input)
                }
              }}
              disabled={sending}
            />
            <button type="submit" disabled={sending || !input.trim()}>
              {sending
                ? t('aiChatThinking', language)
                : t('aiChatSend', language)}
            </button>
          </form>

          {showTest && (
            <div className="ai-chat-test-panel">
              <div className="ai-chat-section-label">
                {t('aiChatLatestToolResult', language)}
              </div>
              <pre>{testResult}</pre>
            </div>
          )}
        </section>
      </div>
    </div>
  )
}
