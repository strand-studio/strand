import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { messagingApi, type MessagingWorkspace } from '../../lib/api/messaging'
import { resourcesApi } from '../../lib/api/resources'

/**
 * Agents are local CLI binaries (codex / claudecode / …) on PATH.
 * Config is: pick which agent the messaging workspace uses, optional model & workdir.
 * No API-key form like models/exchanges.
 */
export function AgentConfigPanel({
  agents,
  initialAgentId,
  onSaved,
  onBack,
}: {
  agents: Array<{ id: string; name: string; configured?: boolean }>
  initialAgentId?: string | null
  onSaved: () => void | Promise<void>
  onBack: () => void
}) {
  const { language } = useLanguage()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [workspace, setWorkspace] = useState<MessagingWorkspace | null>(null)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      setLoading(true)
      try {
        const ws = await messagingApi.getWorkspace()
        if (cancelled) return
        // Prefer the agent the user clicked in the list over whatever is
        // currently saved as the workspace default.
        setWorkspace({
          ...ws,
          agent_type: (initialAgentId || ws.agent_type || '').trim(),
        })
      } catch (e) {
        toast.error(
          e instanceof Error ? e.message : t('messagingLoadFailed', language)
        )
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [initialAgentId, language])

  const save = async () => {
    if (!workspace?.agent_type?.trim()) {
      toast.error(t('messagingSelectAgent', language))
      return
    }
    setSaving(true)
    try {
      // Refuse to mark an agent as configured unless it actually passes probe
      // (binary + auth/readiness). Otherwise TG/chat looks "OK" but replies
      // with Not logged in / Please run /login.
      const probe = await resourcesApi.test('agent', workspace.agent_type)
      if (!probe.ok) {
        toast.error(
          probe.error || t('resourcesAgentNotReady', language)
        )
        return
      }

      const next = await messagingApi.saveWorkspace({
        agent_type: workspace.agent_type,
        agent_model: workspace.agent_model,
        work_dir: workspace.work_dir,
        language: workspace.language,
      })
      setWorkspace(next)
      await resourcesApi.saveDefaults({ default_agent_id: next.agent_type })
      await messagingApi.reload().catch(() => {})
      toast.success(t('messagingWorkspaceSaved', language))
      await onSaved()
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : t('messagingSaveFailed', language)
      )
    } finally {
      setSaving(false)
    }
  }

  const test = async () => {
    const id = workspace?.agent_type?.trim()
    if (!id) {
      toast.error(t('messagingSelectAgent', language))
      return
    }
    setTesting(true)
    try {
      const result = await resourcesApi.test('agent', id)
      if (result.ok) {
        toast.success(
          t('resourcesTestOk', language, { ms: String(result.latency_ms ?? 0) })
        )
      } else {
        toast.error(result.error || t('resourcesTestFailed', language))
      }
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : t('resourcesTestFailed', language)
      )
    } finally {
      setTesting(false)
    }
  }

  if (loading || !workspace) {
    return (
      <p className="text-sm text-strand-text-muted">{t('loading', language)}</p>
    )
  }

  return (
    <div className="space-y-5">
      <div>
        <button
          type="button"
          onClick={onBack}
          className="text-xs text-strand-text-muted hover:text-strand-text mb-3"
        >
          ← {t('resourcesTab', language)}
        </button>
        <h3 className="text-base font-semibold text-strand-text">
          {t('resourcesAgentConfigTitle', language)}
        </h3>
        <p className="text-xs text-strand-text-muted mt-1 leading-relaxed">
          {t('resourcesAgentConfigDesc', language)}
        </p>
      </div>

      <label className="block text-xs text-strand-text-muted">
        {t('messagingAgentType', language)}
        <select
          className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text"
          value={workspace.agent_type || ''}
          onChange={(e) =>
            setWorkspace((w) =>
              w ? { ...w, agent_type: e.target.value } : w
            )
          }
        >
          <option value="">{t('messagingSelectAgent', language)}</option>
          {agents.map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </select>
      </label>

      <label className="block text-xs text-strand-text-muted">
        {t('messagingAgentModel', language)}
        <input
          className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text"
          value={workspace.agent_model || ''}
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
          className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text"
          value={workspace.work_dir || ''}
          onChange={(e) =>
            setWorkspace((w) => (w ? { ...w, work_dir: e.target.value } : w))
          }
        />
      </label>

      <div className="flex flex-wrap gap-2 pt-2">
        <button
          type="button"
          onClick={onBack}
          className="text-xs px-3 py-2 rounded-lg border border-strand-border/50 text-strand-text"
        >
          {t('cancel', language)}
        </button>
        <button
          type="button"
          disabled={testing || !workspace.agent_type}
          onClick={() => void test()}
          className="text-xs px-3 py-2 rounded-lg border border-strand-border/50 text-strand-text disabled:opacity-50"
        >
          {testing ? t('resourcesTesting', language) : t('resourcesTest', language)}
        </button>
        <button
          type="button"
          disabled={saving}
          onClick={() => void save()}
          className="text-xs px-3 py-2 rounded-lg bg-strand-gold text-strand-bg font-medium disabled:opacity-50"
        >
          {t('save', language)}
        </button>
      </div>
    </div>
  )
}
