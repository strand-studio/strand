import React, { useState, useEffect } from 'react'
import type { Exchange } from '../../types'
import { t, type Language } from '../../i18n/translations'
import { api } from '../../lib/api'
import { useAuth } from '../../contexts/AuthContext'
import { HyperliquidWalletConnect } from '../common/HyperliquidWalletConnect'
import { HyperliquidFundsPanel } from '../common/HyperliquidFundsPanel'
import { getExchangeIcon } from '../common/ExchangeIcons'
import {
  TwoStageKeyModal,
  type TwoStageKeyModalResult,
} from '../modals/TwoStageKeyModal'
import {
  WebCryptoEnvironmentCheck,
  type WebCryptoCheckStatus,
} from '../common/WebCryptoEnvironmentCheck'
import {
  BookOpen,
  Trash2,
  HelpCircle,
  ExternalLink,
  UserPlus,
  Key,
  Shield,
  ChevronLeft,
  Check,
  Copy,
  ArrowRight,
} from 'lucide-react'
import { toast } from 'sonner'
import { Tooltip } from './Tooltip'
import { getShortName } from './utils'

// Supported exchange templates
const SUPPORTED_EXCHANGE_TEMPLATES = [
  { exchange_type: 'binance', name: 'Binance Futures', type: 'cex' as const },
  { exchange_type: 'okx', name: 'OKX Futures', type: 'cex' as const },
  { exchange_type: 'bitget', name: 'Bitget Futures', type: 'cex' as const },
  { exchange_type: 'gate', name: 'Gate.io Futures', type: 'cex' as const },
  { exchange_type: 'bybit', name: 'Bybit Futures', type: 'cex' as const },
  { exchange_type: 'kucoin', name: 'KuCoin Futures', type: 'cex' as const },
  { exchange_type: 'hyperliquid', name: 'Hyperliquid', type: 'dex' as const },
  { exchange_type: 'aster', name: 'Aster DEX', type: 'dex' as const },
  { exchange_type: 'lighter', name: 'Lighter', type: 'dex' as const },
  { exchange_type: 'indodax', name: 'Indodax', type: 'cex' as const },
]

interface ExchangeConfigModalProps {
  allExchanges: Exchange[]
  editingExchangeId: string | null
  initialExchangeType?: string | null
  allowedExchangeTypes?: string[]
  onSave: (
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
  ) => Promise<void>
  onDelete: (exchangeId: string) => void
  onClose: () => void
  language: Language
}

// Step indicator component
function StepIndicator({
  currentStep,
  labels,
}: {
  currentStep: number
  labels: string[]
}) {
  return (
    <div className="flex items-center justify-center gap-2 mb-6">
      {labels.map((label, index) => (
        <React.Fragment key={index}>
          <div className="flex items-center gap-2">
            <div
              className="w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold transition-all"
              style={{
                background:
                  index < currentStep
                    ? 'var(--binance-green)'
                    : index === currentStep
                      ? 'var(--strand-accent)'
                      : 'var(--panel-bg-hover)',
                color: index <= currentStep ? '#fff' : 'var(--text-tertiary)',
              }}
            >
              {index < currentStep ? <Check className="w-4 h-4" /> : index + 1}
            </div>
            <span
              className="text-xs font-medium hidden sm:block"
              style={{ color: index === currentStep ? 'var(--text-primary)' : 'var(--text-tertiary)' }}
            >
              {label}
            </span>
          </div>
          {index < labels.length - 1 && (
            <div
              className="w-8 h-0.5 mx-1"
              style={{
                background: index < currentStep ? 'var(--binance-green)' : 'var(--panel-bg-hover)',
              }}
            />
          )}
        </React.Fragment>
      ))}
    </div>
  )
}

// Exchange card component
function ExchangeCard({
  template,
  selected,
  onClick,
  disabled,
}: {
  template: (typeof SUPPORTED_EXCHANGE_TEMPLATES)[0]
  selected: boolean
  onClick: () => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="flex flex-col items-center gap-2 p-4 rounded-xl transition-all hover:scale-105 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
      style={{
        background: selected ? 'color-mix(in srgb, var(--strand-accent) 15%, transparent)' : 'var(--panel-bg)',
        border: selected
          ? '2px solid var(--strand-accent)'
          : '2px solid var(--panel-border)',
      }}
    >
      <div className="relative">
        {getExchangeIcon(template.exchange_type, { width: 48, height: 48 })}
        {selected && (
          <div
            className="absolute -top-1 -right-1 w-5 h-5 rounded-full flex items-center justify-center"
            style={{ background: 'var(--binance-green)' }}
          >
            <Check className="w-3 h-3 text-white" />
          </div>
        )}
      </div>
      <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
        {getShortName(template.name)}
      </span>
      <span
        className="text-xs px-2 py-0.5 rounded-full"
        style={{
          background:
            template.type === 'cex'
              ? 'color-mix(in srgb, var(--strand-accent) 20%, transparent)'
              : 'color-mix(in srgb, var(--strand-accent) 20%, transparent)',
          color: template.type === 'cex' ? 'var(--strand-accent)' : 'var(--strand-accent)',
        }}
      >
        {template.type.toUpperCase()}
      </span>
    </button>
  )
}

