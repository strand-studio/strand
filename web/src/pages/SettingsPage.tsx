import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router'
import { toast } from 'sonner'
import {
  Activity,
  Download,
  Eye,
  EyeOff,
  Upload,
  User,
} from 'lucide-react'
import { ResourcesHubPanel } from '../components/settings/ResourcesHubPanel'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import { useTheme, type ThemeMode } from '../contexts/ThemeContext'
import { t, type Language } from '../i18n/translations'
import { api } from '../lib/api'

type Tab = 'account' | 'resources'

function resolveTab(raw: string | null): Tab {
  // Legacy models / exchanges / channels → Resources hub
  if (
    raw === 'resources' ||
    raw === 'models' ||
    raw === 'exchanges' ||
    raw === 'channels' ||
    raw === 'configure'
  ) {
    return 'resources'
  }
  return 'account'
}

export function SettingsPage() {
  const { user } = useAuth()
  const { language, setLanguage } = useLanguage()
  const { theme, setTheme } = useTheme()
  const [searchParams, setSearchParams] = useSearchParams()
  const [activeTab, setActiveTab] = useState<Tab>(() =>
    resolveTab(searchParams.get('tab'))
  )

  const [newPassword, setNewPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [changingPassword, setChangingPassword] = useState(false)
  const [backupBusy, setBackupBusy] = useState(false)

  useEffect(() => {
    const tab = searchParams.get('tab')
    const next = resolveTab(tab)
    setActiveTab(next)
    if (tab === 'models' || tab === 'exchanges' || tab === 'channels') {
      const cleaned = new URLSearchParams(searchParams)
      cleaned.set('tab', 'resources')
      setSearchParams(cleaned, { replace: true })
    }
  }, [searchParams, setSearchParams])

  const changeTab = (tab: Tab) => {
    setActiveTab(tab)
    const next = new URLSearchParams(searchParams)
    if (tab === 'account') next.delete('tab')
    else next.set('tab', tab)
    next.delete('configure')
    next.delete('id')
    next.delete('type')
    setSearchParams(next, { replace: true })
  }

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

  const tabs: { key: Tab; label: string; icon: React.ReactNode }[] = [
    {
      key: 'account',
      label: t('settingsTabAccount', language),
      icon: <User size={16} />,
    },
    {
      key: 'resources',
      label: t('resourcesTab', language),
      icon: <Activity size={16} />,
    },
  ]

  return (
    <div className="strand-settings min-h-screen bg-strand-bg pt-20 pb-12 px-4">
      <div
        className={`mx-auto ${activeTab === 'resources' ? 'max-w-3xl' : 'max-w-2xl'}`}
      >
        <h1 className="text-xl font-bold text-strand-text mb-6">
          {t('settingsTitle', language)}
        </h1>

        <div className="flex gap-1 mb-6 bg-strand-bg-lighter border border-strand-border rounded-xl p-1">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => changeTab(tab.key)}
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

        <div className="bg-strand-bg-lighter backdrop-blur-xl border border-strand-border rounded-2xl p-6">
          {activeTab === 'resources' && <ResourcesHubPanel />}

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
        </div>
      </div>
    </div>
  )
}
