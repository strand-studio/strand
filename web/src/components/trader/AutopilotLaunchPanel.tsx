import { useEffect, useMemo, useState, type JSX } from 'react'
import { useNavigate } from 'react-router'
import {
  AlertCircle,
  ArrowRight,
  CircleDollarSign,
  CheckCircle2,
  Copy,
  ExternalLink,
  Loader2,
  RefreshCw,
  ShieldCheck,
  Wallet,
  Zap,
} from 'lucide-react'
import { toast } from 'sonner'
import { api } from '../../lib/api'
import { buildDashboardPath, ROUTES } from '../../router/paths'
import {
  ensureClaw402Strategy,
  launchAutopilot,
} from '../../lib/launch/launchAutopilot'
import { runLaunchPreflight } from '../../lib/launch/preflight'
import type { LaunchPreflightResult } from '../../lib/launch/types'
import {
  isHyperliquidExchange,
  pickTradingExchange,
} from '../../lib/launch/resolve'
import type {
  AIModel,
  CurrentBeginnerWalletResponse,
  Exchange,
  ExchangeAccountState,
  TraderInfo,
} from '../../types'
import { HyperliquidWalletConnect } from '../common/HyperliquidWalletConnect'

type LaunchStepStatus = 'ready' | 'action' | 'blocked'

interface AutopilotLaunchPanelProps {
  models: AIModel[]
  exchanges: Exchange[]
  exchangeAccountStates: Record<string, ExchangeAccountState>
  traders?: TraderInfo[]
  isLoggedIn: boolean
  language: string
  onRefresh: () => Promise<void>
  onOpenClaw402Config?: () => void
  onOpenHyperliquidConfig?: () => void
  onOpenExchangeConfig?: (exchangeId?: string | null) => void
}

const MIN_AI_FEE_USDC = 1
const MIN_TRADING_USDC = 12

function parseNumber(value?: string | number) {
  if (typeof value === 'number') return Number.isFinite(value) ? value : 0
  if (!value) return 0
  const parsed = Number(value.replace(/[,$\s]/g, ''))
  return Number.isFinite(parsed) ? parsed : 0
}

function shortAddress(address?: string) {
  if (!address) return '--'
  return `${address.slice(0, 6)}…${address.slice(-4)}`
}

function formatUSDC(value: number) {
  return new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}

async function copyText(
  value: string,
  okMessage: string,
  failMessage: string
) {
  try {
    await navigator.clipboard.writeText(value)
    toast.success(okMessage)
  } catch {
    toast.error(failMessage)
  }
}

