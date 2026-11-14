import { useCallback, useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router'
import { toast } from 'sonner'
import {
  Activity,
  Building2,
  ChevronLeft,
  Cpu,
  MessageSquare,
  Pencil,
  Plus,
  RefreshCw,
  Terminal,
  Wallet,
} from 'lucide-react'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { api } from '../../lib/api'
import {
  resourcesApi,
  type ResourceDefaults,
  type ResourcesOverview,
  type ResourcesUsage,
} from '../../lib/api/resources'
import {
  getPreferredModelId,
  setPreferredModelId,
} from '../../lib/preferredModel'
import type { AIModel, Exchange } from '../../types'
import { ExchangeConfigModal } from '../trader/ExchangeConfigModal'
import { ModelConfigModal } from '../trader/ModelConfigModal'
import { MessagingChannelsPanel } from './MessagingChannelsPanel'
import { AgentConfigPanel } from './AgentConfigPanel'

type Kind = 'model' | 'agent' | 'platform' | 'exchange'

function healthDot(row: {
  tested?: boolean
  last_test_ok?: boolean
}): { label: string; className: string } {
  if (!row.tested) {
    return {
      label: '—',
      className: 'bg-strand-bg-deeper text-strand-text-muted',
    }
  }
  if (row.last_test_ok) {
    return {
      label: 'OK',
      className: 'bg-strand-success/10 text-strand-success',
    }
  }
  return { label: 'FAIL', className: 'bg-red-500/10 text-red-500' }
}

export function ResourcesHubPanel() {
  const { language } = useLanguage()
  const [searchParams, setSearchParams] = useSearchParams()
  const [loading, setLoading] = useState(true)
  const [testingKey, setTestingKey] = useState<string | null>(null)
  const [overview, setOverview] = useState<ResourcesOverview | null>(null)
  const [defaults, setDefaults] = useState<ResourceDefaults>({
    default_model_id: '',
    default_agent_id: '',
    default_exchange_id: '',
    alert_platform_ids: [],
  })
  const [savingDefaults, setSavingDefaults] = useState(false)

  const [configuredModels, setConfiguredModels] = useState<AIModel[]>([])
  const [supportedModels, setSupportedModels] = useState<AIModel[]>([])
  const [exchanges, setExchanges] = useState<Exchange[]>([])
  const [showModelModal, setShowModelModal] = useState(false)
  const [editingModel, setEditingModel] = useState<string | null>(null)
  const [initialModelId, setInitialModelId] = useState<string | null>(null)
  const [showExchangeModal, setShowExchangeModal] = useState(false)
  const [editingExchange, setEditingExchange] = useState<string | null>(null)
  const [initialExchangeType, setInitialExchangeType] = useState<string | null>(
    null
  )
  const [showAgentConfig, setShowAgentConfig] = useState(false)
  const [editingAgent, setEditingAgent] = useState<string | null>(null)

  const refreshConfigs = useCallback(async () => {
    const [models, supported, exchangeConfigs] = await Promise.all([
      api.getModelConfigs(),
      api.getSupportedModels(),
      api.getExchangeConfigs(),
    ])
    setConfiguredModels(models)
    setSupportedModels(supported)
    setExchanges(exchangeConfigs)
  }, [])

  const refresh = useCallback(async (opts?: { quiet?: boolean }) => {
    const quiet = Boolean(opts?.quiet)
    if (!quiet) setLoading(true)
    try {
      const [data] = await Promise.all([
        resourcesApi.overview(),
        refreshConfigs(),
      ])
      setOverview(data)
      setDefaults({
        default_model_id: data.defaults?.default_model_id || '',
        default_agent_id: data.defaults?.default_agent_id || '',
        default_exchange_id: data.defaults?.default_exchange_id || '',
        alert_platform_ids: data.defaults?.alert_platform_ids || [],
      })
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : t('resourcesLoadFailed', language)
      )
    } finally {
      if (!quiet) setLoading(false)
    }
  }, [language, refreshConfigs])

  useEffect(() => {
    void refresh()
  }, [refresh])

  // Deep-link: ?tab=resources&configure=model|exchange&id=&type=
  useEffect(() => {
    const configure = searchParams.get('configure')
    if (configure !== 'model' && configure !== 'exchange' && configure !== 'agent')
      return

    if (configure === 'model') {
      const id = searchParams.get('id')
      setEditingModel(id)
      setInitialModelId(id || searchParams.get('type') || null)
      setShowModelModal(true)
      setShowExchangeModal(false)
      setShowAgentConfig(false)
    } else if (configure === 'exchange') {
      const id = searchParams.get('id')
      setEditingExchange(id)
      setInitialExchangeType(id ? null : searchParams.get('type'))
      setShowExchangeModal(true)
      setShowModelModal(false)
      setShowAgentConfig(false)
    } else if (configure === 'agent') {
      setEditingAgent(searchParams.get('id'))
      setShowAgentConfig(true)
      setShowModelModal(false)
      setShowExchangeModal(false)
    } else {
      return
    }

    const next = new URLSearchParams(searchParams)
    next.delete('configure')
    next.delete('id')
    next.delete('type')
    if (!next.get('tab')) next.set('tab', 'resources')
    setSearchParams(next, { replace: true })
  }, [searchParams, setSearchParams])

  const patchOverviewHealth = (
    kind: Kind,
    id: string,
    patch: {
      tested?: boolean
      last_test_ok?: boolean
      latency_ms?: number
      last_error?: string
    }
  ) => {
    setOverview((prev) => {
      if (!prev) return prev
      const listKey =
        kind === 'model'
          ? 'models'
          : kind === 'agent'
            ? 'agents'
            : kind === 'platform'
              ? 'platforms'
              : 'exchanges'
      const list = prev[listKey]
      const idx = list.findIndex((row) => row.id === id)
      if (idx < 0) return prev
      const nextList = list.slice()
      nextList[idx] = { ...nextList[idx], ...patch }
      return { ...prev, [listKey]: nextList }
    })
  }

  const runTest = async (kind: Kind, id: string) => {
    const key = `${kind}:${id}`
    setTestingKey(key)
    try {
      const result = await resourcesApi.test(kind, id)
      // Soft-update the row in place so list order does not reshuffle.
      patchOverviewHealth(kind, id, {
        tested: true,
        last_test_ok: result.ok,
        latency_ms: result.latency_ms,
        last_error: result.ok ? undefined : result.error || '',
      })
      if (result.ok) {
        toast.success(
          t('resourcesTestOk', language, {
            ms: String(result.latency_ms ?? 0),
          })
        )
      } else {
        toast.error(result.error || t('resourcesTestFailed', language))
      }
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : t('resourcesTestFailed', language)
      )
    } finally {
      setTestingKey(null)
    }
  }

  const retestAll = async () => {
    if (!overview) return
    const jobs: Array<{ kind: Kind; id: string }> = [
      ...overview.models
        .filter((m) => m.enabled !== false)
        .map((m) => ({ kind: 'model' as const, id: m.id })),
      ...overview.exchanges
        .filter((e) => e.enabled !== false)
        .map((e) => ({ kind: 'exchange' as const, id: e.id })),
      ...overview.platforms
        .filter((p) => p.enabled !== false)
        .map((p) => ({ kind: 'platform' as const, id: p.id })),
      ...overview.agents
        .filter((a) => a.configured)
        .map((a) => ({ kind: 'agent' as const, id: a.id })),
    ]
    for (const job of jobs) {
      await runTest(job.kind, job.id)
    }
  }

  const saveDefaults = async () => {
    setSavingDefaults(true)
    try {
      const next = await resourcesApi.saveDefaults(defaults)
      setDefaults(next)
      if (next.default_model_id) {
        setPreferredModelId(next.default_model_id)
      }
      toast.success(t('resourcesDefaultsSaved', language))
    } catch (e) {
      toast.error(
        e instanceof Error
          ? e.message
          : t('resourcesDefaultsSaveFailed', language)
      )
    } finally {
      setSavingDefaults(false)
    }
  }

  const toggleAlertPlatform = (id: string) => {
    setDefaults((d) => {
      const set = new Set(d.alert_platform_ids || [])
      if (set.has(id)) set.delete(id)
      else set.add(id)
      return { ...d, alert_platform_ids: Array.from(set) }
    })
  }

  const openAddModel = () => {
    setEditingModel(null)
    setInitialModelId(null)
    setShowModelModal(true)
    setShowExchangeModal(false)
    setShowAgentConfig(false)
  }

  const openEditModel = (id: string) => {
    setEditingModel(id)
    setInitialModelId(null)
    setShowModelModal(true)
    setShowExchangeModal(false)
    setShowAgentConfig(false)
  }

  const openAddExchange = (type?: string | null) => {
    setEditingExchange(null)
    setInitialExchangeType(type || null)
    setShowExchangeModal(true)
    setShowModelModal(false)
    setShowAgentConfig(false)
  }

  const openEditExchange = (id: string) => {
    setEditingExchange(id)
    setInitialExchangeType(null)
    setShowExchangeModal(true)
    setShowModelModal(false)
    setShowAgentConfig(false)
  }

  const openAgentConfig = (id?: string | null) => {
    setEditingAgent(id || null)
    setShowAgentConfig(true)
    setShowModelModal(false)
    setShowExchangeModal(false)
  }

  const closeConfigPages = () => {
    setShowModelModal(false)
    setShowExchangeModal(false)
    setShowAgentConfig(false)
    setEditingModel(null)
    setInitialModelId(null)
    setEditingExchange(null)
    setInitialExchangeType(null)
    setEditingAgent(null)
  }

  const handleSaveModel = async (
    modelId: string,
    apiKey: string,
    customApiUrl?: string,
    customModelName?: string
  ) => {
    try {
      const existingModel = configuredModels.find(
        (m) =>
          m.id === modelId ||
          m.provider === modelId ||
          (modelId && m.provider === modelId)
      )
      const modelTemplate = supportedModels.find(
        (m) => m.id === modelId || m.provider === modelId
      )
      const modelToUpdate = existingModel || modelTemplate
      if (!modelToUpdate) {
        throw new Error(t('settingsModelNotFound', language))
      }

      let updatedModels: AIModel[]
      if (existingModel) {
        updatedModels = configuredModels.map((m) =>
          m.id === existingModel.id
            ? {
                ...m,
                apiKey: apiKey || m.apiKey || '',
                customApiUrl: customApiUrl || m.customApiUrl || '',
                customModelName: customModelName || m.customModelName || '',
                enabled: true,
              }
            : m
        )
      } else {
        updatedModels = [
          ...configuredModels,
          {
            ...modelToUpdate,
            id: modelToUpdate.id || modelToUpdate.provider || modelId,
            provider: modelToUpdate.provider || modelId,
            apiKey,
            customApiUrl: customApiUrl || '',
            customModelName: customModelName || '',
            enabled: true,
          },
        ]
      }

      await api.updateModelConfigs({
        models: Object.fromEntries(
          updatedModels.map((m) => [
            m.id || m.provider,
            {
              enabled: m.enabled,
              api_key: m.apiKey || '',
              custom_api_url: m.customApiUrl || '',
              custom_model_name: m.customModelName || '',
            },
          ])
        ),
      })
      // Confirm the row actually exists with a usable key before claiming success.
      await refresh()
      const savedId = modelToUpdate.id || modelToUpdate.provider || modelId
      const saved = (
        await api.getModelConfigs().catch(() => [] as AIModel[])
      ).find(
        (m) =>
          m.id === savedId ||
          m.provider === savedId ||
          m.provider === modelToUpdate.provider
      )
      if (!saved || (!saved.has_api_key && !saved.apiKey && apiKey)) {
        throw new Error(t('settingsModelSaveFailed', language))
      }
      toast.success(t('settingsModelSaved', language))
      setShowModelModal(false)
      setEditingModel(null)
      setInitialModelId(null)
    } catch (error) {
      // Surface via ModelConfigModal (toast + inline). Avoid duplicate toasts.
      throw error instanceof Error
        ? error
        : new Error(t('settingsModelSaveFailed', language))
    }
  }

  const handleDeleteModel = async (modelId: string) => {
    try {
      const updatedModels = configuredModels.map((m) =>
        m.id === modelId
          ? {
              ...m,
              apiKey: '',
              customApiUrl: '',
              customModelName: '',
              enabled: false,
            }
          : m
      )
      await api.updateModelConfigs({
        models: Object.fromEntries(
          updatedModels.map((m) => [
            m.provider,
            {
              enabled: m.enabled,
              api_key: m.apiKey || '',
              custom_api_url: m.customApiUrl || '',
              custom_model_name: m.customModelName || '',
            },
          ])
        ),
      })
      toast.success(t('settingsModelRemoved', language))
      setShowModelModal(false)
      setEditingModel(null)
      await refresh()
    } catch {
      toast.error(t('settingsModelRemoveFailed', language))
    }
  }

  const handleSaveExchange = async (
    exchangeId: string | null,
    exchangeType: string,
    accountName: string,
    apiKey: string,
    secretKey?: string,
    passphrase?: string,
    testnet?: boolean,
    hyperliquidWalletAddr?: string,
    asterUser?: string,
    asterSigner?: string,
    asterPrivateKey?: string,
    lighterWalletAddr?: string,
    lighterPrivateKey?: string,
    lighterApiKeyPrivateKey?: string,
    lighterApiKeyIndex?: number
  ) => {
    try {
      if (exchangeId) {
        await api.updateExchangeConfigsEncrypted({
          exchanges: {
            [exchangeId]: {
              enabled: true,
              api_key: apiKey || '',
              secret_key: secretKey || '',
              passphrase: passphrase || '',
              testnet: testnet || false,
              hyperliquid_wallet_addr: hyperliquidWalletAddr || '',
              hyperliquid_unified_account: exchangeType === 'hyperliquid',
              aster_user: asterUser || '',
              aster_signer: asterSigner || '',
              aster_private_key: asterPrivateKey || '',
              lighter_wallet_addr: lighterWalletAddr || '',
              lighter_private_key: lighterPrivateKey || '',
              lighter_api_key_private_key: lighterApiKeyPrivateKey || '',
              lighter_api_key_index: lighterApiKeyIndex || 0,
            },
          },
        })
        toast.success(t('settingsExchangeUpdated', language))
      } else {
        await api.createExchangeEncrypted({
          exchange_type: exchangeType,
          account_name: accountName,
          enabled: true,
          api_key: apiKey || '',
          secret_key: secretKey || '',
          passphrase: passphrase || '',
          testnet: testnet || false,
          hyperliquid_wallet_addr: hyperliquidWalletAddr || '',
          hyperliquid_unified_account: exchangeType === 'hyperliquid',
          aster_user: asterUser || '',
          aster_signer: asterSigner || '',
          aster_private_key: asterPrivateKey || '',
          lighter_wallet_addr: lighterWalletAddr || '',
          lighter_private_key: lighterPrivateKey || '',
          lighter_api_key_private_key: lighterApiKeyPrivateKey || '',
          lighter_api_key_index: lighterApiKeyIndex || 0,
        })
        toast.success(t('settingsExchangeCreated', language))
      }
      setShowExchangeModal(false)
      setEditingExchange(null)
      setInitialExchangeType(null)
      await refresh()
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('settingsExchangeSaveFailed', language)
      )
      throw error instanceof Error
        ? error
        : new Error(t('settingsExchangeSaveFailed', language))
    }
  }

  const handleDeleteExchange = async (exchangeId: string) => {
    try {
      await api.deleteExchange(exchangeId)
      toast.success(t('settingsExchangeDeleted', language))
      setShowExchangeModal(false)
      setEditingExchange(null)
      await refresh()
    } catch {
      toast.error(t('settingsExchangeDeleteFailed', language))
    }
  }

  if (loading && !overview) {
    return (
      <p className="text-sm text-strand-text-muted">{t('loading', language)}</p>
    )
  }

  const preferredLocal = getPreferredModelId()

  if (showModelModal) {
    return (
      <div className="space-y-4">
        <button
          type="button"
          onClick={closeConfigPages}
          className="inline-flex items-center gap-1 text-xs text-strand-text-muted hover:text-strand-text"
        >
          <ChevronLeft size={14} />
          {t('resourcesTab', language)}
        </button>
        <ModelConfigModal
          key={editingModel || initialModelId || 'add-model'}
          embedded
          allModels={supportedModels}
          configuredModels={configuredModels}
          editingModelId={editingModel}
          initialModelId={initialModelId}
          onSave={handleSaveModel}
          onDelete={handleDeleteModel}
          onClose={closeConfigPages}
          language={language}
        />
      </div>
    )
  }

  if (showExchangeModal) {
    return (
      <div className="space-y-4">
        <button
          type="button"
          onClick={closeConfigPages}
          className="inline-flex items-center gap-1 text-xs text-strand-text-muted hover:text-strand-text"
        >
          <ChevronLeft size={14} />
          {t('resourcesTab', language)}
        </button>
        <ExchangeConfigModal
          embedded
          allExchanges={exchanges}
          editingExchangeId={editingExchange}
          initialExchangeType={initialExchangeType}
          onSave={handleSaveExchange}
          onDelete={handleDeleteExchange}
          onClose={closeConfigPages}
          language={language}
        />
      </div>
    )
  }

  if (showAgentConfig) {
    return (
      <AgentConfigPanel
        agents={(overview?.agents || []).map((a) => ({
          id: a.id,
          name: a.name,
          configured: a.configured,
        }))}
        initialAgentId={editingAgent}
        onBack={closeConfigPages}
        onSaved={async () => {
          await refresh()
          closeConfigPages()
        }}
      />
    )
  }

  return (
    <div className="space-y-8">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-strand-text mb-1">
            {t('resourcesTitle', language)}
          </h3>
          <p className="text-xs text-strand-text-muted leading-relaxed">
            {t('resourcesDesc', language)}
          </p>
        </div>
        <div className="flex gap-2 shrink-0">
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
            onClick={() => void retestAll()}
            className="flex items-center gap-1 text-xs px-3 py-1.5 rounded-lg bg-strand-gold/10 text-strand-gold font-medium"
          >
            <Activity size={14} />
            {t('resourcesRetestAll', language)}
          </button>
        </div>
      </div>

      <ResourcesUsageStrip usage={overview?.usage} language={language} />

      <section className="space-y-3 border-t border-strand-border/40 pt-6">
        <div className="flex items-center justify-between gap-3">
          <h4 className="text-sm font-medium text-strand-text">
            {t('resourcesSceneDefaults', language)}
          </h4>
          <button
            type="button"
            disabled={savingDefaults}
            onClick={() => void saveDefaults()}
            className="text-xs px-3 py-1.5 rounded-lg bg-strand-gold text-strand-bg font-medium disabled:opacity-50"
          >
            {t('save', language)}
          </button>
        </div>
        <p className="text-xs text-strand-text-muted">
          {t('resourcesSceneHint', language)}
        </p>
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="block text-xs text-strand-text-muted">
            {t('resourcesDefaultModel', language)}
            <select
              className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text"
              value={defaults.default_model_id}
              onChange={(e) =>
                setDefaults((d) => ({ ...d, default_model_id: e.target.value }))
              }
            >
              <option value="">{t('resourcesAuto', language)}</option>
              {(overview?.models || [])
                .filter((m) => m.enabled !== false)
                .map((m) => (
                  <option key={m.id} value={m.id}>
                    {String(m.name || m.id)}
                    {m.last_test_ok === false && m.tested ? ' ✗' : ''}
                    {preferredLocal === m.id ? ' ★' : ''}
                  </option>
                ))}
            </select>
          </label>
          <label className="block text-xs text-strand-text-muted">
            {t('resourcesDefaultAgent', language)}
            <select
              className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text"
              value={defaults.default_agent_id}
              onChange={(e) =>
                setDefaults((d) => ({ ...d, default_agent_id: e.target.value }))
              }
            >
              <option value="">{t('resourcesAuto', language)}</option>
              {(overview?.agents || []).map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                  {a.configured ? ' · in use' : ''}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-xs text-strand-text-muted sm:col-span-2">
            {t('resourcesDefaultExchange', language)}
            <select
              className="mt-1 w-full rounded-lg border border-strand-border/50 bg-strand-bg-lighter px-3 py-2 text-sm text-strand-text"
              value={defaults.default_exchange_id}
              onChange={(e) =>
                setDefaults((d) => ({
                  ...d,
                  default_exchange_id: e.target.value,
                }))
              }
            >
              <option value="">{t('resourcesAuto', language)}</option>
              {(overview?.exchanges || [])
                .filter((e) => e.enabled !== false)
                .map((e) => (
                  <option key={e.id} value={e.id}>
                    {String(e.account_name || e.name || e.id)}
                    {e.last_test_ok === false && e.tested ? ' ✗' : ''}
                  </option>
                ))}
            </select>
          </label>
        </div>

        <div className="space-y-2">
          <p className="text-xs font-medium text-strand-text-muted">
            {t('resourcesAlertPlatforms', language)}
          </p>
          <p className="text-[11px] text-strand-text-muted">
            {t('resourcesAlertHint', language)}
          </p>
          {(overview?.platforms || []).filter((p) => p.alerts_enabled)
            .length === 0 ? (
            <p className="text-xs text-strand-text-muted">
              {t('resourcesNoAlertPlatforms', language)}{' '}
              <a
                href="#resource-platforms"
                className="text-strand-gold hover:underline"
              >
                {t('messagingTab', language)}
              </a>
            </p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {(overview?.platforms || [])
                .filter((p) => p.alerts_enabled && p.enabled !== false)
                .map((p) => {
                  const on = (defaults.alert_platform_ids || []).includes(p.id)
                  return (
                    <button
                      key={p.id}
                      type="button"
                      onClick={() => toggleAlertPlatform(p.id)}
                      className={`text-xs px-2.5 py-1 rounded-lg border transition-colors ${
                        on
                          ? 'border-strand-gold bg-strand-gold/10 text-strand-gold'
                          : 'border-strand-border/50 text-strand-text-muted'
                      }`}
                    >
                      {String(p.name || p.id)}
                    </button>
                  )
                })}
            </div>
          )}
        </div>
      </section>

      <ResourceSection
        title={t('aiModels', language)}
        icon={<Cpu size={14} />}
        empty={t('settingsNoModelsYet', language)}
        addLabel={t('settingsAddModel', language)}
        onAdd={openAddModel}
        rows={(overview?.models || []).map((m) => {
          const provider = String(m.provider || '')
          const parts: string[] = []
          if (provider) parts.push(provider)
          if (
            provider.toLowerCase() === 'claw402' &&
            m.balance_usdc != null &&
            String(m.balance_usdc).trim() !== ''
          ) {
            const bal = Number(m.balance_usdc)
            parts.push(
              Number.isFinite(bal)
                ? `${bal.toFixed(2)} USDC`
                : `${m.balance_usdc} USDC`
            )
          }
          const todayUsd = Number(m.spend_today_usd ?? 0)
          const todayCalls = Number(m.spend_today_calls ?? 0)
          const weekUsd = Number(m.spend_week_usd ?? 0)
          const weekCalls = Number(m.spend_week_calls ?? 0)
          const hasSpend =
            (Number.isFinite(todayUsd) && todayUsd > 0) ||
            (Number.isFinite(weekUsd) && weekUsd > 0) ||
            todayCalls > 0 ||
            weekCalls > 0
          return {
            id: m.id,
            title: String(m.name || m.id),
            subtitle: parts.join(' · '),
            meta: hasSpend
              ? t('resourcesUsageModelToday', language, {
                  usd: formatUsd(todayUsd),
                  n: todayCalls,
                }) +
                ' · ' +
                t('resourcesUsageModelWeek', language, {
                  usd: formatUsd(weekUsd),
                  n: weekCalls,
                })
              : undefined,
            enabled: m.enabled !== false,
            tested: m.tested,
            last_test_ok: m.last_test_ok,
            last_error: m.last_error,
            latency_ms: m.latency_ms,
            kind: 'model' as const,
          }
        })}
        testingKey={testingKey}
        onTest={(id) => void runTest('model', id)}
        onEdit={openEditModel}
        language={language}
      />

      <ResourceSection
        title={t('resourcesAgents', language)}
        icon={<Terminal size={14} />}
        empty={t('resourcesNoAgents', language)}
        addLabel={t('resourcesConfigure', language)}
        onAdd={() => openAgentConfig()}
        rows={(overview?.agents || []).map((a) => ({
          id: a.id,
          title: a.name,
          subtitle: a.configured
            ? t('resourcesAgentConfigured', language)
            : t('resourcesAgentAvailable', language),
          enabled: true,
          tested: a.tested,
          last_test_ok: a.last_test_ok,
          last_error: a.last_error,
          latency_ms: a.latency_ms,
          kind: 'agent' as const,
        }))}
        testingKey={testingKey}
        onTest={(id) => void runTest('agent', id)}
        onEdit={(id) => openAgentConfig(id)}
        language={language}
        hint={t('resourcesAgentHint', language)}
      />

      <section
        id="resource-platforms"
        className="space-y-3 border-t border-strand-border/40 pt-6"
      >
        <h4 className="text-sm font-medium text-strand-text flex items-center gap-2">
          <MessageSquare size={14} />
          {t('messagingTab', language)}
        </h4>
        <p className="text-[11px] text-strand-text-muted leading-relaxed">
          {t('resourcesPlatformHint', language)}
        </p>
        {(overview?.platforms || []).length > 0 && (
          <ul className="space-y-2 mb-4">
            {(overview?.platforms || []).map((p) => {
              const badge = healthDot(p)
              const key = `platform:${p.id}`
              return (
                <li
                  key={p.id}
                  className="rounded-xl border border-strand-border/40 bg-strand-bg/40 px-3.5 py-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="min-w-0">
                    <div className="text-sm font-medium text-strand-text truncate">
                      {String(p.name || p.id)}
                    </div>
                    <div className="text-[11px] text-strand-text-muted mt-0.5 truncate">
                      {String(p.platform_type || '')}
                      {p.latency_ms != null && p.tested
                        ? ` · ${p.latency_ms}ms`
                        : ''}
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <span
                      className={`text-[11px] px-2 py-0.5 rounded-full ${badge.className}`}
                    >
                      {badge.label}
                    </span>
                    <button
                      type="button"
                      disabled={testingKey === key}
                      onClick={() => void runTest('platform', p.id)}
                      className="text-xs px-2.5 py-1 rounded-lg border border-strand-border/50 text-strand-text hover:bg-strand-bg disabled:opacity-50"
                    >
                      {testingKey === key
                        ? t('resourcesTesting', language)
                        : t('resourcesTest', language)}
                    </button>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
        <MessagingChannelsPanel
          embedded
          platformsOnly
          onChanged={() => void refresh({ quiet: true })}
        />
      </section>

      <ResourceSection
        title={t('exchanges', language)}
        icon={<Building2 size={14} />}
        empty={t('settingsNoExchangesYet', language)}
        addLabel={t('settingsAddExchange', language)}
        onAdd={() => openAddExchange()}
        rows={(overview?.exchanges || []).map((e) => ({
          id: e.id,
          title: String(e.account_name || e.name || e.id),
          subtitle: String(e.exchange_type || ''),
          enabled: e.enabled !== false,
          tested: e.tested,
          last_test_ok: e.last_test_ok,
          last_error: e.last_error,
          latency_ms: e.latency_ms,
          kind: 'exchange' as const,
        }))}
        testingKey={testingKey}
        onTest={(id) => void runTest('exchange', id)}
        onEdit={openEditExchange}
        language={language}
      />
    </div>
  )
}

function formatUsd(n: number | undefined | null): string {
  if (n == null || !Number.isFinite(n)) return '—'
  if (Math.abs(n) >= 100) return `$${n.toFixed(0)}`
  if (Math.abs(n) >= 1) return `$${n.toFixed(2)}`
  if (Math.abs(n) === 0) return '$0'
  return `$${n.toFixed(4)}`
}

function readUsageBucket(
  raw: { usd?: number; calls?: number } | number | undefined
): { usd: number; calls: number } {
  if (raw == null) return { usd: 0, calls: 0 }
  if (typeof raw === 'number') return { usd: raw, calls: 0 }
  return {
    usd: Number(raw.usd ?? 0) || 0,
    calls: Number(raw.calls ?? 0) || 0,
  }
}

function ResourcesUsageStrip({
  usage,
  language,
}: {
  usage?: ResourcesUsage
  language: 'en' | 'zh' | 'id'
}) {
  const claw = usage?.claw402
  const spend = usage?.spend
  const balRaw = claw?.balance_usdc
  const balNum =
    balRaw != null && String(balRaw).trim() !== '' ? Number(balRaw) : NaN
  const hasClaw = Boolean(claw?.wallet_address)
  const status = claw?.balance_status || (hasClaw ? 'unknown' : '')
  const runway =
    claw?.est_runway_days != null && Number.isFinite(claw.est_runway_days)
      ? Math.max(0, claw.est_runway_days)
      : null

  let remainingLabel = t('resourcesUsageNoClaw', language)
  let remainingTone = 'text-strand-text-muted'
  if (hasClaw) {
    if (status === 'ready' && Number.isFinite(balNum)) {
      remainingLabel = `${balNum.toFixed(2)} USDC`
      remainingTone =
        balNum < 1 ? 'text-amber-500' : 'text-strand-success'
    } else if (status === 'empty') {
      remainingLabel = '0.00 USDC'
      remainingTone = 'text-red-500'
    } else if (Number.isFinite(balNum)) {
      remainingLabel = `${balNum.toFixed(2)} USDC`
      remainingTone = 'text-strand-text'
    } else {
      remainingLabel = t('resourcesUsageUnknown', language)
      remainingTone = 'text-strand-text-muted'
    }
  }

  return (
    <section className="space-y-3 border-t border-strand-border/40 pt-6">
      <div className="flex items-center gap-2">
        <Wallet size={14} className="text-strand-text-muted" />
        <h4 className="text-sm font-medium text-strand-text">
          {t('resourcesUsageTitle', language)}
        </h4>
      </div>
      <p className="text-[11px] text-strand-text-muted leading-relaxed">
        {t('resourcesUsageDesc', language)}
      </p>
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="min-w-0">
          <div className="text-[11px] text-strand-text-muted">
            {t('resourcesUsageRemaining', language)}
          </div>
          <div className={`mt-1 text-sm font-medium tabular-nums ${remainingTone}`}>
            {remainingLabel}
          </div>
          {hasClaw && status === 'empty' ? (
            <p className="mt-1 text-[11px] text-red-500/90">
              {t('resourcesUsageEmpty', language)}
            </p>
          ) : hasClaw && runway != null && runway > 0 ? (
            <p className="mt-1 text-[11px] text-strand-text-muted">
              {t('resourcesUsageRunway', language, {
                days: runway >= 10 ? runway.toFixed(0) : runway.toFixed(1),
              })}
            </p>
          ) : !hasClaw ? (
            <p className="mt-1 text-[11px] text-strand-text-muted">
              {t('resourcesUsageByokHint', language)}
            </p>
          ) : null}
        </div>
        <div className="min-w-0">
          <div className="text-[11px] text-strand-text-muted">
            {t('resourcesUsageSpentToday', language)}
          </div>
          <div className="mt-1 text-sm font-medium tabular-nums text-strand-text">
            {formatUsd(spend?.today_usd ?? 0)}
          </div>
          <p className="mt-1 text-[11px] text-strand-text-muted">
            {t('resourcesUsageCalls', language, {
              n: spend?.today_calls ?? 0,
            })}
          </p>
        </div>
        <div className="min-w-0">
          <div className="text-[11px] text-strand-text-muted">
            {t('resourcesUsageSpentWeek', language)}
          </div>
          <div className="mt-1 text-sm font-medium tabular-nums text-strand-text">
            {formatUsd(spend?.week_usd ?? 0)}
          </div>
          <p className="mt-1 text-[11px] text-strand-text-muted">
            {t('resourcesUsageCalls', language, {
              n: spend?.week_calls ?? 0,
            })}
          </p>
        </div>
      </div>
      {(() => {
        const byModel = spend?.by_model_today || {}
        const entries = Object.entries(byModel)
          .map(([name, raw]) => {
            const b = readUsageBucket(raw)
            return { name, ...b }
          })
          .filter((e) => e.usd > 0 || e.calls > 0)
          .sort((a, b) => b.usd - a.usd)
          .slice(0, 8)
        if (entries.length === 0) return null
        return (
          <div className="space-y-1.5 pt-1">
            <p className="text-[11px] font-medium text-strand-text-muted">
              {t('resourcesUsageByModel', language)}
            </p>
            <ul className="space-y-1">
              {entries.map((e) => (
                <li
                  key={e.name}
                  className="flex items-baseline justify-between gap-3 text-[11px]"
                >
                  <span className="truncate text-strand-text">{e.name}</span>
                  <span className="shrink-0 tabular-nums text-strand-text-muted">
                    {formatUsd(e.usd)}
                    {e.calls > 0 ? ` · ${e.calls}` : ''}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )
      })()}
    </section>
  )
}

function ResourceSection({
  title,
  icon,
  empty,
  rows,
  testingKey,
  onTest,
  onEdit,
  onEditHref,
  onAdd,
  onAddHref,
  addLabel,
  hint,
  language,
}: {
  title: string
  icon: React.ReactNode
  empty: string
  rows: Array<{
    id: string
    title: string
    subtitle: string
    meta?: string
    enabled: boolean
    tested?: boolean
    last_test_ok?: boolean
    last_error?: string
    latency_ms?: number
    kind: Kind
  }>
  testingKey: string | null
  onTest: (id: string) => void
  onEdit?: (id: string) => void
  onEditHref?: string
  onAdd?: () => void
  onAddHref?: string
  addLabel?: string
  hint?: string
  language: 'en' | 'zh' | 'id'
}) {
  return (
    <section className="space-y-3 border-t border-strand-border/40 pt-6">
      <div className="flex items-center justify-between gap-3">
        <h4 className="text-sm font-medium text-strand-text flex items-center gap-2">
          {icon}
          {title}
        </h4>
        {onAdd && addLabel && (
          <button
            type="button"
            onClick={onAdd}
            className="flex items-center gap-1 text-xs px-2.5 py-1 rounded-lg bg-strand-gold/10 text-strand-gold font-medium"
          >
            <Plus size={14} />
            {addLabel}
          </button>
        )}
        {onAddHref && addLabel && (
          <Link
            to={onAddHref}
            className="flex items-center gap-1 text-xs px-2.5 py-1 rounded-lg bg-strand-gold/10 text-strand-gold font-medium"
          >
            <Plus size={14} />
            {addLabel}
          </Link>
        )}
      </div>
      {hint && (
        <p className="text-[11px] text-strand-text-muted leading-relaxed">
          {hint}
        </p>
      )}
      {rows.length === 0 ? (
        <p className="text-xs text-strand-text-muted">{empty}</p>
      ) : (
        <ul className="space-y-2">
          {rows.map((row) => {
            const badge = healthDot(row)
            const key = `${row.kind}:${row.id}`
            return (
              <li
                key={row.id}
                className="rounded-xl border border-strand-border/40 bg-strand-bg/40 px-3.5 py-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-medium text-strand-text truncate">
                    {row.title}
                  </div>
                  <div className="text-[11px] text-strand-text-muted mt-0.5 truncate">
                    {row.subtitle}
                    {row.latency_ms != null && row.tested
                      ? ` · ${row.latency_ms}ms`
                      : ''}
                  </div>
                  {row.meta ? (
                    <p className="text-[11px] text-strand-text-muted/90 mt-0.5 tabular-nums truncate">
                      {row.meta}
                    </p>
                  ) : null}
                  {row.last_error && row.tested && !row.last_test_ok ? (
                    <p
                      className="text-[11px] text-red-500/90 mt-1 line-clamp-2"
                      title={row.last_error}
                    >
                      {row.last_error}
                    </p>
                  ) : null}
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <span
                    className={`text-[11px] px-2 py-0.5 rounded-full ${badge.className}`}
                  >
                    {badge.label}
                  </span>
                  {onEdit && (
                    <button
                      type="button"
                      onClick={() => onEdit(row.id)}
                      className="p-1.5 rounded-lg border border-strand-border/50 text-strand-text-muted hover:text-strand-text hover:bg-strand-bg"
                      title={t('resourcesConfigure', language)}
                    >
                      <Pencil size={13} />
                    </button>
                  )}
                  {onEditHref && (
                    <Link
                      to={onEditHref}
                      className="p-1.5 rounded-lg border border-strand-border/50 text-strand-text-muted hover:text-strand-text hover:bg-strand-bg"
                      title={t('resourcesConfigure', language)}
                    >
                      <Pencil size={13} />
                    </Link>
                  )}
                  <button
                    type="button"
                    disabled={testingKey === key}
                    onClick={() => onTest(row.id)}
                    className="text-xs px-2.5 py-1 rounded-lg border border-strand-border/50 text-strand-text hover:bg-strand-bg disabled:opacity-50"
                  >
                    {testingKey === key
                      ? t('resourcesTesting', language)
                      : t('resourcesTest', language)}
                  </button>
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}
