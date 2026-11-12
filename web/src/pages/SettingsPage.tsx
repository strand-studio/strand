import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router'
import { toast } from 'sonner'
import {
  User,
  Cpu,
  Building2,
  Eye,
  EyeOff,
  ChevronRight,
  Plus,
  Pencil,
  Star,
  Download,
  Upload,
} from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import { useTheme, type ThemeMode } from '../contexts/ThemeContext'
import { t, type Language } from '../i18n/translations'
import { api } from '../lib/api'
import {
  getPreferredModelId,
  setPreferredModelId,
} from '../lib/preferredModel'
import { ExchangeConfigModal } from '../components/trader/ExchangeConfigModal'
import { ModelConfigModal } from '../components/trader/ModelConfigModal'
import type { Exchange, AIModel, ExchangeAccountState } from '../types'

type Tab = 'account' | 'models' | 'exchanges'

function configBadge(label: string, active: boolean) {
  return (
    <span
      className={`text-[11px] px-2 py-0.5 rounded-full ${
        active
          ? 'bg-strand-success/10 text-strand-success'
          : 'bg-strand-bg-deeper text-strand-text-muted'
      }`}
    >
      {label}
    </span>
  )
}

export function SettingsPage() {
  const { user } = useAuth()
  const { language, setLanguage } = useLanguage()
  const { theme, setTheme } = useTheme()
  const [searchParams] = useSearchParams()
  const [activeTab, setActiveTab] = useState<Tab>(() => {
    const tab = searchParams.get('tab')
    return tab === 'models' || tab === 'exchanges' ? tab : 'account'
  })

  // Account state
  const [newPassword, setNewPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [changingPassword, setChangingPassword] = useState(false)

  // AI Models state
  const [configuredModels, setConfiguredModels] = useState<AIModel[]>([])
  const [supportedModels, setSupportedModels] = useState<AIModel[]>([])
  const [showModelModal, setShowModelModal] = useState(false)
  const [editingModel, setEditingModel] = useState<string | null>(null)

  // Exchanges state
  const [exchanges, setExchanges] = useState<Exchange[]>([])
  const [exchangeStates, setExchangeStates] = useState<
    Record<string, ExchangeAccountState>
  >({})
  const [exchangeStatesLoading, setExchangeStatesLoading] = useState(false)
  const [showExchangeModal, setShowExchangeModal] = useState(false)
  const [editingExchange, setEditingExchange] = useState<string | null>(null)
  const [preferredModelId, setPreferredModelIdState] = useState(() =>
    getPreferredModelId()
  )
  const [backupBusy, setBackupBusy] = useState(false)

  const refreshModelConfigs = async () => {
    const [configs, supported] = await Promise.all([
      api.getModelConfigs(),
      api.getSupportedModels(),
    ])
    setConfiguredModels(configs)
    setSupportedModels(supported)
  }

  const refreshExchangeConfigs = async () => {
    const [refreshed, accountStateResponse] = await Promise.all([
      api.getExchangeConfigs(),
      api.getExchangeAccountState().catch(() => ({ states: {} })),
    ])
    setExchanges(refreshed)
    setExchangeStates(accountStateResponse.states || {})
  }

  const refreshExchangeStates = async () => {
    setExchangeStatesLoading(true)
    try {
      const response = await api.getExchangeAccountState()
      setExchangeStates(response.states || {})
    } catch {
      toast.error(t('loadExchangeBalancesFailed', language))
    } finally {
      setExchangeStatesLoading(false)
    }
  }

  // Fetch data when tabs are visited
  useEffect(() => {
    if (activeTab === 'models') {
      refreshModelConfigs().catch(() =>
        toast.error(t('loadAIModelsFailed', language))
      )
    }
    if (activeTab === 'exchanges') {
      refreshExchangeConfigs().catch(() =>
        toast.error(t('loadExchangesFailed', language))
      )
    }
  }, [activeTab])

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword.length < 8) {
      toast.error(t('settingsPasswordTooShort', language))
      return
    }
    setChangingPassword(true)
    try {
      const res = await fetch('/api/user/password', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`,
        },
        body: JSON.stringify({ new_password: newPassword }),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => ({}))
        throw new Error(
          data.error || t('settingsPasswordUpdateFailed', language)
        )
      }
      toast.success(t('settingsPasswordUpdated', language))
      setNewPassword('')
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : t('settingsPasswordUpdateFailed', language)
      )
    } finally {
      setChangingPassword(false)
    }
  }

  const handleSaveModel = async (
    modelId: string,
    apiKey: string,
    customApiUrl?: string,
    customModelName?: string
  ) => {
    try {
      const existingModel = configuredModels.find((m) => m.id === modelId)
      const modelTemplate = supportedModels.find((m) => m.id === modelId)
      const modelToUpdate = existingModel || modelTemplate
      if (!modelToUpdate) {
        toast.error(t('settingsModelNotFound', language))
        return
      }

      let updatedModels: AIModel[]
      if (existingModel) {
        updatedModels = configuredModels.map((m) =>
          m.id === modelId
            ? {
                ...m,
                apiKey,
                customApiUrl: customApiUrl || '',
                customModelName: customModelName || '',
                enabled: true,
              }
            : m
        )
      } else {
        updatedModels = [
          ...configuredModels,
          {
            ...modelToUpdate,
            apiKey,
            customApiUrl: customApiUrl || '',
            customModelName: customModelName || '',
            enabled: true,
          },
        ]
      }

      const request = {
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
      }
      await api.updateModelConfigs(request)
      toast.success(t('settingsModelSaved', language))
      await refreshModelConfigs()
      setShowModelModal(false)
      setEditingModel(null)
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('settingsModelSaveFailed', language)
      )
      throw error instanceof Error
        ? error
        : new Error('Failed to save model config')
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
      const request = {
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
      }
      await api.updateModelConfigs(request)
      await refreshModelConfigs()
      setShowModelModal(false)
      setEditingModel(null)
      toast.success(t('settingsModelRemoved', language))
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
      if (exchangeType === 'hyperliquid') {
        toast.error(t('settingsHyperliquidWalletOnly', language))
        return
      }
      if (exchangeId) {
        const request = {
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
        }
        await api.updateExchangeConfigsEncrypted(request)
        toast.success(t('settingsExchangeUpdated', language))
      } else {
        const createRequest = {
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
        }
        await api.createExchangeEncrypted(createRequest)
        toast.success(t('settingsExchangeCreated', language))
      }
      await refreshExchangeConfigs()
      setShowExchangeModal(false)
      setEditingExchange(null)
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
      await refreshExchangeConfigs()
      setShowExchangeModal(false)
      setEditingExchange(null)
    } catch {
      toast.error(t('settingsExchangeDeleteFailed', language))
    }
  }

  const tabs: { key: Tab; label: string; icon: React.ReactNode }[] = [
    {
      key: 'account',
      label: t('settingsTabAccount', language),
      icon: <User size={16} />,
    },
    {
      key: 'models',
      label: t('aiModels', language),
      icon: <Cpu size={16} />,
    },
    {
      key: 'exchanges',
      label: t('exchanges', language),
      icon: <Building2 size={16} />,
    },
  ]

  return (
    <div className="strand-settings min-h-screen bg-strand-bg pt-20 pb-12 px-4">
      <div className="max-w-2xl mx-auto">
        <h1 className="text-xl font-bold text-strand-text mb-6">
          {t('settingsTitle', language)}
        </h1>

        {/* Tabs */}
        <div className="flex gap-1 mb-6 bg-strand-bg-lighter border border-[rgba(25,36,52,0.14)] rounded-xl p-1">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`flex-1 flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all
                ${
                  activeTab === tab.key
                    ? 'bg-strand-gold text-strand-bg'
                    : 'text-strand-text-muted hover:text-strand-text'
                }`}
            >
              {tab.icon}
              <span className="hidden sm:inline">{tab.label}</span>
            </button>
          ))}
        </div>

        {/* Tab Content */}
        <div className="bg-strand-bg-lighter backdrop-blur-xl border border-[rgba(25,36,52,0.14)] rounded-2xl p-6">
          {/* Account Tab */}
          {activeTab === 'account' && (
            <div className="space-y-6">
              <div>
                <p className="text-xs text-strand-text-muted mb-1">
                  {t('email', language)}
                </p>
                <p className="text-sm text-strand-text font-medium">
                  {user?.email}
                </p>
              </div>

              <div className="border-t border-strand-border/40 pt-6 space-y-4">
                <h3 className="text-sm font-semibold text-strand-text">
                  {t('preferences', language)}
                </h3>
                <div className="grid gap-4 sm:grid-cols-2">
                  <label className="block space-y-1.5">
                    <span className="text-xs font-medium text-strand-text-muted">
                      {t('language', language)}
                    </span>
                    <select
                      value={language}
                      onChange={(e) =>
                        setLanguage(e.target.value as Language)
                      }
                      className="w-full rounded-xl border border-strand-border/40 bg-strand-bg px-3 py-2.5 text-sm text-strand-text"
                    >
                      <option value="en">English</option>
                      <option value="zh">中文</option>
                      <option value="id">Bahasa Indonesia</option>
                    </select>
                  </label>
                  <label className="block space-y-1.5">
                    <span className="text-xs font-medium text-strand-text-muted">
                      {t('theme', language)}
                    </span>
                    <select
                      value={theme}
                      onChange={(e) => setTheme(e.target.value as ThemeMode)}
                      className="w-full rounded-xl border border-strand-border/40 bg-strand-bg px-3 py-2.5 text-sm text-strand-text"
                    >
                      <option value="system">{t('themeSystem', language)}</option>
                      <option value="light">{t('themeLight', language)}</option>
                      <option value="dark">{t('themeDark', language)}</option>
                    </select>
                  </label>
                </div>
              </div>

              <div className="border-t border-strand-border/40 pt-6 space-y-3">
                <h3 className="text-sm font-semibold text-strand-text">
                  {t('backupTitle', language)}
                </h3>
                <p className="text-xs text-strand-text-muted">
                  {t('backupDesc', language)}
                </p>
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    disabled={backupBusy}
                    onClick={async () => {
                      setBackupBusy(true)
                      try {
                        const data = await api.exportBackup(false)
                        const blob = new Blob(
                          [JSON.stringify(data, null, 2)],
                          { type: 'application/json' }
                        )
                        const url = URL.createObjectURL(blob)
                        const a = document.createElement('a')
                        a.href = url
                        a.download = `strand-backup-${new Date().toISOString().slice(0, 10)}.json`
                        a.click()
                        URL.revokeObjectURL(url)
                        toast.success(t('backupExported', language))
                      } catch (err) {
                        toast.error(
                          err instanceof Error
                            ? err.message
                            : t('backupExportFailed', language)
                        )
                      } finally {
                        setBackupBusy(false)
                      }
                    }}
                    className="inline-flex items-center gap-1.5 text-xs font-medium bg-strand-bg-deeper hover:bg-strand-bg px-3 py-2 rounded-lg"
                  >
                    <Download size={14} />
                    {t('backupExportNoSecrets', language)}
                  </button>
                  <button
                    type="button"
                    disabled={backupBusy}
                    onClick={async () => {
                      const ok = window.confirm(
                        t('backupSecretsConfirm', language)
                      )
                      if (!ok) return
                      setBackupBusy(true)
                      try {
                        const data = await api.exportBackup(true)
                        const blob = new Blob(
                          [JSON.stringify(data, null, 2)],
                          { type: 'application/json' }
                        )
                        const url = URL.createObjectURL(blob)
                        const a = document.createElement('a')
                        a.href = url
                        a.download = `strand-backup-secrets-${new Date().toISOString().slice(0, 10)}.json`
                        a.click()
                        URL.revokeObjectURL(url)
                        toast.success(t('backupSecretsExported', language))
                      } catch (err) {
                        toast.error(
                          err instanceof Error
                            ? err.message
                            : t('backupExportFailed', language)
                        )
                      } finally {
                        setBackupBusy(false)
                      }
                    }}
                    className="inline-flex items-center gap-1.5 text-xs font-medium bg-strand-danger/10 text-strand-danger hover:bg-strand-danger/20 px-3 py-2 rounded-lg"
                  >
                    <Download size={14} />
                    {t('backupExportWithSecrets', language)}
                  </button>
                  <label className="inline-flex items-center gap-1.5 text-xs font-medium bg-strand-gold/10 hover:bg-strand-gold/20 text-strand-gold px-3 py-2 rounded-lg cursor-pointer">
                    <Upload size={14} />
                    {t('backupImport', language)}
                    <input
                      type="file"
                      accept="application/json,.json"
                      className="hidden"
                      onChange={async (e) => {
                        const file = e.target.files?.[0]
                        e.target.value = ''
                        if (!file) return
                        setBackupBusy(true)
                        try {
                          const text = await file.text()
                          const parsed = JSON.parse(text)
                          const restoreSecrets = Boolean(
                            parsed.include_secrets &&
                              window.confirm(
                                t('backupRestoreSecretsConfirm', language)
                              )
                          )
                          const result = await api.importBackup({
                            ...parsed,
                            restore_secrets: restoreSecrets,
                          })
                          toast.success(
                            t('backupRestored', language, {
                              strategies: result.restored_strategies || 0,
                              models: result.restored_models || 0,
                              exchanges: result.restored_exchanges || 0,
                            })
                          )
                        } catch (err) {
                          toast.error(
                            err instanceof Error
                              ? err.message
                              : t('backupImportFailed', language)
                          )
                        } finally {
                          setBackupBusy(false)
                        }
                      }}
                    />
                  </label>
                </div>
              </div>

              <div className="border-t border-strand-border/40 pt-6">
                <h3 className="text-sm font-semibold text-strand-text mb-4">
                  {t('changePassword', language)}
                </h3>
                <form onSubmit={handleChangePassword} className="space-y-4">
                  <div>
                    <label className="block text-xs font-medium text-strand-text-muted mb-2">
                      {t('newPassword', language)}
                    </label>
                    <div className="relative">
                      <input
                        type={showPassword ? 'text' : 'password'}
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        className="w-full bg-strand-bg-deeper border border-strand-border/40 rounded-xl px-4 py-3 pr-11 text-sm text-strand-text placeholder-strand-text-muted focus:outline-hidden focus:border-strand-gold/60 focus:ring-1 focus:ring-strand-gold/30 transition-all"
                        placeholder={t('passwordMinChars', language)}
                        required
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword(!showPassword)}
                        className="absolute right-3.5 top-1/2 -translate-y-1/2 text-strand-text-muted hover:text-strand-text transition-colors"
                      >
                        {showPassword ? (
                          <EyeOff size={16} />
                        ) : (
                          <Eye size={16} />
                        )}
                      </button>
                    </div>
                  </div>
                  <button
                    type="submit"
                    disabled={changingPassword || newPassword.length < 8}
                    className="w-full bg-strand-gold hover:bg-strand-gold-highlight active:scale-[0.98] text-strand-bg font-semibold py-3 rounded-xl text-sm transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {changingPassword
                      ? t('updatingPassword', language)
                      : t('updatePassword', language)}
                  </button>
                </form>
              </div>
            </div>
          )}

          {/* AI Models Tab */}
          {activeTab === 'models' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <p className="text-sm text-strand-text-muted">
                  {t('settingsModelsCount', language, {
                    count: configuredModels.length,
                  })}
                </p>
                <button
                  onClick={() => {
                    setEditingModel(null)
                    setShowModelModal(true)
                  }}
                  className="flex items-center gap-1.5 text-xs font-medium bg-strand-gold/10 hover:bg-strand-gold/20 text-strand-gold px-3 py-1.5 rounded-lg transition-colors"
                >
                  <Plus size={14} />
                  {t('settingsAddModel', language)}
                </button>
              </div>

              {configuredModels.length === 0 ? (
                <div className="text-center py-8 text-strand-text-muted text-sm">
                  {t('settingsNoModelsYet', language)}
                </div>
              ) : (
                <div className="space-y-2">
                  {configuredModels.map((model) => (
                    <div
                      key={model.id}
                      className="w-full flex items-center justify-between px-4 py-3 rounded-xl bg-strand-bg-deeper border border-[rgba(25,36,52,0.14)] group"
                    >
                      <button
                        type="button"
                        onClick={() => {
                          setEditingModel(model.id)
                          setShowModelModal(true)
                        }}
                        className="flex items-center gap-3 flex-1 text-left min-w-0"
                      >
                        <div className="w-8 h-8 rounded-lg bg-strand-bg-deeper flex items-center justify-center">
                          <Cpu size={14} className="text-strand-text" />
                        </div>
                        <div className="text-left min-w-0">
                          <p className="text-sm font-medium text-strand-text">
                            {model.name}
                          </p>
                          <div className="flex flex-wrap items-center gap-1.5 mt-1">
                            <p className="text-xs text-strand-text-muted">
                              {model.provider}
                            </p>
                            {configBadge(
                              t('settingsBadgeApiKey', language),
                              !!model.has_api_key
                            )}
                            {model.customModelName
                              ? configBadge(
                                  t('settingsBadgeCustomModel', language),
                                  true
                                )
                              : null}
                            {model.customApiUrl
                              ? configBadge(
                                  t('settingsBadgeBaseUrl', language),
                                  true
                                )
                              : null}
                            {preferredModelId === model.id
                              ? configBadge(
                                  t('settingsBadgeDefault', language),
                                  true
                                )
                              : null}
                          </div>
                        </div>
                      </button>
                      <div className="flex items-center gap-2 shrink-0">
                        <button
                          type="button"
                          title={t('settingsSetPreferred', language)}
                          onClick={() => {
                            const next =
                              preferredModelId === model.id ? '' : model.id
                            setPreferredModelId(next)
                            setPreferredModelIdState(next)
                            toast.success(
                              next
                                ? t('settingsPreferredUpdated', language)
                                : t('settingsPreferredCleared', language)
                            )
                          }}
                          className="p-1.5 rounded-lg hover:bg-strand-bg"
                        >
                          <Star
                            size={14}
                            className={
                              preferredModelId === model.id
                                ? 'text-strand-gold fill-strand-gold'
                                : 'text-strand-text-muted'
                            }
                          />
                        </button>
                        <span
                          className={`text-xs px-2 py-0.5 rounded-full ${model.enabled ? 'bg-strand-success/10 text-strand-success' : 'bg-strand-bg-deeper text-strand-text-muted'}`}
                        >
                          {model.enabled
                            ? t('settingsActive', language)
                            : t('settingsInactive', language)}
                        </span>
                        <button
                          type="button"
                          onClick={() => {
                            setEditingModel(model.id)
                            setShowModelModal(true)
                          }}
                        >
                          <Pencil
                            size={14}
                            className="text-strand-text-muted"
                          />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Exchanges Tab */}
          {activeTab === 'exchanges' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <p className="text-sm text-strand-text-muted">
                  {t('settingsAccountsCount', language, {
                    count: exchanges.length,
                  })}
                </p>
                <div className="flex items-center gap-2">
                  <button
                    onClick={refreshExchangeStates}
                    disabled={exchangeStatesLoading}
                    className="text-xs font-medium bg-strand-bg-deeper hover:bg-strand-bg-deeper disabled:opacity-60 text-strand-text px-3 py-1.5 rounded-lg transition-colors"
                  >
                    {exchangeStatesLoading
                      ? t('settingsRefreshing', language)
                      : t('settingsRefreshBalances', language)}
                  </button>
                  <button
                    onClick={() => {
                      setEditingExchange(null)
                      setShowExchangeModal(true)
                    }}
                    className="flex items-center gap-1.5 text-xs font-medium bg-strand-gold/10 hover:bg-strand-gold/20 text-strand-gold px-3 py-1.5 rounded-lg transition-colors"
                  >
                    <Plus size={14} />
                    {t('settingsAddExchange', language)}
                  </button>
                </div>
              </div>

              {exchanges.length === 0 ? (
                <div className="text-center py-8 text-strand-text-muted text-sm">
                  {t('settingsNoExchangesYet', language)}
                </div>
              ) : (
                <div className="space-y-2">
                  {exchanges.map((exchange) => {
                    const accountState = exchangeStates[exchange.id]
                    return (
                      <button
                        key={exchange.id}
                        onClick={() => {
                          setEditingExchange(exchange.id)
                          setShowExchangeModal(true)
                        }}
                        className="w-full flex items-center justify-between px-4 py-3 rounded-xl bg-strand-bg-deeper hover:bg-strand-bg-deeper border border-[rgba(25,36,52,0.14)] transition-colors group"
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-lg bg-strand-bg-deeper flex items-center justify-center">
                            <Building2 size={14} className="text-strand-text" />
                          </div>
                          <div className="text-left">
                            <p className="text-sm font-medium text-strand-text">
                              {exchange.account_name || exchange.name}
                            </p>
                            <div className="flex flex-wrap items-center gap-1.5 mt-1">
                              <p className="text-xs text-strand-text-muted capitalize">
                                {exchange.exchange_type || exchange.type}
                              </p>
                              {configBadge(
                                t('settingsBadgeApiKey', language),
                                !!exchange.has_api_key
                              )}
                              {configBadge(
                                t('settingsBadgeSecret', language),
                                !!exchange.has_secret_key
                              )}
                              {exchange.has_passphrase
                                ? configBadge(
                                    t('settingsBadgePassphrase', language),
                                    true
                                  )
                                : null}
                              {exchange.hyperliquidWalletAddr
                                ? configBadge(
                                    t('settingsBadgeWallet', language),
                                    true
                                  )
                                : null}
                              {exchange.has_aster_private_key
                                ? configBadge('Aster Key', true)
                                : null}
                              {exchange.has_lighter_private_key ||
                              exchange.has_lighter_api_key_private_key
                                ? configBadge('Lighter Key', true)
                                : null}
                            </div>
                            {accountState && (
                              <div className="flex flex-wrap items-center gap-2 mt-2 text-xs">
                                {accountState.status === 'ok' ? (
                                  <>
                                    <span className="rounded-full bg-strand-success/10 px-2 py-0.5 font-mono text-strand-success">
                                      {t('settingsBalance', language)}{' '}
                                      {accountState.display_balance ||
                                        `${accountState.total_equity?.toFixed(2) ?? '--'} ${accountState.asset || ''}`}
                                    </span>
                                    {typeof accountState.available_balance ===
                                      'number' && (
                                      <span className="text-strand-text-muted">
                                        {t('settingsAvailable', language)}{' '}
                                        {accountState.available_balance.toFixed(
                                          2
                                        )}{' '}
                                        {accountState.asset || ''}
                                      </span>
                                    )}
                                  </>
                                ) : (
                                  <span className="rounded-full bg-strand-gold/10 px-2 py-0.5 text-strand-gold">
                                    {t('settingsBalanceUnavailable', language)}
                                    :{' '}
                                    {accountState.error_message ||
                                      accountState.status}
                                  </span>
                                )}
                              </div>
                            )}
                          </div>
                        </div>
                        <ChevronRight
                          size={14}
                          className="text-strand-text-muted group-hover:text-strand-text-muted transition-colors"
                        />
                      </button>
                    )
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* AI Model Modal */}
      {showModelModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-xs px-4">
          <ModelConfigModal
            allModels={supportedModels}
            configuredModels={configuredModels}
            editingModelId={editingModel}
            onSave={handleSaveModel}
            onDelete={handleDeleteModel}
            onClose={() => {
              setShowModelModal(false)
              setEditingModel(null)
            }}
            language={language}
          />
        </div>
      )}

      {/* Exchange Modal */}
      {showExchangeModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-xs px-4">
          <ExchangeConfigModal
            allExchanges={exchanges}
            editingExchangeId={editingExchange}
            onSave={handleSaveExchange}
            onDelete={handleDeleteExchange}
            onClose={() => {
              setShowExchangeModal(false)
              setEditingExchange(null)
            }}
            language={language}
          />
        </div>
      )}
    </div>
  )
}