export function ExchangeConfigModal({
  allExchanges,
  editingExchangeId,
  initialExchangeType,
  allowedExchangeTypes,
  onSave,
  onDelete,
  onClose,
  language,
}: ExchangeConfigModalProps) {
  const { user } = useAuth()
  // Step: 0 = select exchange, 1 = configure
  const [currentStep, setCurrentStep] = useState(
    editingExchangeId || initialExchangeType ? 1 : 0
  )
  const [selectedExchangeType, setSelectedExchangeType] = useState(
    initialExchangeType || ''
  )
  const [apiKey, setApiKey] = useState('')
  const [secretKey, setSecretKey] = useState('')
  const [passphrase, setPassphrase] = useState('')
  const [testnet, setTestnet] = useState(false)
  const [showGuide, setShowGuide] = useState(false)
  const [serverIP, setServerIP] = useState<{
    public_ip: string
    message: string
  } | null>(null)
  const [loadingIP, setLoadingIP] = useState(false)
  const [copiedIP, setCopiedIP] = useState(false)
  const [webCryptoStatus, setWebCryptoStatus] =
    useState<WebCryptoCheckStatus>('idle')
  const [showBinanceGuide, setShowBinanceGuide] = useState(false)

  // Aster fields
  const [asterUser, setAsterUser] = useState('')
  const [asterSigner, setAsterSigner] = useState('')
  const [asterPrivateKey, setAsterPrivateKey] = useState('')

  // Lighter fields
  const [lighterWalletAddr, setLighterWalletAddr] = useState('')
  const [lighterApiKeyPrivateKey, setLighterApiKeyPrivateKey] = useState('')
  const [lighterApiKeyIndex, setLighterApiKeyIndex] = useState(0)

  // Other state
  const [secureInputTarget, setSecureInputTarget] = useState<
    null | 'hyperliquid' | 'aster' | 'lighter'
  >(null)
  const [isSaving, setIsSaving] = useState(false)
  const [savePhase, setSavePhase] = useState<'idle' | 'testing' | 'saving'>(
    'idle'
  )
  const [saveError, setSaveError] = useState('')
  const [accountName, setAccountName] = useState('')

  const selectedExchange = editingExchangeId
    ? allExchanges?.find((e) => e.id === editingExchangeId)
    : null

  const selectedTemplate = editingExchangeId
    ? SUPPORTED_EXCHANGE_TEMPLATES.find(
        (t) => t.exchange_type === selectedExchange?.exchange_type
      )
    : SUPPORTED_EXCHANGE_TEMPLATES.find(
        (t) => t.exchange_type === selectedExchangeType
      )

  const currentExchangeType = editingExchangeId
    ? selectedExchange?.exchange_type
    : selectedExchangeType

  const exchangeRegistrationLinks: Record<
    string,
    { url: string; hasReferral?: boolean }
  > = {
    binance: {
      url: 'https://www.binance.com/join',
      hasReferral: true,
    },
    okx: { url: 'https://www.okx.com/join/1865360', hasReferral: true },
    bybit: { url: 'https://partner.bybit.com/b/83856', hasReferral: true },
    bitget: {
      url: 'https://www.bitget.com/referral/register?from=referral&clacCode=c8a43172',
      hasReferral: true,
    },
    gate: { url: 'https://www.gatenode.xyz/share/VQBGUAxY', hasReferral: true },
    kucoin: {
      url: 'https://www.kucoin.com/r/broker/CXEV7XKK',
      hasReferral: true,
    },
    hyperliquid: {
      url: 'https://app.hyperliquid.xyz/join/AITRADING',
      hasReferral: true,
    },
    aster: {
      url: 'https://www.asterdex.com/en/referral/fdfc0e',
      hasReferral: true,
    },
    lighter: {
      url: 'https://app.lighter.xyz/?referral=68151432',
      hasReferral: true,
    },
    indodax: { url: 'https://indodax.com/ref/Saep23/1', hasReferral: true },
  }

  // Initialize form when editing
  useEffect(() => {
    if (editingExchangeId && selectedExchange) {
      setAccountName(selectedExchange.account_name || '')
      setApiKey(selectedExchange.apiKey || '')
      setSecretKey(selectedExchange.secretKey || '')
      setPassphrase('')
      setTestnet(selectedExchange.testnet || false)
      setAsterUser(selectedExchange.asterUser || '')
      setAsterSigner(selectedExchange.asterSigner || '')
      setAsterPrivateKey('')
      setLighterWalletAddr(selectedExchange.lighterWalletAddr || '')
      setLighterApiKeyPrivateKey('')
      setLighterApiKeyIndex(selectedExchange.lighterApiKeyIndex || 0)
    }
  }, [editingExchangeId, selectedExchange])

  // CEX venues that typically require API key IP allowlists.
  const needsWhitelistIP = [
    'binance',
    'bybit',
    'okx',
    'bitget',
    'gate',
    'kucoin',
  ].includes(currentExchangeType || '')

  // Load server IP for CEX whitelist UX
  useEffect(() => {
    if (needsWhitelistIP && !serverIP) {
      setLoadingIP(true)
      api
        .getServerIP()
        .then((data) => setServerIP(data))
        .catch((err) => console.error('Failed to load server IP:', err))
        .finally(() => setLoadingIP(false))
    }
  }, [needsWhitelistIP, serverIP])

  const handleCopyIP = async (ip: string) => {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(ip)
        setCopiedIP(true)
        setTimeout(() => setCopiedIP(false), 2000)
        toast.success(t('ipCopied', language))
      } else {
        const textArea = document.createElement('textarea')
        textArea.value = ip
        textArea.style.position = 'fixed'
        textArea.style.left = '-999999px'
        document.body.appendChild(textArea)
        textArea.select()
        document.execCommand('copy')
        document.body.removeChild(textArea)
        setCopiedIP(true)
        setTimeout(() => setCopiedIP(false), 2000)
        toast.success(t('ipCopied', language))
      }
    } catch {
      toast.error(t('copyIPFailed', language))
    }
  }

  const secureInputContextLabel =
    secureInputTarget === 'aster'
      ? t('asterExchangeName', language)
      : secureInputTarget === 'hyperliquid'
        ? t('hyperliquidExchangeName', language)
        : undefined

  const handleSecureInputComplete = ({ value }: TwoStageKeyModalResult) => {
    const trimmed = value.trim()
    if (secureInputTarget === 'hyperliquid') setApiKey(trimmed)
    if (secureInputTarget === 'aster') setAsterPrivateKey(trimmed)
    if (secureInputTarget === 'lighter') {
      setLighterApiKeyPrivateKey(trimmed)
      toast.success(t('lighterApiKeyImported', language))
    }
    setSecureInputTarget(null)
  }

  const handleSelectExchange = (exchangeType: string) => {
    setSelectedExchangeType(exchangeType)
    setCurrentStep(1)
  }

  const handleBack = () => {
    if (editingExchangeId) {
      onClose()
    } else {
      setCurrentStep(0)
      setSelectedExchangeType('')
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isSaving) return
    if (!editingExchangeId && !selectedExchangeType) return

    const trimmedAccountName = accountName.trim()
    if (!trimmedAccountName) {
      toast.error(t('exchangeConfig.pleaseEnterAccountName', language))
      return
    }

    const exchangeId = editingExchangeId || null
    const exchangeType = currentExchangeType || ''

    if (currentExchangeType === 'hyperliquid') {
      toast.error(
        language === 'zh'
          ? '请通过钱包授权流程连接 Hyperliquid。'
          : 'Use the wallet authorization flow to connect Hyperliquid.'
      )
      return
    }

    setIsSaving(true)
    setSavePhase('testing')
    setSaveError('')
    try {
      const testPayload: Parameters<typeof api.testExchangeConfig>[0] = {
        exchange_id: exchangeId || undefined,
        exchange_type: exchangeType,
        testnet,
      }

      if (
        currentExchangeType === 'binance' ||
        currentExchangeType === 'bybit' ||
        currentExchangeType === 'indodax' ||
        currentExchangeType === 'gate'
      ) {
        if (!apiKey.trim() || !secretKey.trim()) return
        testPayload.api_key = apiKey.trim()
        testPayload.secret_key = secretKey.trim()
      } else if (
        currentExchangeType === 'okx' ||
        currentExchangeType === 'bitget' ||
        currentExchangeType === 'kucoin'
      ) {
        if (!apiKey.trim() || !secretKey.trim() || !passphrase.trim()) return
        testPayload.api_key = apiKey.trim()
        testPayload.secret_key = secretKey.trim()
        testPayload.passphrase = passphrase.trim()
      } else if (currentExchangeType === 'aster') {
        if (!asterUser.trim() || !asterSigner.trim() || !asterPrivateKey.trim())
          return
        testPayload.aster_user = asterUser.trim()
        testPayload.aster_signer = asterSigner.trim()
        testPayload.aster_private_key = asterPrivateKey.trim()
      } else if (currentExchangeType === 'lighter') {
        if (!lighterWalletAddr.trim() || !lighterApiKeyPrivateKey.trim()) return
        testPayload.lighter_wallet_addr = lighterWalletAddr.trim()
        testPayload.lighter_api_key_private_key = lighterApiKeyPrivateKey.trim()
        testPayload.lighter_api_key_index = lighterApiKeyIndex
      } else {
        if (!apiKey.trim() || !secretKey.trim()) return
        testPayload.api_key = apiKey.trim()
        testPayload.secret_key = secretKey.trim()
      }

      await api.testExchangeConfig(testPayload)
      setSavePhase('saving')

      if (
        currentExchangeType === 'binance' ||
        currentExchangeType === 'bybit' ||
        currentExchangeType === 'indodax'
      ) {
        await onSave(
          exchangeId,
          exchangeType,
          trimmedAccountName,
          apiKey.trim(),
          secretKey.trim(),
          '',
          testnet
        )
      } else if (
        currentExchangeType === 'okx' ||
        currentExchangeType === 'bitget' ||
        currentExchangeType === 'kucoin'
      ) {
        await onSave(
          exchangeId,
          exchangeType,
          trimmedAccountName,
          apiKey.trim(),
          secretKey.trim(),
          passphrase.trim(),
          testnet
        )
      } else if (currentExchangeType === 'aster') {
        await onSave(
          exchangeId,
          exchangeType,
          trimmedAccountName,
          '',
          '',
          '',
          testnet,
          undefined,
          asterUser.trim(),
          asterSigner.trim(),
          asterPrivateKey.trim()
        )
      } else if (currentExchangeType === 'lighter') {
        await onSave(
          exchangeId,
          exchangeType,
          trimmedAccountName,
          '',
          '',
          '',
          testnet,
          undefined,
          undefined,
          undefined,
          undefined,
          lighterWalletAddr.trim(),
          '',
          lighterApiKeyPrivateKey.trim(),
          lighterApiKeyIndex
        )
      } else {
        await onSave(
          exchangeId,
          exchangeType,
          trimmedAccountName,
          apiKey.trim(),
          secretKey.trim(),
          '',
          testnet
        )
      }
    } catch (error) {
      const message =
        error instanceof Error && error.message
          ? error.message
          : 'Connection test failed. Credentials were not saved.'
      setSaveError(message)
      toast.error(message)
    } finally {
      setIsSaving(false)
      setSavePhase('idle')
    }
  }

  const stepLabels = [
    t('exchangeConfig.selectExchange', language),
    t('exchangeConfig.configure', language),
  ]
  const cexExchanges = SUPPORTED_EXCHANGE_TEMPLATES.filter(
    (t) =>
      t.type === 'cex' &&
      (!allowedExchangeTypes || allowedExchangeTypes.includes(t.exchange_type))
  )
  const dexExchanges = SUPPORTED_EXCHANGE_TEMPLATES.filter(
    (t) =>
      t.type === 'dex' &&
      (!allowedExchangeTypes || allowedExchangeTypes.includes(t.exchange_type))
  )

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4 overflow-y-auto backdrop-blur-xs">
      <div
        className="rounded-2xl w-full max-w-2xl relative my-8 shadow-2xl"
        style={{ background: 'var(--panel-bg)', maxHeight: 'calc(100vh - 4rem)' }}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 pb-2">
          <div className="flex items-center gap-3">
            {currentStep > 0 && !editingExchangeId && (
              <button
                type="button"
                onClick={handleBack}
                className="p-2 rounded-lg hover:bg-black/5 transition-colors"
              >
                <ChevronLeft className="w-5 h-5" style={{ color: 'var(--text-tertiary)' }} />
              </button>
            )}
            <h3 className="text-xl font-bold" style={{ color: 'var(--text-primary)' }}>
              {editingExchangeId
                ? t('editExchange', language)
                : t('addExchange', language)}
            </h3>
          </div>
          <div className="flex items-center gap-2">
            {currentExchangeType === 'binance' && currentStep === 1 && (
              <button
                type="button"
                onClick={() => setShowGuide(true)}
                className="px-3 py-2 rounded-lg text-sm font-semibold transition-all hover:scale-105 flex items-center gap-2"
                style={{
                  background: 'color-mix(in srgb, var(--strand-accent) 10%, transparent)',
                  color: 'var(--strand-accent)',
                }}
              >
                <BookOpen className="w-4 h-4" />
                {t('viewGuide', language)}
              </button>
            )}
            {editingExchangeId && (
              <button
                type="button"
                onClick={() => onDelete(editingExchangeId)}
                className="p-2 rounded-lg hover:bg-strand-danger/20 transition-colors"
                style={{ color: 'var(--binance-red)' }}
              >
                <Trash2 className="w-4 h-4" />
              </button>
            )}
            <button
              type="button"
              onClick={onClose}
              className="p-2 rounded-lg hover:bg-black/5 transition-colors"
              style={{ color: 'var(--text-tertiary)' }}
            >
              ✕
            </button>
          </div>
        </div>

        {/* Step Indicator */}
        {!editingExchangeId && (
          <div className="px-6">
            <StepIndicator currentStep={currentStep} labels={stepLabels} />
          </div>
        )}

        {/* Content */}
        <div
          className="px-6 pb-6 overflow-y-auto"
          style={{ maxHeight: 'calc(100vh - 16rem)' }}
        >
          {/* Step 0: Select Exchange */}
          {currentStep === 0 && !editingExchangeId && (
            <div className="space-y-6">
              {/* WebCrypto Check */}
              <div className="space-y-2">
                <div
                  className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide"
                  style={{ color: 'var(--text-tertiary)' }}
                >
                  <Shield className="w-4 h-4" />
                  {t('environmentSteps.checkTitle', language)}
                </div>
                <WebCryptoEnvironmentCheck
                  language={language}
                  variant="card"
                  onStatusChange={setWebCryptoStatus}
                />
              </div>

              {/* Exchange Grid */}
              <div className="space-y-4">
                <div
                  className="text-sm font-semibold"
                  style={{ color: 'var(--text-primary)' }}
                >
                  {t('exchangeConfig.chooseExchange', language)}
                </div>

                {/* CEX */}
                <div className="space-y-3">
                  <div
                    className="text-xs font-medium uppercase tracking-wide"
                    style={{ color: 'var(--strand-accent)' }}
                  >
                    {t('exchangeConfig.centralizedExchanges', language)}
                  </div>
                  <div className="grid grid-cols-3 sm:grid-cols-5 gap-3">
                    {cexExchanges.map((template) => (
                      <ExchangeCard
                        key={template.exchange_type}
                        template={template}
                        selected={
                          selectedExchangeType === template.exchange_type
                        }
                        onClick={() =>
                          handleSelectExchange(template.exchange_type)
                        }
                        disabled={
                          webCryptoStatus !== 'secure' &&
                          webCryptoStatus !== 'disabled'
                        }
                      />
                    ))}
                  </div>
                </div>

                {/* DEX */}
                <div className="space-y-3">
                  <div
                    className="text-xs font-medium uppercase tracking-wide"
                    style={{ color: 'var(--strand-accent)' }}
                  >
                    {t('exchangeConfig.decentralizedExchanges', language)}
                  </div>
                  <div className="grid grid-cols-3 sm:grid-cols-5 gap-3">
                    {dexExchanges.map((template) => (
                      <ExchangeCard
                        key={template.exchange_type}
                        template={template}
                        selected={
                          selectedExchangeType === template.exchange_type
                        }
                        onClick={() =>
                          handleSelectExchange(template.exchange_type)
                        }
                        disabled={
                          webCryptoStatus !== 'secure' &&
                          webCryptoStatus !== 'disabled'
                        }
                      />
                    ))}
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Step 1: Configure */}
          {(currentStep === 1 || editingExchangeId) && selectedTemplate && (
            <form onSubmit={handleSubmit} className="space-y-5">
              {/* Selected Exchange Header */}
              <div
                className="p-4 rounded-xl flex items-center gap-4"
                style={{
                  background: 'var(--background)',
                  border: '1px solid var(--panel-border)',
                }}
              >
                {getExchangeIcon(selectedTemplate.exchange_type, {
                  width: 48,
                  height: 48,
                })}
                <div className="flex-1">
                  <div
                    className="font-semibold text-lg"
                    style={{ color: 'var(--text-primary)' }}
                  >
                    {getShortName(selectedTemplate.name)}
                  </div>
                  <div className="text-xs" style={{ color: 'var(--text-tertiary)' }}>
                    {selectedTemplate.type.toUpperCase()} •{' '}
                    {selectedTemplate.exchange_type}
                  </div>
                </div>
                <a
                  href={
                    exchangeRegistrationLinks[currentExchangeType || '']?.url ||
                    '#'
                  }
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 px-4 py-2 rounded-lg transition-all hover:scale-105"
                  style={{
                    background: 'color-mix(in srgb, var(--strand-accent) 10%, transparent)',
                    border: '1px solid color-mix(in srgb, var(--strand-accent) 30%, transparent)',
                  }}
                >
                  <UserPlus className="w-4 h-4" style={{ color: 'var(--strand-accent)' }} />
                  <span
                    className="text-sm font-medium"
                    style={{ color: 'var(--strand-accent)' }}
                  >
                    {t('exchangeConfig.register', language)}
                  </span>
                  {exchangeRegistrationLinks[currentExchangeType || '']
                    ?.hasReferral && (
                    <span
                      className="text-xs px-1.5 py-0.5 rounded-sm"
                      style={{
                        background: 'var(--binance-green-border)',
                        color: 'var(--binance-green)',
                      }}
                    >
                      {t('exchangeConfig.bonus', language)}
                    </span>
                  )}
                </a>
              </div>

              {/* Account Name */}
              <div className="space-y-2">
                <label
                  className="flex items-center gap-2 text-sm font-semibold"
                  style={{ color: 'var(--text-primary)' }}
                >
                  <Key className="w-4 h-4" style={{ color: 'var(--strand-accent)' }} />
                  {t('exchangeConfig.accountName', language)} *
                </label>
                <input
                  type="text"
                  value={accountName}
                  onChange={(e) => setAccountName(e.target.value)}
                  placeholder={t(
                    'exchangeConfig.accountNamePlaceholder',
                    language
                  )}
                  className="w-full px-4 py-3 rounded-xl text-base"
                  style={{
                    background: 'var(--background)',
                    border: '1px solid var(--panel-border)',
                    color: 'var(--text-primary)',
                  }}
                  required
                />
              </div>

              {/* Network: Mainnet / Testnet — shown for venues that support it */}
              {currentExchangeType &&
                currentExchangeType !== 'aster' &&
                currentExchangeType !== 'hyperliquid' && (
                  <div
                    className="p-4 rounded-xl space-y-3"
                    style={{
                      background: testnet
                        ? 'var(--binance-red-bg)'
                        : 'var(--binance-green-bg)',
                      border: `1px solid ${
                        testnet
                          ? 'var(--binance-red-border)'
                          : 'var(--binance-green-border)'
                      }`,
                    }}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <div
                          className="text-sm font-semibold"
                          style={{ color: 'var(--text-primary)' }}
                        >
                          {language === 'zh'
                            ? '网络环境'
                            : language === 'id'
                              ? 'Jaringan'
                              : 'Network'}
                        </div>
                        <div className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
                          {testnet
                            ? language === 'zh'
                              ? 'Testnet：连接测试会打到测试网端点；余额与实盘不同。'
                              : 'Testnet: credential checks hit test endpoints — balances differ from mainnet.'
                            : language === 'zh'
                              ? 'Mainnet：校验与交易均走正式环境。'
                              : 'Mainnet: verification and trading use production endpoints.'}
                        </div>
                      </div>
                      <div className="flex rounded-lg overflow-hidden border border-[var(--panel-border)]">
                        <button
                          type="button"
                          onClick={() => setTestnet(false)}
                          className="px-3 py-2 text-xs font-semibold"
                          style={{
                            background: !testnet ? 'var(--binance-green)' : 'var(--background)',
                            color: !testnet ? '#fff' : 'var(--text-tertiary)',
                          }}
                        >
                          Mainnet
                        </button>
                        <button
                          type="button"
                          onClick={() => setTestnet(true)}
                          className="px-3 py-2 text-xs font-semibold"
                          style={{
                            background: testnet ? 'var(--binance-red)' : 'var(--background)',
                            color: testnet ? '#fff' : 'var(--text-tertiary)',
                          }}
                        >
                          Testnet
                        </button>
                      </div>
                    </div>
                  </div>
                )}

              {currentExchangeType === 'aster' && (
                <div
                  className="p-3 rounded-xl text-xs"
                  style={{
                    background: 'color-mix(in srgb, var(--strand-accent) 8%, transparent)',
                    border: '1px solid color-mix(in srgb, var(--strand-accent) 22%, transparent)',
                    color: 'var(--text-tertiary)',
                  }}
                >
                  {language === 'zh'
                    ? 'Aster 使用 API Pro（User / Signer / Private Key）。保存前会做余额连通性校验；请确认字段来自同一套 API Pro 凭证。'
                    : 'Aster uses API Pro (User / Signer / Private Key). Save runs a live balance check — fields must belong to the same API Pro credential set.'}
                </div>
              )}

              {currentExchangeType === 'lighter' && (
                <div
                  className="p-3 rounded-xl text-xs"
                  style={{
                    background: testnet
                      ? 'var(--binance-red-bg)'
                      : 'color-mix(in srgb, var(--strand-accent) 8%, transparent)',
                    border: `1px solid ${
                      testnet
                        ? 'var(--binance-red-border)'
                        : 'color-mix(in srgb, var(--strand-accent) 22%, transparent)'
                    }`,
                    color: 'var(--text-tertiary)',
                  }}
                >
                  <div className="flex items-center justify-between gap-3 mb-2">
                    <span style={{ color: 'var(--text-primary)', fontWeight: 600 }}>
                      {language === 'zh'
                        ? 'Lighter 网络'
                        : 'Lighter network'}
                    </span>
                    <div className="flex rounded-lg overflow-hidden border border-[var(--panel-border)]">
                      <button
                        type="button"
                        onClick={() => setTestnet(false)}
                        className="px-3 py-1.5 text-xs font-semibold"
                        style={{
                          background: !testnet ? 'var(--binance-green)' : 'var(--background)',
                          color: !testnet ? '#fff' : 'var(--text-tertiary)',
                        }}
                      >
                        Mainnet
                      </button>
                      <button
                        type="button"
                        onClick={() => setTestnet(true)}
                        className="px-3 py-1.5 text-xs font-semibold"
                        style={{
                          background: testnet ? 'var(--binance-red)' : 'var(--background)',
                          color: testnet ? '#fff' : 'var(--text-tertiary)',
                        }}
                      >
                        Testnet
                      </button>
                    </div>
                  </div>
                  {language === 'zh'
                    ? '需要钱包地址 + API Key 私钥 + Index。Testnet/Mainnet 选错会导致连通性校验失败。'
                    : 'Needs wallet address + API key private key + index. Wrong Testnet/Mainnet selection fails the connection test.'}
                </div>
              )}

              {/* CEX Fields */}
              {(currentExchangeType === 'binance' ||
                currentExchangeType === 'bybit' ||
                currentExchangeType === 'okx' ||
                currentExchangeType === 'bitget' ||
                currentExchangeType === 'gate' ||
                currentExchangeType === 'kucoin' ||
                currentExchangeType === 'indodax') && (
                <>
                  {currentExchangeType === 'binance' && (
                    <div
                      className="p-4 rounded-xl cursor-pointer transition-colors"
                      style={{
                        background: 'color-mix(in srgb, var(--strand-accent) 8%, transparent)',
                        border: '1px solid color-mix(in srgb, var(--strand-accent) 20%, transparent)',
                      }}
                      onClick={() => setShowBinanceGuide(!showBinanceGuide)}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span style={{ color: 'var(--strand-accent)' }}>ℹ️</span>
                          <span
                            className="text-sm font-medium"
                            style={{ color: 'var(--text-primary)' }}
                          >
                            {t('exchangeConfig.useBinanceFuturesApi', language)}
                          </span>
                        </div>
                        <span style={{ color: 'var(--text-tertiary)' }}>
                          {showBinanceGuide ? '▲' : '▼'}
                        </span>
                      </div>
                      {showBinanceGuide && (
                        <div
                          className="mt-3 pt-3 text-sm"
                          style={{
                            borderTop: '1px solid color-mix(in srgb, var(--strand-accent) 20%, transparent)',
                            color: 'var(--text-primary)',
                          }}
                        >
                          <a
                            href="https://www.binance.com/en/support/faq/how-to-create-api-keys-on-binance-360002502072"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1 hover:underline"
                            style={{ color: 'var(--strand-accent)' }}
                            onClick={(e) => e.stopPropagation()}
                          >
                            {t('exchangeConfig.viewTutorial', language)}{' '}
                            <ExternalLink className="w-3 h-3" />
                          </a>
                        </div>
                      )}
                    </div>
                  )}

                  {editingExchangeId && selectedExchange && (
                    <div
                      className="p-3 rounded-xl text-xs"
                      style={{
                        background: 'var(--binance-green-bg)',
                        border: '1px solid var(--binance-green-border)',
                        color: 'var(--binance-green)',
                      }}
                    >
                      Saved credential status: API Key{' '}
                      {selectedExchange.has_api_key
                        ? 'configured'
                        : 'not configured'}
                      {' · '}
                      Secret{' '}
                      {selectedExchange.has_secret_key
                        ? 'configured'
                        : 'not configured'}
                      {currentExchangeType === 'okx' ||
                      currentExchangeType === 'bitget' ||
                      currentExchangeType === 'kucoin'
                        ? ` · Passphrase ${selectedExchange.has_passphrase ? 'configured' : 'not configured'}`
                        : ''}
                    </div>
                  )}

                  <div className="space-y-2">
                    <label
                      className="flex items-center gap-2 text-sm font-semibold"
                      style={{ color: 'var(--text-primary)' }}
                    >
                      <Key className="w-4 h-4" style={{ color: 'var(--strand-accent)' }} />
                      {t('apiKey', language)}
                    </label>
                    <input
                      type="password"
                      value={apiKey}
                      onChange={(e) => setApiKey(e.target.value)}
                      placeholder={
                        editingExchangeId && selectedExchange?.has_api_key
                          ? 'Saved. Re-enter to replace.'
                          : t('enterAPIKey', language)
                      }
                      className="w-full px-4 py-3 rounded-xl"
                      style={{
                        background: 'var(--background)',
                        border: '1px solid var(--panel-border)',
                        color: 'var(--text-primary)',
                      }}
                      required
                    />
                  </div>

                  <div className="space-y-2">
                    <label
                      className="flex items-center gap-2 text-sm font-semibold"
                      style={{ color: 'var(--text-primary)' }}
                    >
                      <Shield
                        className="w-4 h-4"
                        style={{ color: 'var(--strand-accent)' }}
                      />
                      {t('secretKey', language)}
                    </label>
                    <input
                      type="password"
                      value={secretKey}
                      onChange={(e) => setSecretKey(e.target.value)}
                      placeholder={
                        editingExchangeId && selectedExchange?.has_secret_key
                          ? 'Saved. Re-enter to replace.'
                          : t('enterSecretKey', language)
                      }
                      className="w-full px-4 py-3 rounded-xl"
                      style={{
                        background: 'var(--background)',
                        border: '1px solid var(--panel-border)',
                        color: 'var(--text-primary)',
                      }}
                      required
                    />
                  </div>

                  {(currentExchangeType === 'okx' ||
                    currentExchangeType === 'bitget' ||
                    currentExchangeType === 'kucoin') && (
                    <div className="space-y-2">
                      <label
                        className="flex items-center gap-2 text-sm font-semibold"
                        style={{ color: 'var(--text-primary)' }}
                      >
                        <Key className="w-4 h-4" style={{ color: 'var(--strand-accent)' }} />
                        {t('passphrase', language)}
                      </label>
                      <input
                        type="password"
                        value={passphrase}
                        onChange={(e) => setPassphrase(e.target.value)}
                        placeholder={
                          editingExchangeId && selectedExchange?.has_passphrase
                            ? 'Saved. Re-enter to replace.'
                            : t('enterPassphrase', language)
                        }
                        className="w-full px-4 py-3 rounded-xl"
                        style={{
                          background: 'var(--background)',
                          border: '1px solid var(--panel-border)',
                          color: 'var(--text-primary)',
                        }}
                        required
                      />
                    </div>
                  )}

                  {needsWhitelistIP && (
                    <div
                      className="p-4 rounded-xl"
                      style={{
                        background: 'color-mix(in srgb, var(--strand-accent) 10%, transparent)',
                        border: '1px solid color-mix(in srgb, var(--strand-accent) 20%, transparent)',
                      }}
                    >
                      <div
                        className="text-sm font-semibold mb-2"
                        style={{ color: 'var(--strand-accent)' }}
                      >
                        {t('whitelistIP', language)}
                      </div>
                      <div
                        className="text-xs mb-3"
                        style={{ color: 'var(--text-tertiary)' }}
                      >
                        {t('whitelistIPDesc', language)}
                      </div>
                      {loadingIP ? (
                        <div className="text-xs" style={{ color: 'var(--text-tertiary)' }}>
                          {t('loadingServerIP', language)}
                        </div>
                      ) : serverIP?.public_ip ? (
                        <div
                          className="flex items-center gap-2 p-3 rounded-lg"
                          style={{ background: 'var(--background)' }}
                        >
                          <code
                            className="flex-1 text-sm font-mono"
                            style={{ color: 'var(--strand-accent)' }}
                          >
                            {serverIP.public_ip}
                          </code>
                          <button
                            type="button"
                            onClick={() => handleCopyIP(serverIP.public_ip)}
                            className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all hover:scale-105"
                            style={{
                              background: 'color-mix(in srgb, var(--strand-accent) 20%, transparent)',
                              color: 'var(--strand-accent)',
                            }}
                          >
                            <Copy className="w-3 h-3" />
                            {copiedIP
                              ? t('ipCopied', language)
                              : t('copyIP', language)}
                          </button>
                        </div>
                      ) : null}
                    </div>
                  )}
                </>
              )}

              {/* Aster Fields */}
              {currentExchangeType === 'aster' && (
                <>
                  <div
                    className="p-4 rounded-xl"
                    style={{
                      background: 'color-mix(in srgb, var(--strand-accent) 10%, transparent)',
                      border: '1px solid color-mix(in srgb, var(--strand-accent) 30%, transparent)',
                    }}
                  >
                    <div className="flex items-start gap-2">
                      <span style={{ fontSize: '16px' }}>🔐</span>
                      <div>
                        <div
                          className="text-sm font-semibold mb-1"
                          style={{ color: 'var(--strand-accent)' }}
                        >
                          {t('asterApiProTitle', language)}
                        </div>
                        <div className="text-xs" style={{ color: 'var(--text-tertiary)' }}>
                          {t('asterApiProDesc', language)}
                        </div>
                      </div>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <label
                      className="flex items-center gap-2 text-sm font-semibold"
                      style={{ color: 'var(--text-primary)' }}
                    >
                      {t('asterUserLabel', language)}
                      <Tooltip content={t('asterUserDesc', language)}>
                        <HelpCircle
                          className="w-4 h-4 cursor-help"
                          style={{ color: 'var(--strand-accent)' }}
                        />
                      </Tooltip>
                    </label>
                    <input
                      type="text"
                      value={asterUser}
                      onChange={(e) => setAsterUser(e.target.value)}
                      placeholder={t('enterAsterUser', language)}
                      className="w-full px-4 py-3 rounded-xl"
                      style={{
                        background: 'var(--background)',
                        border: '1px solid var(--panel-border)',
                        color: 'var(--text-primary)',
                      }}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <label
                      className="flex items-center gap-2 text-sm font-semibold"
                      style={{ color: 'var(--text-primary)' }}
                    >
                      {t('asterSignerLabel', language)}
                      <Tooltip content={t('asterSignerDesc', language)}>
                        <HelpCircle
                          className="w-4 h-4 cursor-help"
                          style={{ color: 'var(--strand-accent)' }}
                        />
                      </Tooltip>
                    </label>
                    <input
                      type="text"
                      value={asterSigner}
                      onChange={(e) => setAsterSigner(e.target.value)}
                      placeholder={t('enterAsterSigner', language)}
                      className="w-full px-4 py-3 rounded-xl"
                      style={{
                        background: 'var(--background)',
                        border: '1px solid var(--panel-border)',
                        color: 'var(--text-primary)',
                      }}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <label
                      className="flex items-center gap-2 text-sm font-semibold"
                      style={{ color: 'var(--text-primary)' }}
                    >
                      {t('asterPrivateKeyLabel', language)}
                      <Tooltip content={t('asterPrivateKeyDesc', language)}>
                        <HelpCircle
                          className="w-4 h-4 cursor-help"
                          style={{ color: 'var(--strand-accent)' }}
                        />
                      </Tooltip>
                    </label>
                    <input
                      type="password"
                      value={asterPrivateKey}
                      onChange={(e) => setAsterPrivateKey(e.target.value)}
                      placeholder={t('enterAsterPrivateKey', language)}
                      className="w-full px-4 py-3 rounded-xl"
                      style={{
                        background: 'var(--background)',
                        border: '1px solid var(--panel-border)',
                        color: 'var(--text-primary)',
                      }}
                      required
                    />
                  </div>
                </>
              )}

              {/* Hyperliquid Wallet Authorization */}
              {currentExchangeType === 'hyperliquid' && (
                <div className="space-y-4">
                  <div
                    className="p-4 rounded-xl"
                    style={{
                      background: 'color-mix(in srgb, var(--strand-accent) 10%, transparent)',
                      border: '1px solid color-mix(in srgb, var(--strand-accent) 30%, transparent)',
                    }}
                  >
                    <div className="flex items-start gap-2">
                      <span style={{ fontSize: '16px' }}>🔐</span>
                      <div>
                        <div
                          className="text-sm font-semibold mb-1"
                          style={{ color: 'var(--strand-accent)' }}
                        >
                          {language === 'zh'
                            ? 'Hyperliquid 需要钱包授权'
                            : 'Hyperliquid requires wallet authorization'}
                        </div>
                        <div
                          className="text-xs leading-5"
                          style={{ color: 'var(--text-tertiary)' }}
                        >
                          {language === 'zh'
                            ? '不支持手动填私钥/API Key。请用 MetaMask、Rabby、OKX、Coinbase Wallet 等 EVM 钱包连接，授权 Agent 并批准 builder fee。'
                            : 'Manual private-key/API-key entry is disabled. Use MetaMask, Rabby, OKX, Coinbase Wallet or another EVM wallet to connect, authorize the agent, and approve the builder fee.'}
                        </div>
                      </div>
                    </div>
                  </div>
                  <div className="flex justify-start">
                    <HyperliquidWalletConnect
                      language={language}
                      isLoggedIn={Boolean(user)}
                      variant="inline"
                    />
                  </div>
                  {selectedExchange?.hyperliquidWalletAddr && (
                    <HyperliquidFundsPanel
                      language={language}
                      walletAddress={selectedExchange.hyperliquidWalletAddr}
                      unifiedAccount={
                        selectedExchange.hyperliquidUnifiedAccount ?? true
                      }
                    />
                  )}
                </div>
              )}

              {/* Lighter Fields */}
              {currentExchangeType === 'lighter' && (
                <>
                  <div
                    className="p-4 rounded-xl"
                    style={{
                      background: 'color-mix(in srgb, var(--strand-accent) 10%, transparent)',
                      border: '1px solid color-mix(in srgb, var(--strand-accent) 30%, transparent)',
                    }}
                  >
                    <div className="flex items-start gap-2">
                      <span style={{ fontSize: '16px' }}>🔐</span>
                      <div>
                        <div
                          className="text-sm font-semibold mb-1"
                          style={{ color: 'var(--strand-accent)' }}
                        >
                          {t('exchangeConfig.lighterApiKeySetup', language)}
                        </div>
                        <div className="text-xs" style={{ color: 'var(--text-tertiary)' }}>
                          {t('exchangeConfig.lighterApiKeyDesc', language)}
                        </div>
                      </div>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <label
                      className="text-sm font-semibold"
                      style={{ color: 'var(--text-primary)' }}
                    >
                      {t('lighterWalletAddress', language)} *
                    </label>
                    <input
                      type="text"
                      value={lighterWalletAddr}
                      onChange={(e) => setLighterWalletAddr(e.target.value)}
                      placeholder={t('enterLighterWalletAddress', language)}
                      className="w-full px-4 py-3 rounded-xl"
                      style={{
                        background: 'var(--background)',
                        border: '1px solid var(--panel-border)',
                        color: 'var(--text-primary)',
                      }}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <label
                      className="flex items-center gap-2 text-sm font-semibold"
                      style={{ color: 'var(--text-primary)' }}
                    >
                      {t('lighterApiKeyPrivateKey', language)} *
                      <button
                        type="button"
                        onClick={() => setSecureInputTarget('lighter')}
                        className="text-xs underline"
                        style={{ color: 'var(--strand-accent)' }}
                      >
                        {t('secureInputButton', language)}
                      </button>
                    </label>
                    <input
                      type="password"
                      value={lighterApiKeyPrivateKey}
                      onChange={(e) =>
                        setLighterApiKeyPrivateKey(e.target.value)
                      }
                      placeholder={t('enterLighterApiKeyPrivateKey', language)}
                      className="w-full px-4 py-3 rounded-xl font-mono"
                      style={{
                        background: 'var(--background)',
                        border: '1px solid var(--panel-border)',
                        color: 'var(--text-primary)',
                      }}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <label
                      className="flex items-center gap-2 text-sm font-semibold"
                      style={{ color: 'var(--text-primary)' }}
                    >
                      {t('exchangeConfig.apiKeyIndex', language)}
                      <Tooltip
                        content={t(
                          'exchangeConfig.apiKeyIndexTooltip',
                          language
                        )}
                      >
                        <HelpCircle
                          className="w-4 h-4 cursor-help"
                          style={{ color: 'var(--strand-accent)' }}
                        />
                      </Tooltip>
                    </label>
                    <input
                      type="number"
                      min={0}
                      max={255}
                      value={lighterApiKeyIndex}
                      onChange={(e) =>
                        setLighterApiKeyIndex(parseInt(e.target.value) || 0)
                      }
                      className="w-full px-4 py-3 rounded-xl"
                      style={{
                        background: 'var(--background)',
                        border: '1px solid var(--panel-border)',
                        color: 'var(--text-primary)',
                      }}
                    />
                  </div>
                </>
              )}

              {/* Buttons */}
              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={handleBack}
                  className="flex-1 px-4 py-3 rounded-xl text-sm font-semibold transition-all hover:bg-black/5"
                  style={{ background: 'var(--panel-bg-hover)', color: 'var(--text-tertiary)' }}
                >
                  {currentExchangeType === 'hyperliquid'
                    ? t('closeGuide', language)
                    : editingExchangeId
                      ? t('cancel', language)
                      : t('exchangeConfig.back', language)}
                </button>
                {currentExchangeType !== 'hyperliquid' && (
                  <button
                    type="submit"
                    disabled={isSaving || !accountName.trim()}
                    className="flex-1 flex items-center justify-center gap-2 px-4 py-3 rounded-xl text-sm font-bold transition-all hover:scale-[1.02] disabled:opacity-50 disabled:cursor-not-allowed"
                    style={{ background: 'var(--strand-accent)', color: '#fff' }}
                  >
                    {isSaving
                      ? savePhase === 'testing'
                        ? t('modelConfig.testingConnection', language)
                        : t('saving', language)
                      : (
                          <>
                            {t('saveConfig', language)}{' '}
                            <ArrowRight className="w-4 h-4" />
                          </>
                        )}
                  </button>
                )}
              </div>
              {saveError ? (
                <p role="alert" className="text-sm text-red-600 pt-2">
                  {saveError}
                </p>
              ) : null}
            </form>
          )}
        </div>
      </div>

      {/* Binance Guide Modal */}
      {showGuide && (
        <div
          className="fixed inset-0 bg-black/75 flex items-center justify-center z-50 p-4"
          onClick={() => setShowGuide(false)}
        >
          <div
            className="rounded-2xl p-6 w-full max-w-4xl"
            style={{ background: 'var(--panel-bg)' }}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-4">
              <h3
                className="text-xl font-bold flex items-center gap-2"
                style={{ color: 'var(--text-primary)' }}
              >
                <BookOpen className="w-6 h-6" style={{ color: 'var(--strand-accent)' }} />
                {t('binanceSetupGuide', language)}
              </h3>
              <button
                onClick={() => setShowGuide(false)}
                className="px-4 py-2 rounded-lg text-sm font-semibold"
                style={{ background: 'var(--panel-bg-hover)', color: 'var(--text-tertiary)' }}
              >
                {t('closeGuide', language)}
              </button>
            </div>
            <div className="overflow-y-auto max-h-[80vh]">
              <img
                src="/images/guide.png"
                alt={t('binanceSetupGuide', language)}
                className="w-full h-auto rounded-lg"
              />
            </div>
          </div>
        </div>
      )}

      {/* Secure Input Modal */}
      <TwoStageKeyModal
        isOpen={secureInputTarget !== null}
        language={language}
        contextLabel={secureInputContextLabel}
        expectedLength={64}
        onCancel={() => setSecureInputTarget(null)}
        onComplete={handleSecureInputComplete}
      />
    </div>
  )
}
