import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router'
import useSWR from 'swr'
import { api } from '../../lib/api'
import { ApiError } from '../../lib/httpClient'
import { ROUTES, buildResourcesConfigurePath } from '../../router/paths'
import type {
  TraderInfo,
  CreateTraderRequest,
  AIModel,
  Exchange,
  ExchangeAccountState,
} from '../../types'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { useAuth } from '../../contexts/AuthContext'
import { TraderConfigModal } from './TraderConfigModal'
import { TradersList } from './TradersList'
import { AutopilotLaunchPanel } from './AutopilotLaunchPanel'
import {
  Bot,
  Plus,
  ArrowUpRight,
  Cable,
  RefreshCw,
  Settings2,
} from 'lucide-react'
import './ConfigPage.css'
import { confirmToast } from '../../lib/notify'
import { toast } from 'sonner'
import {
  describeLaunchFailures,
  primaryFailingExchangeId,
  primarySetupTarget,
} from '../../lib/launch/preflight'
import type { LaunchPreflightResult } from '../../lib/launch/types'
import { selectableResources } from '../../lib/preferredModel'

interface AITradersPageProps {
  onTraderSelect?: (traderId: string) => void
}

export function AITradersPage({ onTraderSelect }: AITradersPageProps) {
  const { language } = useLanguage()
  const { user, token } = useAuth()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [configsLoading, setConfigsLoading] = useState(true)
  const [configError, setConfigError] = useState('')
  const [showLaunch, setShowLaunch] = useState(
    searchParams.get('setup') === 'hyperliquid-funds'
  )
  const configView =
    searchParams.get('view') === 'autopilot' ||
    searchParams.get('setup') === 'hyperliquid-funds'
      ? 'autopilot'
      : 'connections'
  const changeConfigView = (view: string) => {
    const next = new URLSearchParams(searchParams)
    if (view === 'autopilot') next.set('view', view)
    else next.delete('view')
    setSearchParams(next, { replace: true })
  }
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [editingTrader, setEditingTrader] = useState<any>(null)
  const [allModels, setAllModels] = useState<AIModel[]>([])
  const [allExchanges, setAllExchanges] = useState<Exchange[]>([])
  const [exchangeAccountStates, setExchangeAccountStates] = useState<
    Record<string, ExchangeAccountState>
  >({})
  const [visibleTraderAddresses, setVisibleTraderAddresses] = useState<
    Set<string>
  >(new Set())
  const [copiedId, setCopiedId] = useState<string | null>(null)

  const refreshAccountStates = async () => {
    try {
      const response = await api.getExchangeAccountState()
      setExchangeAccountStates(response.states || {})
    } catch {
      setExchangeAccountStates({})
    }
  }

  const goConfigureModel = (id?: string | null) => {
    navigate(buildResourcesConfigurePath({ kind: 'model', id }))
  }

  const goConfigureExchange = (id?: string | null, type?: string | null) => {
    navigate(buildResourcesConfigurePath({ kind: 'exchange', id, type }))
  }

  const goResources = () => navigate(ROUTES.resources)

  const loadConfigs = async () => {
    setConfigsLoading(true)
    setConfigError('')
    if (!user || !token) {
      setConfigsLoading(false)
      return
    }

    try {
      const [modelConfigs, exchangeConfigs] = await Promise.all([
        api.getModelConfigs(),
        api.getExchangeConfigs(),
        refreshAccountStates(),
      ])
      setAllModels(modelConfigs)
      setAllExchanges(exchangeConfigs)
    } catch (error) {
      setConfigError(
        'Connections could not be loaded. Saved credentials have not been changed.'
      )
      throw error
    } finally {
      setConfigsLoading(false)
    }
  }

  // Toggle wallet address visibility for a trader
  const toggleTraderAddressVisibility = (traderId: string) => {
    setVisibleTraderAddresses((prev) => {
      const next = new Set(prev)
      if (next.has(traderId)) {
        next.delete(traderId)
      } else {
        next.add(traderId)
      }
      return next
    })
  }

  // Copy wallet address to clipboard
  const handleCopyAddress = async (id: string, address: string) => {
    try {
      await navigator.clipboard.writeText(address)
      setCopiedId(id)
      setTimeout(() => setCopiedId(null), 2000)
    } catch (err) {
      console.error('Failed to copy address:', err)
    }
  }

  const {
    data: traders,
    mutate: mutateTraders,
    isLoading: isTradersLoading,
  } = useSWR<TraderInfo[]>(user && token ? 'traders' : null, api.getTraders, {
    refreshInterval: 5000,
  })

  useEffect(() => {
    void loadConfigs().catch(() => {})
  }, [user, token])

  const configuredModels =
    allModels?.filter((m) => {
      return (
        m.enabled ||
        m.has_api_key ||
        (m.customApiUrl && m.customApiUrl.trim() !== '')
      )
    }) || []

  const configuredExchanges =
    allExchanges?.filter((e) => {
      if (e.id === 'aster') {
        return e.asterUser && e.asterUser.trim() !== ''
      }
      if (e.id === 'hyperliquid') {
        return e.hyperliquidWalletAddr && e.hyperliquidWalletAddr.trim() !== ''
      }
      return (
        e.enabled ||
        e.has_api_key ||
        e.has_hyperliquid_secret ||
        e.has_aster_private_key ||
        e.has_lighter_private_key ||
        e.has_lighter_api_key_private_key
      )
    }) || []

  const enabledModels = selectableResources(
    allModels?.filter((m) => m.enabled) || []
  )
  const enabledExchanges = selectableResources(
    allExchanges?.filter((e) => {
      if (!e.enabled) return false
      if (e.id === 'aster') {
        return (
          e.asterUser &&
          e.asterUser.trim() !== '' &&
          e.asterSigner &&
          e.asterSigner.trim() !== ''
        )
      }
      if (e.id === 'hyperliquid') {
        return e.hyperliquidWalletAddr && e.hyperliquidWalletAddr.trim() !== ''
      }
      return true
    }) || []
  )

  const handleCreateTrader = async (data: CreateTraderRequest) => {
    try {
      const model = allModels?.find((m) => m.id === data.ai_model_id)
      const exchange = allExchanges?.find((e) => e.id === data.exchange_id)

      if (!model?.enabled) {
        toast.error(t('modelNotConfigured', language))
        return
      }

      if (!exchange?.enabled) {
        toast.error(t('exchangeNotConfigured', language))
        return
      }

      await api.createTrader(data)
      toast.success(t('aiTradersToast.created', language))
      setShowCreateModal(false)
      await mutateTraders()
    } catch (error) {
      console.error('Failed to create trader:', error)
      toast.error(t('createTraderFailed', language))
    }
  }

  const handleEditTrader = async (traderId: string) => {
    try {
      const traderConfig = await api.getTraderConfig(traderId)
      setEditingTrader(traderConfig)
      setShowEditModal(true)
    } catch (error) {
      console.error('Failed to fetch trader config:', error)
      toast.error(t('getTraderConfigFailed', language))
    }
  }

  const handleSaveEditTrader = async (data: CreateTraderRequest) => {
    if (!editingTrader) return

    try {
      const model = enabledModels?.find((m) => m.id === data.ai_model_id)
      const exchange = enabledExchanges?.find((e) => e.id === data.exchange_id)

      if (!model) {
        toast.error(t('modelConfigNotExist', language))
        return
      }

      if (!exchange) {
        toast.error(t('exchangeConfigNotExist', language))
        return
      }

      const request = {
        name: data.name,
        ai_model_id: data.ai_model_id,
        exchange_id: data.exchange_id,
        strategy_id: data.strategy_id,
        scan_interval_minutes: data.scan_interval_minutes,
        is_cross_margin: data.is_cross_margin,
        show_in_competition: data.show_in_competition,
      }

      // Running + only scan interval changed → dedicated live runtime API.
      const nextScan = data.scan_interval_minutes ?? 0
      const scanOnly =
        Boolean(editingTrader.is_running) &&
        nextScan > 0 &&
        nextScan !== editingTrader.scan_interval_minutes &&
        data.name === editingTrader.trader_name &&
        data.exchange_id === editingTrader.exchange_id &&
        (data.strategy_id || '') === (editingTrader.strategy_id || '') &&
        Boolean(data.is_cross_margin) ===
          Boolean(editingTrader.is_cross_margin) &&
        Boolean(data.show_in_competition) ===
          Boolean(editingTrader.show_in_competition)

      if (scanOnly && editingTrader.trader_id) {
        const runtime = await api.updateTraderRuntime(
          editingTrader.trader_id,
          nextScan
        )
        toast.success(
          runtime.applied_live
            ? t('trader.runtime.updated', language)
            : t('aiTradersToast.saved', language)
        )
      } else {
        await api.updateTrader(editingTrader.trader_id!, request)
        toast.success(t('aiTradersToast.saved', language))
      }
      setShowEditModal(false)
      setEditingTrader(null)
      await mutateTraders()
    } catch (error) {
      console.error('Failed to update trader:', error)
      if (error instanceof ApiError) {
        const keyed =
          error.errorKey && t(error.errorKey, language) !== error.errorKey
            ? t(error.errorKey, language)
            : ''
        toast.error(keyed || error.message || t('updateTraderFailed', language))
        return
      }
      toast.error(t('updateTraderFailed', language))
    }
  }

  const handleDeleteTrader = async (traderId: string) => {
    {
      const ok = await confirmToast(t('confirmDeleteTrader', language))
      if (!ok) return
    }

    try {
      await api.deleteTrader(traderId)
      toast.success(t('aiTradersToast.deleted', language))

      await mutateTraders()
    } catch (error) {
      console.error('Failed to delete trader:', error)
      toast.error(t('deleteTraderFailed', language))
    }
  }

  const handleToggleTrader = async (traderId: string, running: boolean) => {
    try {
      if (running) {
        await api.stopTrader(traderId)
        toast.success(t('aiTradersToast.stopped', language))
      } else {
        await api.startTrader(traderId)
        toast.success(t('aiTradersToast.started', language))
      }

      await mutateTraders()
    } catch (error) {
      // Prefer concrete server / preflight reason over a generic i18n cover.
      if (error instanceof ApiError) {
        if (error.errorKey === 'trader.start.preflight_failed') {
          const preflight = error.errorData?.preflight as
            | LaunchPreflightResult
            | undefined
          const detail =
            (preflight && describeLaunchFailures(preflight)) ||
            error.message ||
            t('trader.start.preflight_failed', language)
          toast.error(detail)
          if (preflight) {
            const target = primarySetupTarget(preflight)
            if (target === 'claw402') {
              navigate(`${ROUTES.welcome}?deposit=1`)
            } else if (target === 'hyperliquid') {
              goConfigureExchange(null, 'hyperliquid')
            } else if (target === 'exchange') {
              goConfigureExchange(primaryFailingExchangeId(preflight))
            } else if (target === 'hyperliquid-funds') {
              const next = new URLSearchParams(searchParams)
              next.set('setup', 'hyperliquid-funds')
              next.set('view', 'autopilot')
              setSearchParams(next)
            }
          }
          return
        }
        const keyed =
          error.errorKey && t(error.errorKey, language) !== error.errorKey
            ? t(error.errorKey, language)
            : ''
        toast.error(keyed || error.message || t('operationFailed', language))
        return
      }
      if (error instanceof Error && error.message) {
        toast.error(error.message)
        return
      }
      toast.error(t('operationFailed', language))
    }
  }

  const handleToggleCompetition = async (
    traderId: string,
    currentShowInCompetition: boolean
  ) => {
    try {
      const newValue = !currentShowInCompetition
      await api.toggleCompetition(traderId, newValue)
      toast.success(
        newValue
          ? t('aiTradersToast.showInCompetition', language)
          : t('aiTradersToast.hideInCompetition', language)
      )

      await mutateTraders()
    } catch (error) {
      console.error('Failed to toggle competition visibility:', error)
      toast.error(t('operationFailed', language))
    }
  }

  useEffect(() => {
    if (!user || !token) return

    const setupTarget = searchParams.get('setup')
    if (!setupTarget) return

    if (setupTarget === 'claw402') {
      navigate(`${ROUTES.welcome}?deposit=1`)
    } else if (setupTarget === 'hyperliquid') {
      const existing = allExchanges?.find(
        (exchange) =>
          exchange.exchange_type === 'hyperliquid' ||
          exchange.id === 'hyperliquid'
      )
      goConfigureExchange(existing?.id || null, existing ? null : 'hyperliquid')
    } else if (setupTarget === 'exchange') {
      goConfigureExchange(searchParams.get('exchange_id'))
    } else if (setupTarget === 'hyperliquid-funds') {
      setShowLaunch(true)
      document
        .getElementById('autopilot-launch-panel')
        ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      toast.info(
        'Deposit USDC to your Hyperliquid account, the balance check updates automatically.'
      )
    } else {
      return
    }

    const nextParams = new URLSearchParams(searchParams)
    if (setupTarget === 'hyperliquid-funds') nextParams.set('view', 'autopilot')
    nextParams.delete('setup')
    nextParams.delete('exchange_id')
    setSearchParams(nextParams, { replace: true })
  }, [allExchanges, navigate, searchParams, setSearchParams, token, user])

  const refreshLaunchState = async () => {
    await Promise.all([loadConfigs(), mutateTraders()])
  }

  const hasRunningTrader = (traders || []).some((trader) => trader.is_running)
  const launchPanel = (
    <AutopilotLaunchPanel
      models={allModels}
      exchanges={allExchanges}
      exchangeAccountStates={exchangeAccountStates}
      traders={traders || []}
      isLoggedIn={Boolean(user && token)}
      language={language}
      onRefresh={refreshLaunchState}
      onOpenClaw402Config={() => navigate(`${ROUTES.welcome}?deposit=1`)}
      onOpenHyperliquidConfig={() => {
        const existing = allExchanges?.find(
          (exchange) =>
            exchange.exchange_type === 'hyperliquid' ||
            exchange.id === 'hyperliquid'
        )
        goConfigureExchange(existing?.id || null, existing ? null : 'hyperliquid')
      }}
      onOpenExchangeConfig={(exchangeId) => goConfigureExchange(exchangeId)}
    />
  )

  return (
    <div className="config-page">
      <header className="config-page-heading">
        <div>
          <span className="config-eyebrow">
            {t('pageAutopilotEyebrow', language)}
          </span>
          <h1>{t('pageAutopilotTitle', language)}</h1>
          <p>{t('pageAutopilotDesc', language)}</p>
        </div>
        <button
          className="config-primary"
          onClick={() => navigate(ROUTES.portfolio)}
        >
          {t('pageAutopilotCta', language)} <ArrowUpRight size={15} />
        </button>
      </header>
      <div className="config-layout">
        <aside className="config-sidebar">
          <span className="config-nav-label">SETUP</span>
          <button
            aria-current={configView === 'connections' ? 'page' : undefined}
            onClick={() => changeConfigView('connections')}
          >
            <Cable size={17} />
            <span>
              {t('pageAutopilotConnections', language)}
              <small>{t('pageAutopilotConnectionsDesc', language)}</small>
            </span>
          </button>
          <button
            aria-current={configView === 'autopilot' ? 'page' : undefined}
            onClick={() => changeConfigView('autopilot')}
          >
            <Bot size={17} />
            <span>
              {t('pageAutopilotTraders', language)}
              <small>{t('pageAutopilotTradersDesc', language)}</small>
            </span>
            <b>{traders?.length || 0}</b>
          </button>
          <span className="config-nav-label">PREFERENCES</span>
          <button onClick={goResources}>
            <Settings2 size={17} />
            <span>
              {t('resourcesTab', language)}
              <small>{t('pageAutopilotSettingsHint', language)}</small>
            </span>
          </button>
          <div className="config-sidebar-note">
            <strong>{t('pageAutopilotNoteTitle', language)}</strong>
            <p>{t('pageAutopilotNoteBody', language)}</p>
          </div>
        </aside>
        <div className="config-main">
          <div className="config-content-heading">
            <div>
              <h2>
                {configView === 'connections'
                  ? t('pageAutopilotConnections', language)
                  : t('pageAutopilotTraders', language)}
              </h2>
              <p>
                {configView === 'connections'
                  ? t('pageAutopilotConnectionsDesc', language)
                  : t('pageAutopilotTradersDesc', language)}
              </p>
            </div>
            <button
              className="config-refresh"
              disabled={configsLoading}
              onClick={() => {
                void refreshLaunchState().catch(() => {})
              }}
            >
              <RefreshCw size={14} />{' '}
              {configsLoading ? 'Refreshing…' : 'Refresh'}
            </button>
          </div>
          {configError && (
            <p className="config-load-error" role="alert">
              {configError}
            </p>
          )}
          {configView === 'connections' && (
            <>
              <section className="config-resources-hub">
                <div>
                  <span className="config-eyebrow">RESOURCES</span>
                  <h3>{t('resourcesTitle', language)}</h3>
                  <p>
                    {configsLoading
                      ? 'Checking saved connections…'
                      : `${configuredModels.length} AI · ${configuredExchanges.length} exchange · configure, test, and choose defaults in Settings → Resources.`}
                  </p>
                  {configError && (
                    <p className="config-load-error" role="alert">
                      {configError}
                    </p>
                  )}
                </div>
                <div className="config-resources-actions">
                  <button className="config-primary" onClick={goResources}>
                    {t('resourcesTab', language)} <ArrowUpRight size={15} />
                  </button>
                  <button onClick={() => goConfigureModel()}>
                    {t('settingsAddModel', language)}
                  </button>
                  <button onClick={() => goConfigureExchange()}>
                    {t('settingsAddExchange', language)}
                  </button>
                </div>
              </section>

              <section className="config-next">
                <span className="config-eyebrow">WHAT’S NEXT?</span>
                <h3>From connections to your first trade.</h3>
                <div>
                  <button onClick={() => navigate(ROUTES.planning)}>
                    <span>
                      <strong>Open Trade</strong>
                      <small>
                        Build plans with AI. Review and approve each trade
                        yourself.
                      </small>
                    </span>
                    <ArrowUpRight size={18} />
                  </button>
                  <button onClick={() => changeConfigView('autopilot')}>
                    <span>
                      <strong>Set up Autopilot</strong>
                      <small>
                        Choose a strategy and explicitly start an automated
                        trader.
                      </small>
                    </span>
                    <ArrowUpRight size={18} />
                  </button>
                </div>
              </section>
            </>
          )}
          {configView === 'autopilot' && (
            <>
              <div className="config-autopilot-actions">
                <div>
                  <strong>
                    {isTradersLoading
                      ? 'Checking Autopilot status…'
                      : hasRunningTrader
                        ? 'Autopilot is running'
                        : 'No Autopilot is running'}
                  </strong>
                  <p>
                    {configsLoading
                      ? 'Checking saved connections…'
                      : `${configuredModels.length} AI connection${configuredModels.length === 1 ? '' : 's'} · ${configuredExchanges.length} trading account${configuredExchanges.length === 1 ? '' : 's'}`}
                  </p>
                </div>
                <button onClick={() => setShowLaunch((v) => !v)}>
                  {showLaunch ? 'Hide guided setup' : 'Guided setup'}
                </button>
                <button
                  className="config-primary"
                  onClick={() => setShowCreateModal(true)}
                  disabled={
                    configsLoading ||
                    !configuredModels.length ||
                    !configuredExchanges.length
                  }
                >
                  <Plus size={14} /> Create Autopilot
                </button>
              </div>
              {showLaunch && launchPanel}
              {/* Traders List */}
              <TradersList
                traders={traders}
                isLoading={isTradersLoading}
                allExchanges={allExchanges}
                configuredModelsCount={configuredModels.length}
                configuredExchangesCount={configuredExchanges.length}
                visibleTraderAddresses={visibleTraderAddresses}
                copiedId={copiedId}
                language={language}
                onTraderSelect={onTraderSelect}
                onNavigate={(path: any) => navigate(path)}
                onEditTrader={handleEditTrader}
                onToggleTrader={handleToggleTrader}
                onToggleCompetition={handleToggleCompetition}
                onDeleteTrader={handleDeleteTrader}
                onToggleTraderAddress={toggleTraderAddressVisibility}
                onCopyAddress={handleCopyAddress}
              />
            </>
          )}
        </div>
      </div>
      {/* Create Trader Modal */}
      {showCreateModal && (
        <TraderConfigModal
          isOpen={showCreateModal}
          isEditMode={false}
          availableModels={enabledModels}
          availableExchanges={enabledExchanges}
          onSave={handleCreateTrader}
          onClose={() => setShowCreateModal(false)}
        />
      )}

      {/* Edit Trader Modal */}
      {showEditModal && editingTrader && (
        <TraderConfigModal
          isOpen={showEditModal}
          isEditMode={true}
          traderData={editingTrader}
          availableModels={enabledModels}
          availableExchanges={enabledExchanges}
          onSave={handleSaveEditTrader}
          onClose={() => {
            setShowEditModal(false)
            setEditingTrader(null)
          }}
        />
      )}

    </div>
  )
}
