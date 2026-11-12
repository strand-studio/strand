import { useState } from 'react'
import { Link } from 'react-router'
import { Trophy } from 'lucide-react'
import useSWR from 'swr'
import { api } from '../../lib/api'
import type { CompetitionData } from '../../types'
import { ComparisonChart } from '../charts/ComparisonChart'
import { TraderConfigViewModal } from './TraderConfigViewModal'
import { getTraderColor } from '../../utils/traderColors'
import { useAuth } from '../../contexts/AuthContext'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { PunkAvatar, getTraderAvatar } from '../common/PunkAvatar'
import { DeepVoidBackground } from '../common/DeepVoidBackground'
import { ROUTES } from '../../router/paths'

export function CompetitionPage() {
  const { language } = useLanguage()
  const { user, token } = useAuth()
  const isLoggedIn = Boolean(user && token)
  const [selectedTrader, setSelectedTrader] = useState<any>(null)
  const [isModalOpen, setIsModalOpen] = useState(false)

  const { data: competition } = useSWR<CompetitionData>(
    'competition',
    api.getCompetition,
    {
      refreshInterval: 15000, // Refresh every 15s (competition data does not need very frequent updates)
      revalidateOnFocus: false,
      dedupingInterval: 10000,
    }
  )

  const handleTraderClick = async (traderId: string) => {
    try {
      const traderConfig = await api.getPublicTraderConfig(traderId)
      setSelectedTrader(traderConfig)
      setIsModalOpen(true)
    } catch (error) {
      console.error('Failed to fetch trader config:', error)
      // For logged-out users we do not show detailed config; this is expected behavior
      // The competition page is mainly for viewing the leaderboard and basic info
    }
  }

  const closeModal = () => {
    setIsModalOpen(false)
    setSelectedTrader(null)
  }

  const rulesBanner = competition?.rules ? (
    <div className="mb-4 rounded-xl border border-strand-gold/25 bg-strand-gold/10 px-4 py-3 text-sm text-strand-text">
      <div className="font-semibold text-strand-gold">
        {language === 'zh' ? '公平竞赛规则' : 'Fair-play rules'}
      </div>
      <p className="mt-1 text-xs leading-5 text-strand-text-muted">
        {competition.rules.description ||
          (language === 'zh'
            ? `参考初始资金 $${competition.rules.reference_initial_usd ?? 1000} · 计分 ${competition.rules.scoring || 'pnl_pct'}`
            : `Reference initial $${competition.rules.reference_initial_usd ?? 1000} · scoring ${competition.rules.scoring || 'pnl_pct'}`)}
        {competition.rules.start_at
          ? ` · window ${competition.rules.start_at}${competition.rules.end_at ? ` → ${competition.rules.end_at}` : ''}`
          : ''}
      </p>
    </div>
  ) : null

  if (!competition) {
    return (
      <DeepVoidBackground className="py-8" disableAnimation>
        <div className="container mx-auto max-w-7xl px-4 md:px-8">
          <div className="space-y-6">
            <div className="animate-pulse bg-strand-bg-lighter border border-strand-gold/20 rounded-xl p-8">
              <div className="flex items-center justify-between mb-6">
                <div className="space-y-3 flex-1">
                  <div className="h-8 w-64 bg-strand-bg-deeper rounded-sm"></div>
                  <div className="h-4 w-48 bg-strand-bg-deeper rounded-sm"></div>
                </div>
                <div className="h-12 w-32 bg-strand-bg-deeper rounded-sm"></div>
              </div>
            </div>
            <div className="bg-strand-bg-lighter border border-strand-gold/20 rounded-xl p-6">
              <div className="h-6 w-40 mb-4 bg-strand-bg-deeper rounded-sm"></div>
              <div className="space-y-3">
                <div className="h-20 w-full bg-strand-bg-deeper rounded-sm"></div>
                <div className="h-20 w-full bg-strand-bg-deeper rounded-sm"></div>
              </div>
            </div>
          </div>
        </div>
      </DeepVoidBackground>
    )
  }

  // If data returns but there are no traders, show the empty state
  if (!competition.traders || competition.traders.length === 0) {
    return (
      <DeepVoidBackground className="py-8" disableAnimation>
        <div className="container mx-auto max-w-7xl px-4 md:px-8 space-y-8 animate-fade-in">
          {/* Competition Header - compact */}
          <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-3 md:gap-0">
            <div className="flex items-center gap-3 md:gap-4">
              <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl flex items-center justify-center bg-strand-bg-lighter border border-strand-gold/30">
                <Trophy className="w-6 h-6 md:w-7 md:h-7 text-strand-gold" />
              </div>
              <div>
                <span className="mb-1 block text-[10px] font-semibold tracking-[0.14em] text-strand-gold">
                  {t('pageLeaderboardEyebrow', language)}
                </span>
                <h1 className="text-xl md:text-2xl font-bold flex items-center gap-2 text-strand-text">
                  {t('pageLeaderboardTitle', language)}
                  <span className="text-xs font-normal px-2 py-1 rounded-sm bg-strand-gold/10 text-strand-gold border border-strand-gold/20">
                    0 {t('traders', language)}
                  </span>
                </h1>
                <p className="text-xs text-strand-text-muted max-w-xl">
                  {t('pageLeaderboardDesc', language)}
                </p>
              </div>
            </div>
          </div>

          {rulesBanner}

          {/* Empty State */}
          <div className="bg-strand-bg-lighter border border-strand-gold/20 rounded-xl p-16 text-center">
            <Trophy className="w-16 h-16 mx-auto mb-4 text-strand-text-muted" />
            <h3 className="text-lg font-bold mb-2 text-strand-text">
              {t('noTraders', language)}
            </h3>
            <p className="text-sm text-strand-text-muted mb-6">
              {t('createFirstTraderLeaderboard', language)}
            </p>
            <Link
              to={isLoggedIn ? ROUTES.autopilot : ROUTES.login}
              onClick={() => {
                if (!isLoggedIn) {
                  try {
                    sessionStorage.setItem('returnUrl', ROUTES.autopilot)
                  } catch {
                    /* ignore */
                  }
                }
              }}
              className="inline-flex items-center justify-center rounded-sm bg-strand-gold px-4 py-2 text-sm font-semibold text-strand-bg hover:bg-strand-gold-highlight"
            >
              {isLoggedIn
                ? t('autopilotNav', language)
                : language === 'zh'
                  ? '登录后启动 Autopilot'
                  : language === 'id'
                    ? 'Masuk untuk mulai Autopilot'
                    : 'Sign in to start Autopilot'}
            </Link>
          </div>
        </div>
      </DeepVoidBackground>
    )
  }

  // Sort by return percentage
  const sortedTraders = [...competition.traders].sort(
    (a, b) => b.total_pnl_pct - a.total_pnl_pct
  )

  // Find the leader
  const leader = sortedTraders[0]

  return (
    <DeepVoidBackground className="py-8" disableAnimation>
      <div className="w-full px-4 md:px-8 space-y-8 animate-fade-in">
        {/* Competition Header - compact */}
        <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-3 md:gap-0">
          <div className="flex items-center gap-3 md:gap-4">
            <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl flex items-center justify-center bg-strand-bg-lighter border border-strand-gold/30">
              <Trophy className="w-6 h-6 md:w-7 md:h-7 text-strand-gold" />
            </div>
            <div>
              <span className="mb-1 block text-[10px] font-semibold tracking-[0.14em] text-strand-gold">
                {t('pageLeaderboardEyebrow', language)}
              </span>
              <h1 className="text-xl md:text-2xl font-bold flex items-center gap-2 text-strand-text">
                {t('pageLeaderboardTitle', language)}
                <span className="text-xs font-normal px-2 py-1 rounded-sm bg-strand-gold/10 text-strand-gold border border-strand-gold/20">
                  {competition.count} {t('traders', language)}
                </span>
              </h1>
              <p className="text-xs text-strand-text-muted max-w-xl">
                {t('pageLeaderboardDesc', language)}
              </p>
            </div>
          </div>
          <div className="text-left md:text-right w-full md:w-auto">
            <div className="text-xs mb-1 text-strand-text-muted">
              {t('leader', language)}
            </div>
            <div className="text-base md:text-lg font-bold text-strand-gold">
              {leader?.trader_name}
            </div>
            <div
              className="text-sm font-semibold"
              style={{
                color: (leader?.total_pnl ?? 0) >= 0 ? '#2E8B57' : '#D6433A',
              }}
            >
              {(leader?.total_pnl ?? 0) >= 0 ? '+' : ''}
              {leader?.total_pnl_pct?.toFixed(2) || '0.00'}%
            </div>
          </div>
        </div>

        {rulesBanner}

        {/* Left/Right Split: Performance Chart + Leaderboard */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Left: Performance Comparison Chart */}
          <div
            className="bg-strand-bg-lighter border border-strand-gold/20 rounded-xl p-6 animate-slide-in hover:border-strand-gold/40 transition-colors"
            style={{ animationDelay: '0.1s' }}
          >
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-lg font-bold flex items-center gap-2 text-strand-text">
                {t('performanceComparison', language)}
              </h2>
              <div className="text-xs text-strand-text-muted">
                {t('realTimePnL', language)}
              </div>
            </div>
            <ComparisonChart traders={sortedTraders.slice(0, 10)} />
          </div>

          {/* Right: Leaderboard */}
          <div
            className="bg-strand-bg-lighter border border-strand-gold/20 rounded-xl p-6 animate-slide-in hover:border-strand-gold/40 transition-colors"
            style={{ animationDelay: '0.1s' }}
          >
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-lg font-bold flex items-center gap-2 text-strand-text">
                {t('leaderboard', language)}
              </h2>
              <div className="text-xs px-2 py-1 rounded-sm bg-strand-gold/10 text-strand-gold border border-strand-gold/20">
                {t('live', language)}
              </div>
            </div>
            <div className="space-y-2">
              {sortedTraders.map((trader, index) => {
                const isLeader = index === 0
                const traderColor = getTraderColor(
                  sortedTraders,
                  trader.trader_id
                )

                return (
                  <div
                    key={trader.trader_id}
                    onClick={() => handleTraderClick(trader.trader_id)}
                    className="rounded-sm p-3 transition-all duration-300 hover:-translate-y-px cursor-pointer hover:shadow-lg"
                    style={{
                      background: isLeader
                        ? 'rgba(41,91,232, 0.08)'
                        : '#ffffff',
                      border: `1px solid ${isLeader ? 'rgba(41,91,232, 0.4)' : 'rgba(25,36,52,0.14)'}`,
                      boxShadow: isLeader
                        ? '0 0 0 1px rgba(41,91,232, 0.15)'
                        : 'none',
                    }}
                  >
                    <div className="flex items-center justify-between">
                      {/* Rank & Avatar & Name */}
                      <div className="flex items-center gap-3">
                        {/* Rank Badge */}
                        <div
                          className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                          style={{
                            background:
                              index === 0
                                ? '#295be8'
                                : index === 1
                                  ? 'linear-gradient(135deg, #C0C0C0 0%, #E8E8E8 100%)'
                                  : index === 2
                                    ? 'linear-gradient(135deg, #CD7F32 0%, #E8A64C 100%)'
                                    : '#edf1f7',
                            color:
                              index === 0
                                ? '#fff'
                                : index < 3
                                  ? '#000'
                                  : '#707e92',
                          }}
                        >
                          {index + 1}
                        </div>
                        {/* Punk Avatar */}
                        <PunkAvatar
                          seed={getTraderAvatar(
                            trader.trader_id,
                            trader.trader_name
                          )}
                          size={36}
                          className="rounded-lg"
                        />
                        <div>
                          <div
                            className="font-bold text-sm"
                            style={{ color: '#192434' }}
                          >
                            {trader.trader_name}
                          </div>
                          <div
                            className="text-xs mono font-semibold"
                            style={{ color: traderColor }}
                          >
                            {trader.ai_model.toUpperCase()} +{' '}
                            {trader.exchange.toUpperCase()}
                          </div>
                        </div>
                      </div>

                      {/* Stats */}
                      <div className="flex items-center gap-2 md:gap-3 flex-wrap md:flex-nowrap">
                        {/* Total Equity */}
                        <div className="text-right">
                          <div className="text-xs" style={{ color: '#707e92' }}>
                            {t('equity', language)}
                          </div>
                          <div
                            className="text-xs md:text-sm font-bold mono"
                            style={{ color: '#192434' }}
                          >
                            {trader.total_equity?.toFixed(2) || '0.00'}
                          </div>
                        </div>

                        {/* P&L */}
                        <div className="text-right min-w-[70px] md:min-w-[90px]">
                          <div className="text-xs" style={{ color: '#707e92' }}>
                            {t('pnl', language)}
                          </div>
                          <div
                            className="text-base md:text-lg font-bold mono"
                            style={{
                              color:
                                (trader.total_pnl ?? 0) >= 0
                                  ? '#2E8B57'
                                  : '#D6433A',
                            }}
                          >
                            {(trader.total_pnl ?? 0) >= 0 ? '+' : ''}
                            {trader.total_pnl_pct?.toFixed(2) || '0.00'}%
                          </div>
                          <div
                            className="text-xs mono"
                            style={{ color: '#707e92' }}
                          >
                            {(trader.total_pnl ?? 0) >= 0 ? '+' : ''}
                            {trader.total_pnl?.toFixed(2) || '0.00'}
                          </div>
                        </div>

                        {/* Positions */}
                        <div className="text-right">
                          <div className="text-xs" style={{ color: '#707e92' }}>
                            {t('pos', language)}
                          </div>
                          <div
                            className="text-xs md:text-sm font-bold mono"
                            style={{ color: '#192434' }}
                          >
                            {trader.position_count}
                          </div>
                          <div className="text-xs" style={{ color: '#707e92' }}>
                            {trader.margin_used_pct.toFixed(1)}%
                          </div>
                        </div>

                        {/* Status */}
                        <div>
                          <div
                            className="px-2 py-1 rounded-sm text-xs font-bold"
                            style={
                              trader.is_running
                                ? {
                                    background: 'rgba(46, 139, 87, 0.1)',
                                    color: '#2E8B57',
                                  }
                                : {
                                    background: 'rgba(214, 67, 58, 0.1)',
                                    color: '#D6433A',
                                  }
                            }
                          >
                            {trader.is_running ? '●' : '○'}
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        </div>

        {/* Head-to-Head Stats */}
        {competition.traders.length === 2 && (
          <div
            className="bg-strand-bg-lighter border border-strand-gold/20 rounded-xl p-6 animate-slide-in"
            style={{ animationDelay: '0.3s' }}
          >
            <h2 className="text-lg font-bold mb-6 flex items-center gap-2 text-strand-text">
              {t('headToHead', language)}
            </h2>
            <div className="grid grid-cols-2 gap-4">
              {sortedTraders.map((trader, index) => {
                const isWinning = index === 0
                const opponent = sortedTraders[1 - index]

                // Check if both values are valid numbers
                const hasValidData =
                  trader.total_pnl_pct != null &&
                  opponent.total_pnl_pct != null &&
                  !isNaN(trader.total_pnl_pct) &&
                  !isNaN(opponent.total_pnl_pct)

                const gap = hasValidData
                  ? trader.total_pnl_pct - opponent.total_pnl_pct
                  : NaN

                return (
                  <div
                    key={trader.trader_id}
                    className="p-4 rounded-sm transition-all duration-300 hover:scale-[1.02]"
                    style={
                      isWinning
                        ? {
                            background: 'rgba(46, 139, 87, 0.08)',
                            border: '2px solid rgba(46, 139, 87, 0.3)',
                            boxShadow: 'none',
                          }
                        : {
                            background: '#ffffff',
                            border: '1px solid rgba(25,36,52,0.14)',
                            boxShadow: 'none',
                          }
                    }
                  >
                    <div className="text-center">
                      {/* Avatar */}
                      <div className="flex justify-center mb-3">
                        <PunkAvatar
                          seed={getTraderAvatar(
                            trader.trader_id,
                            trader.trader_name
                          )}
                          size={56}
                          className="rounded-xl"
                        />
                      </div>
                      <div
                        className="text-sm md:text-base font-bold mb-2"
                        style={{
                          color: getTraderColor(
                            sortedTraders,
                            trader.trader_id
                          ),
                        }}
                      >
                        {trader.trader_name}
                      </div>
                      <div
                        className="text-lg md:text-2xl font-bold mono mb-1"
                        style={{
                          color:
                            (trader.total_pnl ?? 0) >= 0
                              ? '#2E8B57'
                              : '#D6433A',
                        }}
                      >
                        {trader.total_pnl_pct != null &&
                        !isNaN(trader.total_pnl_pct)
                          ? `${trader.total_pnl_pct >= 0 ? '+' : ''}${trader.total_pnl_pct.toFixed(2)}%`
                          : '—'}
                      </div>
                      {hasValidData && isWinning && gap > 0 && (
                        <div
                          className="text-xs font-semibold"
                          style={{ color: '#2E8B57' }}
                        >
                          {t('leadingBy', language, { gap: gap.toFixed(2) })}
                        </div>
                      )}
                      {hasValidData && !isWinning && gap < 0 && (
                        <div
                          className="text-xs font-semibold"
                          style={{ color: '#D6433A' }}
                        >
                          {t('behindBy', language, {
                            gap: Math.abs(gap).toFixed(2),
                          })}
                        </div>
                      )}
                      {!hasValidData && (
                        <div
                          className="text-xs font-semibold"
                          style={{ color: '#707e92' }}
                        >
                          —
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* Trader Config View Modal */}
        <TraderConfigViewModal
          isOpen={isModalOpen}
          onClose={closeModal}
          traderData={selectedTrader}
        />
      </div>
    </DeepVoidBackground>
  )
}