export function AutopilotLaunchPanel({
  models,
  exchanges,
  exchangeAccountStates,
  traders = [],
  isLoggedIn,
  language,
  onRefresh,
  onOpenClaw402Config,
  onOpenHyperliquidConfig,
  onOpenExchangeConfig,
}: AutopilotLaunchPanelProps) {
  const navigate = useNavigate()
  const [wallet, setWallet] = useState<CurrentBeginnerWalletResponse | null>(
    null
  )
  const [walletLoading, setWalletLoading] = useState(false)
  const [launching, setLaunching] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const isZh = language === 'zh'

  const claw402Model = useMemo(
    () =>
      models.find(
        (model) =>
          model.provider === 'claw402' &&
          model.enabled &&
          (model.has_api_key || model.apiKey || model.walletAddress)
      ) || null,
    [models]
  )

  const hyperliquidExchange = useMemo(
    () =>
      exchanges.find(
        (exchange) =>
          exchange.exchange_type === 'hyperliquid' &&
          exchange.enabled &&
          Boolean(exchange.hyperliquidWalletAddr) &&
          Boolean(exchange.hyperliquidBuilderApproved)
      ) || null,
    [exchanges]
  )

  // Prefer ready HL, else first ready CEX — same policy as launchAutopilot.
  const launchExchange = useMemo(
    () => pickTradingExchange(exchanges),
    [exchanges]
  )
  const launchIsHyperliquid = Boolean(
    launchExchange && isHyperliquidExchange(launchExchange)
  )

  // Preflight target: ready venue, else any partial HL / CEX so the server can
  // report the missing piece instead of blanking the panel.
  const preflightExchange = useMemo(
    () =>
      launchExchange ||
      hyperliquidExchange ||
      exchanges.find((exchange) => exchange.exchange_type === 'hyperliquid') ||
      exchanges.find((exchange) => exchange.enabled) ||
      null,
    [exchanges, hyperliquidExchange, launchExchange]
  )

  // Server-side preflight is the source of truth for balances: it queries the
  // chain / exchange live (30s server cache) instead of trusting the balance
  // snapshot cached in the model object. Poll while the panel is visible so
  // deposits show up without a manual refresh.
  const [preflight, setPreflight] = useState<LaunchPreflightResult | null>(null)
  const claw402ModelId = claw402Model?.id
  const preflightExchangeId = preflightExchange?.id
  useEffect(() => {
    if (!isLoggedIn || !claw402ModelId || !preflightExchangeId) {
      setPreflight(null)
      return
    }
    let cancelled = false
    const check = async () => {
      try {
        const result = await runLaunchPreflight({
          ai_model_id: claw402ModelId,
          exchange_id: preflightExchangeId,
        })
        if (!cancelled) setPreflight(result)
      } catch {
        // keep the last known result; client-derived fallbacks still render
      }
    }
    void check()
    const timer = setInterval(() => void check(), 20000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [isLoggedIn, claw402ModelId, preflightExchangeId])

  const preflightCheck = (id: string) =>
    preflight?.checks.find((check) => check.id === id)

  const feeWalletAddress =
    claw402Model?.walletAddress ||
    wallet?.address ||
    preflightCheck('ai_wallet')?.address ||
    ''
  const feeFundsCheck = preflightCheck('ai_wallet_funds')
  const feeWalletBalance =
    feeFundsCheck?.actual ??
    parseNumber(claw402Model?.balanceUsdc || wallet?.balance_usdc)
  const minAIFeeUSDC = preflight?.min_ai_fee_usdc ?? MIN_AI_FEE_USDC
  const feeReady = feeFundsCheck
    ? feeFundsCheck.status !== 'failed' && Boolean(feeWalletAddress)
    : Boolean(feeWalletAddress) && feeWalletBalance >= minAIFeeUSDC

  // A stored builder-approved flag is not proof that this saved signing agent
  // is still approved. A different Strand installation can replace the agent.
  const [verifiedExchangeId, setVerifiedExchangeId] = useState<string | null>(
    null
  )
  useEffect(() => {
    setVerifiedExchangeId(null)
    if (
      !hyperliquidExchange?.hyperliquidWalletAddr ||
      !hyperliquidExchange.hyperliquidAgentAddress ||
      hyperliquidExchange.testnet
    )
      return
    let cancelled = false
    const check = async () => {
      try {
        const proof = await api.getHyperliquidAgent(
          hyperliquidExchange.hyperliquidWalletAddr!
        )
        const approved =
          proof.builderApproved &&
          proof.agents?.some(
            (agent) =>
              agent.address.toLowerCase() ===
                hyperliquidExchange.hyperliquidAgentAddress!.toLowerCase() &&
              agent.validUntil > Date.now()
          )
        if (!cancelled)
          setVerifiedExchangeId(approved ? hyperliquidExchange.id : null)
      } catch {
        if (!cancelled) setVerifiedExchangeId(null)
      }
    }
    void check()
    const timer = setInterval(() => void check(), 30000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [hyperliquidExchange])
  const hyperliquidConnected =
    !!hyperliquidExchange && verifiedExchangeId === hyperliquidExchange.id
  const exchangeState = preflightExchange
    ? exchangeAccountStates[preflightExchange.id]
    : undefined
  const accountCheck = preflightCheck('exchange_account')
  const tradingFundsCheck = preflightCheck('exchange_funds')
  const tradeAuthCheck = preflightCheck('exchange_trade_auth')
  // CEX: require live preflight account ok + trade-auth not failed (not just keys on file).
  const cexConnectedStrict =
    Boolean(launchExchange) &&
    (preflight
      ? accountCheck?.status === 'ok' &&
        (tradeAuthCheck == null ||
          tradeAuthCheck.status === 'ok' ||
          tradeAuthCheck.status === 'skipped')
      : exchangeState?.status === 'ok')
  const exchangeConnected = launchIsHyperliquid
    ? hyperliquidConnected
    : cexConnectedStrict
  const tradingBalance =
    tradingFundsCheck?.actual ??
    parseNumber(exchangeState?.available_balance ?? exchangeState?.total_equity)
  const minTradingUSDC = preflight?.min_trading_usdc ?? MIN_TRADING_USDC
  const tradingBalanceReady =
    exchangeConnected &&
    (accountCheck && tradingFundsCheck
      ? accountCheck.status === 'ok' && tradingFundsCheck.status !== 'failed'
      : exchangeState?.status === 'ok' && tradingBalance >= minTradingUSDC)

  const autopilotTrader = useMemo(
    () =>
      traders.find((trader) => trader.trader_name === 'Strand Autopilot') ||
      traders.find((trader) => trader.trader_name === 'NOFX Autopilot') ||
      traders.find((trader) =>
        (trader.strategy_name || '').toLowerCase().includes('claw402')
      ) ||
      null,
    [traders]
  )

  const allReady = feeReady && exchangeConnected && tradingBalanceReady

  const loadWallet = async () => {
    setWalletLoading(true)
    try {
      setWallet(await api.getCurrentBeginnerWallet())
    } catch {
      setWallet(null)
    } finally {
      setWalletLoading(false)
    }
  }

  useEffect(() => {
    void loadWallet()
  }, [])

  const refreshEverything = async () => {
    setRefreshing(true)
    try {
      await Promise.all([onRefresh(), loadWallet()])
    } finally {
      setRefreshing(false)
    }
  }

  const handleLaunch = async () => {
    if (!claw402Model || !launchExchange) return
    setLaunching(true)
    try {
      // Shared launch path (same as Strategy Studio): server preflight with
      // fresh balances first, then strategy provisioning, then create/start.
      const outcome = await launchAutopilot({
        ensureStrategy: ensureClaw402Strategy,
        scanIntervalMinutes: 5,
      })

      if (!outcome.ok) {
        toast.error(outcome.message)
        if (outcome.kind === 'preflight') {
          setPreflight(outcome.preflight)
        }
        if (outcome.kind !== 'error') {
          if (outcome.setupTarget === 'claw402') {
            onOpenClaw402Config?.()
          } else if (outcome.setupTarget === 'hyperliquid') {
            onOpenHyperliquidConfig?.()
          } else if (outcome.setupTarget === 'exchange') {
            const exchangeId =
              outcome.kind === 'preflight'
                ? outcome.preflight.checks.find(
                    (c) => c.status === 'failed' && c.exchange_id
                  )?.exchange_id
                : null
            onOpenExchangeConfig?.(exchangeId)
          }
        }
        await refreshEverything()
        return
      }

      if (outcome.warning) {
        toast.warning(outcome.warning)
      }
      await onRefresh()
      toast.success(
        isZh ? 'Strand Autopilot 已在运行' : 'Strand Autopilot is running'
      )
      navigate(buildDashboardPath(outcome.traderId))
    } finally {
      setLaunching(false)
    }
  }

  const steps: Array<{
    title: string
    detail: string
    status: LaunchStepStatus
    meta?: string
    action?: JSX.Element
  }> = [
    {
      title: isZh
        ? '步骤 1 · 给 AI 钱包充值（$1+）'
        : 'Step 1 · Fund the AI wallet ($1+)',
      detail: isZh
        ? 'AI 每次思考会支付极少费用。向该地址转入 Base 网络上 ≥$1 的 USDC——可从币安、OKX、Coinbase 或任意钱包。与交易资金分开。'
        : 'The AI pays a tiny fee each time it thinks. Send $1 or more of USDC on the Base network to this address — from Binance, OKX, Coinbase or any wallet. Separate from your trading money.',
      status: feeReady ? 'ready' : 'action',
      meta: feeWalletAddress
        ? `${shortAddress(feeWalletAddress)} · ${formatUSDC(feeWalletBalance)} USDC${
            feeReady
              ? ''
              : isZh
                ? ` · 需要 ≥ ${minAIFeeUSDC} USDC`
                : ` · needs ≥ ${minAIFeeUSDC} USDC`
          }`
        : isZh
          ? '约 1 分钟——我们会帮你创建钱包'
          : 'Takes 1 minute — we create the wallet for you',
      action: feeWalletAddress ? (
        <div className="flex flex-wrap items-center gap-3">
          <button
            type="button"
            onClick={() => navigate(`${ROUTES.welcome}?deposit=1`)}
            className="inline-flex items-center gap-1.5 text-xs font-semibold text-strand-gold hover:text-strand-accent"
          >
            <CircleDollarSign className="h-3.5 w-3.5" />
            {isZh ? '充值' : 'Deposit'}
          </button>
          <button
            type="button"
            onClick={() =>
              void copyText(
                feeWalletAddress,
                isZh ? '已复制 AI 费用钱包地址' : 'AI fee wallet copied',
                isZh ? '复制失败' : 'Copy failed'
              )
            }
            className="inline-flex items-center gap-1.5 text-xs font-semibold text-strand-gold hover:text-strand-accent"
          >
            <Copy className="h-3.5 w-3.5" />
            {isZh ? '复制' : 'Copy'}
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => navigate(`${ROUTES.welcome}?deposit=1`)}
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-strand-gold hover:text-strand-accent"
        >
          <ArrowRight className="h-3.5 w-3.5" />
          {isZh ? '创建' : 'Create'}
        </button>
      ),
    },
    {
      title: launchIsHyperliquid
        ? isZh
          ? '步骤 2 · 连接 Hyperliquid'
          : 'Step 2 · Connect Hyperliquid'
        : launchExchange
          ? isZh
            ? `步骤 2 · 交易场所 · ${launchExchange.account_name || launchExchange.exchange_type}`
            : `Step 2 · Trading venue · ${launchExchange.account_name || launchExchange.exchange_type}`
          : isZh
            ? '步骤 2 · 连接交易场所'
            : 'Step 2 · Connect a trading venue',
      detail: launchIsHyperliquid
        ? isZh
          ? '用加密钱包（Rabby 或 MetaMask）授权一次 Strand。AI 可为你下单，但永远无法提走资金。'
          : 'Approve Strand once with your crypto wallet (Rabby or MetaMask). This lets the AI place trades for you — it can never withdraw your money.'
        : launchExchange
          ? isZh
            ? 'API 密钥（及所需 IP 白名单）已登记。启动前服务器会再次校验权限与余额。'
            : 'API keys (and IP whitelist if required) are already on file. Server preflight will re-check permissions and balance before start.'
          : isZh
            ? '连接 Hyperliquid（钱包授权）或添加 CEX API 账户（币安 / Bybit / OKX…）。'
            : 'Connect Hyperliquid (wallet authorize) or add a CEX API account (Binance / Bybit / OKX…).',
      status: exchangeConnected ? 'ready' : 'action',
      meta: launchIsHyperliquid
        ? hyperliquidExchange?.hyperliquidWalletAddr
          ? `${shortAddress(hyperliquidExchange.hyperliquidWalletAddr)} · ${
              hyperliquidConnected
                ? isZh
                  ? '已授权'
                  : 'authorized'
                : isZh
                  ? '授权未验证 — 请重新连接'
                  : 'authorization not verified — reconnect'
            } `
          : isZh
            ? '几次点击 + 3 次钱包签名'
            : 'A few clicks + 3 wallet signatures'
        : launchExchange
          ? isZh
            ? `${launchExchange.exchange_type} · 凭证已就绪`
            : `${launchExchange.exchange_type} · credentials present`
          : isZh
            ? '需要 Hyperliquid 或 CEX 凭证'
            : 'Hyperliquid or CEX credentials required',
      action: (
        <button
          type="button"
          onClick={() =>
            launchIsHyperliquid || !launchExchange
              ? onOpenHyperliquidConfig?.()
              : onOpenExchangeConfig?.(launchExchange.id)
          }
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-strand-gold hover:text-strand-accent"
        >
          <Wallet className="h-3.5 w-3.5" />
          {isZh ? '打开' : 'Open'}
        </button>
      ),
    },
    {
      title: isZh
        ? '步骤 3 · 注入交易资金（$12+）'
        : 'Step 3 · Add trading money ($12+)',
      detail: launchIsHyperliquid
        ? isZh
          ? '向 Hyperliquid 账户存入 USDC（app.hyperliquid.xyz → Deposit，Arbitrum 上的 USDC）。这是 AI 用来交易的资金——可先少存，随时加仓。'
          : 'Deposit USDC into your Hyperliquid account (app.hyperliquid.xyz → Deposit, USDC on Arbitrum). This is what the AI trades with — start small, you can add more anytime.'
        : isZh
          ? '在已连接交易所的合约钱包中保持至少 $12 可用。这是 Autopilot 交易用资金。'
          : 'Keep at least $12 available on the connected exchange futures wallet. This is what Autopilot trades with.',
      status: tradingBalanceReady
        ? 'ready'
        : exchangeConnected
          ? 'action'
          : 'blocked',
      meta: exchangeConnected
        ? `${formatUSDC(tradingBalance)} USDC ${
            isZh ? '可用' : 'available'
          }${
            tradingBalanceReady
              ? ''
              : isZh
                ? ` · 需要 ≥ ${minTradingUSDC} USDC`
                : ` · needs ≥ ${minTradingUSDC} USDC`
          }`
        : isZh
          ? '请先完成步骤 2'
          : 'Finish step 2 first',
    },
    {
      title: isZh ? '步骤 4 · 点击启动' : 'Step 4 · Press start',
      detail: isZh
        ? 'AI 每隔几分钟读取市场、自行选单并管理仓位。在「资产」页可实时观看每一步决策——随时一键停止。'
        : 'The AI reads the market every few minutes, picks its trades, and manages them on its own. Watch every decision live on Portfolio — stop it with one click anytime.',
      status: allReady ? 'ready' : 'blocked',
      meta: autopilotTrader?.is_running
        ? isZh
          ? '运行中 — 打开资产页查看'
          : 'Running — open Portfolio to watch'
        : autopilotTrader
          ? isZh
            ? '可以启动'
            : 'Ready to start'
          : allReady
            ? isZh
              ? '一切就绪 — 按下按钮'
              : 'Everything is ready — press the button'
            : isZh
              ? '步骤 1–3 变绿后解锁'
              : 'Unlocks when steps 1–3 are green',
    },
  ]

  const renderPrimaryAction = () => {
    if (!feeReady) {
      return (
        <button
          type="button"
          onClick={() => navigate(`${ROUTES.welcome}?deposit=1`)}
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-strand-gold px-4 py-3 text-sm font-bold text-white hover:bg-strand-accent"
        >
          {isZh ? '设置 AI 钱包' : 'Set up the AI wallet'}
          <ArrowRight className="h-4 w-4" />
        </button>
      )
    }

    if (!exchangeConnected) {
      return (
        <button
          type="button"
          onClick={() => {
            if (launchExchange && !launchIsHyperliquid) {
              onOpenExchangeConfig?.(launchExchange.id)
              return
            }
            if (onOpenHyperliquidConfig) {
              onOpenHyperliquidConfig()
            } else {
              document
                .getElementById('hyperliquid-quick-connect')
                ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
            }
          }}
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-strand-gold px-4 py-3 text-sm font-bold text-white hover:bg-strand-accent"
        >
          {launchExchange && !launchIsHyperliquid
            ? isZh
              ? '完成交易所配置'
              : 'Finish exchange setup'
            : isZh
              ? '连接 Hyperliquid'
              : 'Connect Hyperliquid'}
          <ArrowRight className="h-4 w-4" />
        </button>
      )
    }

    if (!tradingBalanceReady) {
      if (launchIsHyperliquid) {
        return (
          <a
            href="https://app.hyperliquid.xyz/"
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-strand-gold px-4 py-3 text-sm font-bold text-white hover:bg-strand-accent"
          >
            {isZh ? '在 Hyperliquid 存入 USDC' : 'Deposit USDC on Hyperliquid'}
            <ExternalLink className="h-4 w-4" />
          </a>
        )
      }
      return (
        <button
          type="button"
          onClick={() => onOpenExchangeConfig?.(launchExchange?.id)}
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-strand-gold px-4 py-3 text-sm font-bold text-white hover:bg-strand-accent"
        >
          {isZh ? '查看交易所余额' : 'Check exchange balance'}
          <ArrowRight className="h-4 w-4" />
        </button>
      )
    }

    if (autopilotTrader?.is_running) {
      return (
        <button
          type="button"
          onClick={() =>
            navigate(buildDashboardPath(autopilotTrader.trader_id))
          }
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-strand-success px-4 py-3 text-sm font-bold text-white hover:bg-strand-success/80"
        >
          {isZh ? '打开资产' : 'Open Portfolio'}
          <ArrowRight className="h-4 w-4" />
        </button>
      )
    }

    return (
      <button
        type="button"
        onClick={() => void handleLaunch()}
        disabled={launching || !allReady}
        className="inline-flex items-center justify-center gap-2 rounded-lg bg-strand-gold px-4 py-3 text-sm font-bold text-white hover:bg-strand-accent disabled:cursor-not-allowed disabled:opacity-60"
      >
        {launching ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <Zap className="h-4 w-4" />
        )}
        {isZh ? '启动 Strand Autopilot' : 'Start Strand Autopilot'}
      </button>
    )
  }

  return (
    <section
      id="autopilot-launch-panel"
      className="overflow-hidden rounded-xl border border-strand-gold/20 bg-strand-bg-lighter"
    >
      <div className="grid gap-0 xl:grid-cols-[1.05fr_0.95fr]">
        <div className="p-5 md:p-6">
          <div className="mb-5 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <div className="mb-2 inline-flex items-center gap-2 rounded-full border border-strand-gold/25 bg-strand-gold/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-strand-gold">
                <ShieldCheck className="h-3.5 w-3.5" />
                {isZh ? '引导启动' : 'Guided Launch'}
              </div>
              <h2 className="text-2xl font-bold tracking-tight text-strand-text md:text-3xl">
                {isZh
                  ? '几分钟启动 Strand Autopilot'
                  : 'Start Strand Autopilot in minutes'}
              </h2>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-strand-text-muted">
                {launchIsHyperliquid || !launchExchange
                  ? isZh
                    ? '四步启动：AI 费钱包 ≥ $1 Base USDC；交易保证金 ≥ $12；Hyperliquid 钱包授权 Agent（不可提现）。通过 Preflight 后启动，随时可在资产页暂停 / 停止。'
                    : 'Four steps: AI fee wallet ≥ $1 Base USDC; trading margin ≥ $12; Hyperliquid wallet-authorize a trade-only Agent. Pass Preflight, then start — pause or stop anytime from Portfolio.'
                  : isZh
                    ? `使用 ${launchExchange.account_name || launchExchange.exchange_type}。确认 API 凭证、IP 白名单，以及可用 ≥$${minTradingUSDC}，再通过 Preflight 启动。`
                    : `Using ${launchExchange.account_name || launchExchange.exchange_type}. Confirm API credentials, IP whitelist, and ≥$${minTradingUSDC} available, then pass Preflight to start.`}
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => void refreshEverything()}
                disabled={refreshing || walletLoading}
                className="inline-flex items-center justify-center gap-2 rounded-lg border border-strand-gold/20 bg-strand-bg-deeper px-3 py-2 text-xs font-semibold text-strand-text-muted hover:text-strand-text disabled:opacity-60"
              >
                <RefreshCw
                  className={`h-3.5 w-3.5 ${refreshing || walletLoading ? 'animate-spin' : ''}`}
                />
                {isZh ? '刷新' : 'Refresh'}
              </button>
              {renderPrimaryAction()}
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            {steps.map((step, index) => (
              <div
                key={step.title}
                className="rounded-lg border border-strand-gold/20 bg-strand-bg p-4"
              >
                <div className="flex items-start gap-3">
                  <div
                    className={`mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border text-sm font-bold ${
                      step.status === 'ready'
                        ? 'border-strand-success/30 bg-strand-success/15 text-strand-success'
                        : step.status === 'action'
                          ? 'border-strand-gold/30 bg-strand-gold/15 text-strand-gold'
                          : 'border-strand-gold/20 bg-strand-bg-deeper text-strand-text-muted'
                    }`}
                  >
                    {step.status === 'ready' ? (
                      <CheckCircle2 className="h-4 w-4" />
                    ) : step.status === 'action' ? (
                      index + 1
                    ) : (
                      <AlertCircle className="h-4 w-4" />
                    )}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <h3 className="font-semibold text-strand-text">
                        {step.title}
                      </h3>
                      {step.action}
                    </div>
                    <p className="mt-1 text-xs leading-5 text-strand-text-muted">
                      {step.detail}
                    </p>
                    <div className="mt-3 font-mono text-xs text-strand-gold/90">
                      {step.meta}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <aside className="border-t border-strand-gold/20 bg-strand-bg p-5 md:p-6 xl:border-l xl:border-t-0">
          <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-strand-text">
            <Wallet className="h-4 w-4 text-strand-gold" />
            {launchIsHyperliquid || !launchExchange
              ? isZh
                ? 'Hyperliquid 授权'
                : 'Hyperliquid setup'
              : isZh
                ? `交易所 · ${launchExchange.account_name || launchExchange.exchange_type}`
                : `Exchange · ${launchExchange.account_name || launchExchange.exchange_type}`}
          </div>
          {launchIsHyperliquid || !launchExchange ? (
            hyperliquidConnected ? (
              <div className="rounded-lg border border-strand-success/25 bg-strand-success/10 p-4">
                <div className="flex items-center gap-2 text-sm font-semibold text-strand-success">
                  <CheckCircle2 className="h-4 w-4" />
                  {isZh ? '交易授权已就绪' : 'Trading authorization is ready'}
                </div>
                <div className="mt-2 font-mono text-xs text-strand-success/90">
                  {shortAddress(hyperliquidExchange?.hyperliquidWalletAddr)}
                </div>
                <p className="mt-3 text-xs leading-5 text-strand-text-muted">
                  {isZh
                    ? '资金留在你的 Hyperliquid 账户。Strand 仅保存自动化执行所需的 Agent 密钥。'
                    : 'Funds stay in your Hyperliquid account. Strand only stores the authorized Agent key required for automated execution.'}
                </p>
              </div>
            ) : (
              <div>
                <div id="hyperliquid-quick-connect">
                  <HyperliquidWalletConnect
                    language={isZh ? 'zh' : 'en'}
                    isLoggedIn={isLoggedIn}
                    variant="inline"
                    onSaved={refreshEverything}
                  />
                </div>
              </div>
            )
          ) : (
            <div className="rounded-lg border border-strand-gold/20 bg-strand-bg-deeper p-4">
              <div className="text-sm font-semibold text-strand-text">
                {exchangeConnected
                  ? isZh
                    ? '交易所账户已通过校验'
                    : 'Exchange account verified'
                  : isZh
                    ? '完成 API 密钥与 IP 白名单'
                    : 'Finish API keys and IP allowlist'}
              </div>
              <p className="mt-2 text-xs leading-5 text-strand-text-muted">
                {isZh
                  ? '启动前会再次探测余额与交易权限。若失败，会打开对应交易所配置。'
                  : 'Preflight re-checks balance and trade auth before start. Failures deep-link into that exchange’s settings.'}
              </p>
              <div className="mt-3 font-mono text-xs text-strand-gold/90">
                {launchExchange.exchange_type}
                {exchangeState?.status
                  ? ` · ${exchangeState.status}`
                  : ''}
                {tradingBalance > 0
                  ? ` · ${formatUSDC(tradingBalance)} avail`
                  : ''}
              </div>
              <button
                type="button"
                onClick={() => onOpenExchangeConfig?.(launchExchange.id)}
                className="mt-4 inline-flex items-center gap-1.5 text-xs font-semibold text-strand-gold hover:text-strand-accent"
              >
                <Wallet className="h-3.5 w-3.5" />
                {isZh ? '打开交易所配置' : 'Open exchange settings'}
              </button>
            </div>
          )}
        </aside>
      </div>
    </section>
  )
}
