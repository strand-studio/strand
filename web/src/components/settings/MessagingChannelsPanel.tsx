import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router'
import { toast } from 'sonner'
import { Plus, RefreshCw, Trash2 } from 'lucide-react'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import {
  messagingApi,
  type MessagingChannel,
  type MessagingPlatformSchema,
  type MessagingWorkspace,
} from '../../lib/api/messaging'
import { ROUTES } from '../../router/paths'

export function MessagingChannelsPanel({
  platformsOnly = false,
  embedded = false,
  onChanged,
}: {
  /** Hide conversation-agent block (managed under Resources → Agents). */
  platformsOnly?: boolean
  /** Compact heading when nested inside Resources. */
  embedded?: boolean
  /** Notify parent (e.g. Resources hub) to refresh overview after bind/unbind. */
  onChanged?: () => void
} = {}) {
  const { language } = useLanguage()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [platforms, setPlatforms] = useState<MessagingPlatformSchema[]>([])
  const [agents, setAgents] = useState<string[]>([])
  const [workspace, setWorkspace] = useState<MessagingWorkspace | null>(null)
  const [channels, setChannels] = useState<MessagingChannel[]>([])
  const [adding, setAdding] = useState(false)
  const [platformType, setPlatformType] = useState('telegram')
  const [name, setName] = useState('')
  const [creds, setCreds] = useState<Record<string, string>>({})
  const [chatEnabled, setChatEnabled] = useState(true)
  const [alertsEnabled, setAlertsEnabled] = useState(true)
  const [boundPeer, setBoundPeer] = useState('')
  const [allowFrom, setAllowFrom] = useState('')

  const schema = useMemo(
    () => platforms.find((p) => p.type === platformType) || platforms[0],
    [platforms, platformType]
  )

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const [supported, ws, list] = await Promise.all([
        messagingApi.getSupported(),
        messagingApi.getWorkspace(),
        messagingApi.listChannels(),
      ])
      setPlatforms(supported.platforms)
      setAgents(supported.agents)
      setWorkspace(ws)
      setChannels(list)
      if (supported.platforms.length && !supported.platforms.some((p) => p.type === platformType)) {
        setPlatformType(supported.platforms[0].type)
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('messagingLoadFailed', language))
    } finally {
      setLoading(false)
    }
  }, [language, platformType])

  useEffect(() => {
    void refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const saveWorkspace = async () => {
    if (!workspace) return
    setSaving(true)
    try {
      const next = await messagingApi.saveWorkspace({
        agent_type: workspace.agent_type,
        agent_model: workspace.agent_model,
        work_dir: workspace.work_dir,
        language: workspace.language,
      })
      setWorkspace(next)
      toast.success(t('messagingWorkspaceSaved', language))
      await messagingApi.reload().catch(() => {})
      await refresh()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('messagingSaveFailed', language))
    } finally {
      setSaving(false)
    }
  }

  const createChannel = async () => {
    if (!name.trim() || !schema) return
    setSaving(true)
    try {
      await messagingApi.createChannel({
        name: name.trim(),
        platform_type: schema.type,
        credentials: creds,
        chat_enabled: chatEnabled,
        alerts_enabled: alertsEnabled,
        allow_from: allowFrom,
        bound_peer: boundPeer,
      })
      toast.success(t('messagingChannelCreated', language))
      setAdding(false)
      setName('')
      setCreds({})
      setBoundPeer('')
      setAllowFrom('')
      await refresh()
      onChanged?.()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('messagingSaveFailed', language))
    } finally {
      setSaving(false)
    }
  }

  const toggleChannel = async (
    ch: MessagingChannel,
    patch: Partial<MessagingChannel>
  ) => {
    try {
      await messagingApi.updateChannel(ch.id, {
        chat_enabled: patch.chat_enabled ?? ch.chat_enabled,
        alerts_enabled: patch.alerts_enabled ?? ch.alerts_enabled,
        enabled: patch.enabled ?? ch.enabled,
      })
      await refresh()
      onChanged?.()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('messagingSaveFailed', language))
    }
  }

  const removeChannel = async (id: string) => {
    if (!window.confirm(t('messagingDeleteConfirm', language))) return
    try {
      await messagingApi.deleteChannel(id)
      toast.success(t('messagingChannelDeleted', language))
      await refresh()
      onChanged?.()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('messagingSaveFailed', language))
    }
  }

  if (loading && !workspace) {
    return (
      <p className="text-sm text-strand-text-muted">
        {t('loading', language)}
      </p>
    )
  }

  return (
    <div className={embedded ? 'space-y-4' : 'space-y-8'}>
      {!embedded && (
        <div>
          <h3 className="text-sm font-semibold text-strand-text mb-1">
            {t('messagingTitle', language)}
          </h3>
          <p className="text-xs text-strand-text-muted leading-relaxed">
            {t('messagingDesc', language)}
          </p>
        </div>
      )}

      {!platformsOnly && (
        <section className="space-y-3 border-t border-strand-border/40 pt-6">
          <div className="flex items-center justify-between gap-3">
            <h4 className="text-sm font-medium text-strand-text">
              {t('messagingWorkspace', language)}
            </h4>
            <div className="flex gap-2">
              <Link
                to={ROUTES.resources}
                className="text-xs px-3 py-1.5 rounded-lg border border-strand-border/50 text-strand-text"
              >
                {t('resourcesTab', language)}
              </Link>
              <button
                type="button"
                disabled={saving}
                onClick={() => void saveWorkspace()}
                className="text-xs px-3 py-1.5 rounded-lg bg-strand-gold text-strand-bg font-medium disabled:opacity-50"
              >
                {t('save', language)}
              </button>
            </div>
          </div>
          <p className="text-xs text-strand-text-muted">
            {t('messagingWorkspaceHint', language)}{' '}
            {t('resourcesTestInHub', language)}
          </p>
          <label className="block text-xs text-strand-text-muted">
            {t('messagingAgentType', language)}
            <select
              className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg px-3 py-2 text-sm text-strand-text"
              value={workspace?.agent_type || ''}
              onChange={(e) =>
                setWorkspace((w) =>
                  w ? { ...w, agent_type: e.target.value } : w
                )
              }
            >
              <option value="">{t('messagingSelectAgent', language)}</option>
              {agents.map((a) => (
                <option key={a} value={a}>
                  {a}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-xs text-strand-text-muted">
            {t('messagingAgentModel', language)}
            <input
              className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg px-3 py-2 text-sm text-strand-text"
              value={workspace?.agent_model || ''}
              onChange={(e) =>
                setWorkspace((w) =>
                  w ? { ...w, agent_model: e.target.value } : w
                )
              }
              placeholder="optional"
            />
          </label>
          <label className="block text-xs text-strand-text-muted">
            {t('messagingWorkDir', language)}
            <input
              className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg px-3 py-2 text-sm text-strand-text"
              value={workspace?.work_dir || ''}
              onChange={(e) =>
                setWorkspace((w) =>
                  w ? { ...w, work_dir: e.target.value } : w
                )
              }
            />
          </label>
        </section>
      )}

      <section
        className={
          embedded
            ? 'space-y-3'
            : 'space-y-3 border-t border-strand-border/40 pt-6'
        }
      >
        <div className="flex items-center justify-between gap-3">
          <h4 className="text-sm font-medium text-strand-text">
            {t('messagingBindings', language)}
          </h4>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => void refresh()}
              className="p-1.5 rounded-lg border border-strand-border/50 text-strand-text-muted hover:text-strand-text"
              title="Refresh"
            >
              <RefreshCw size={14} />
            </button>
            <button
              type="button"
              onClick={() => setAdding((v) => !v)}
              className="flex items-center gap-1 text-xs px-3 py-1.5 rounded-lg bg-strand-gold text-strand-bg font-medium"
            >
              <Plus size={14} />
              {t('messagingAdd', language)}
            </button>
          </div>
        </div>

        {adding && schema && (
          <div className="rounded-xl border border-strand-border/50 bg-strand-bg p-4 space-y-3">
            <label className="block text-xs text-strand-text-muted">
              {t('messagingPlatform', language)}
              <select
                className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text"
                value={platformType}
                onChange={(e) => {
                  setPlatformType(e.target.value)
                  setCreds({})
                }}
              >
                {platforms.map((p) => (
                  <option key={p.type} value={p.type}>
                    {p.label}
                  </option>
                ))}
              </select>
            </label>
            <p className="text-[11px] text-strand-text-muted">
              {schema.description}
            </p>
            <label className="block text-xs text-strand-text-muted">
              {t('messagingChannelName', language)}
              <input
                className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text placeholder:text-strand-text-muted"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. main-telegram"
              />
            </label>
            {schema.fields.map((f) => (
              <label key={f.key} className="block text-xs text-strand-text-muted">
                {f.label}
                {f.required ? ' *' : ''}
                <input
                  type={f.secret ? 'password' : 'text'}
                  className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text placeholder:text-strand-text-muted"
                  value={creds[f.key] || ''}
                  placeholder={f.placeholder}
                  onChange={(e) =>
                    setCreds((prev) => ({ ...prev, [f.key]: e.target.value }))
                  }
                />
                {f.help && (
                  <span className="block mt-1 text-[11px] opacity-80">
                    {f.help}
                  </span>
                )}
              </label>
            ))}
            <label className="block text-xs text-strand-text-muted">
              {t('messagingBoundPeer', language)}
              <input
                className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text placeholder:text-strand-text-muted"
                value={boundPeer}
                onChange={(e) => setBoundPeer(e.target.value)}
                placeholder={
                  schema.bound_peer_placeholder ||
                  t('messagingBoundPeerPlaceholder', language)
                }
              />
              {schema.bound_peer_help ? (
                <span className="block mt-1 text-[11px] opacity-80">
                  {schema.bound_peer_help}
                </span>
              ) : null}
            </label>
            <label className="block text-xs text-strand-text-muted">
              {t('messagingAllowFrom', language)}
              <input
                className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text placeholder:text-strand-text-muted"
                value={allowFrom}
                onChange={(e) => setAllowFrom(e.target.value)}
              />
            </label>
            <div className="flex flex-wrap gap-4 text-xs text-strand-text">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={chatEnabled}
                  onChange={(e) => setChatEnabled(e.target.checked)}
                />
                {t('messagingChat', language)}
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={alertsEnabled}
                  onChange={(e) => setAlertsEnabled(e.target.checked)}
                />
                {t('messagingAlerts', language)}
              </label>
            </div>
            <div className="flex gap-2 justify-end">
              <button
                type="button"
                className="text-xs px-3 py-1.5 rounded-lg border border-strand-border/50"
                onClick={() => setAdding(false)}
              >
                {t('cancel', language)}
              </button>
              <button
                type="button"
                disabled={saving || !name.trim()}
                className="text-xs px-3 py-1.5 rounded-lg bg-strand-gold text-strand-bg font-medium disabled:opacity-50"
                onClick={() => void createChannel()}
              >
                {t('messagingCreate', language)}
              </button>
            </div>
          </div>
        )}

        {channels.length === 0 ? (
          <p className="text-xs text-strand-text-muted">
            {t('messagingEmpty', language)}
          </p>
        ) : (
          <ul className="space-y-2">
            {channels.map((ch) => (
              <li
                key={ch.id}
                className="rounded-xl border border-strand-border/40 px-3 py-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"
              >
                <div>
                  <div className="text-sm font-medium text-strand-text">
                    {ch.name}{' '}
                    <span className="text-strand-text-muted font-normal">
                      · {ch.platform_type}
                    </span>
                  </div>
                  <div className="text-[11px] text-strand-text-muted mt-0.5">
                    {ch.status}
                    {ch.last_error ? ` · ${ch.last_error}` : ''}
                    {ch.bound_peer ? ` · peer ${ch.bound_peer}` : ''}
                  </div>
                </div>
                <div className="flex flex-wrap items-center gap-3 text-xs">
                  <label className="flex items-center gap-1.5">
                    <input
                      type="checkbox"
                      checked={ch.enabled}
                      onChange={(e) =>
                        void toggleChannel(ch, { enabled: e.target.checked })
                      }
                    />
                    {t('enabled', language)}
                  </label>
                  <label className="flex items-center gap-1.5">
                    <input
                      type="checkbox"
                      checked={ch.chat_enabled}
                      onChange={(e) =>
                        void toggleChannel(ch, {
                          chat_enabled: e.target.checked,
                        })
                      }
                    />
                    {t('messagingChat', language)}
                  </label>
                  <label className="flex items-center gap-1.5">
                    <input
                      type="checkbox"
                      checked={ch.alerts_enabled}
                      onChange={(e) =>
                        void toggleChannel(ch, {
                          alerts_enabled: e.target.checked,
                        })
                      }
                    />
                    {t('messagingAlerts', language)}
                  </label>
                  <button
                    type="button"
                    className="p-1.5 text-strand-text-muted hover:text-red-500"
                    onClick={() => void removeChannel(ch.id)}
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}
