export type Page =
  | 'planning'
  | 'competition'
  | 'traders'
  | 'trader'
  | 'strategy'
  | 'strategy-market'
  | 'markets'
  | 'chat'
  | 'faq'
  | 'login'
  | 'register'

/**
 * Canonical paths match top-nav English labels:
 * Trade · Markets · Portfolio · Autopilot · Strategy · Leaderboard · Chat · Help
 * Legacy paths remain as redirect targets only.
 */
export const ROUTES = {
  planning: '/trade',
  markets: '/markets',
  portfolio: '/portfolio',
  autopilot: '/autopilot',
  strategy: '/strategy',
  leaderboard: '/leaderboard',
  chat: '/chat',
  help: '/help',
  settings: '/settings',
  home: '/',
  login: '/login',
  register: '/register',
  setup: '/setup',
  welcome: '/welcome',
  resetPassword: '/reset-password',
  strategyMarket: '/strategy-market',

  /** @deprecated aliases → canonical paths above */
  frontier: '/frontier',
  dashboard: '/portfolio',
  traders: '/autopilot',
  competition: '/leaderboard',
  faq: '/help',
  /** @deprecated use ROUTES.markets */
  data: '/data',

  /** literal legacy URLs used only for <Navigate> redirects */
  legacyDashboard: '/dashboard',
  legacyTraders: '/traders',
  legacyCompetition: '/competition',
  legacyFaq: '/faq',
} as const

export const PAGE_PATHS: Record<Page, string> = {
  planning: ROUTES.planning,
  competition: ROUTES.leaderboard,
  traders: ROUTES.autopilot,
  trader: ROUTES.portfolio,
  strategy: ROUTES.strategy,
  'strategy-market': ROUTES.strategyMarket,
  markets: ROUTES.markets,
  chat: ROUTES.chat,
  faq: ROUTES.help,
  login: ROUTES.login,
  register: ROUTES.register,
}

export const LEGACY_HASH_ROUTES: Record<string, string> = {
  competition: ROUTES.leaderboard,
  traders: ROUTES.autopilot,
  trader: ROUTES.portfolio,
  details: ROUTES.portfolio,
  strategy: ROUTES.strategy,
  'strategy-market': ROUTES.strategyMarket,
  data: ROUTES.markets,
  markets: ROUTES.markets,
  frontier: ROUTES.planning,
  planning: ROUTES.planning,
  dashboard: ROUTES.portfolio,
  faq: ROUTES.help,
}

export function buildDashboardPath(traderSlug?: string): string {
  if (!traderSlug) {
    return ROUTES.portfolio
  }

  return `${ROUTES.portfolio}?trader=${encodeURIComponent(traderSlug)}`
}

/** Markets detail: /markets/:exchange/:symbol */
export function buildMarketSymbolPath(exchange: string, symbol: string): string {
  return `${ROUTES.markets}/${encodeURIComponent(exchange || 'hyperliquid')}/${encodeURIComponent(symbol)}`
}

export function getCurrentPageForPath(pathname: string): Page | undefined {
  if (
    pathname === ROUTES.markets ||
    pathname === ROUTES.data ||
    pathname.startsWith(`${ROUTES.markets}/`)
  ) {
    return 'markets'
  }
  return getCurrentPageForPathExact(pathname)
}

function getCurrentPageForPathExact(pathname: string): Page | undefined {
  switch (pathname) {
    case ROUTES.planning:
    case ROUTES.frontier:
    case '/planning':
    case '/planning/':
      return 'planning'
    case ROUTES.welcome:
    case ROUTES.autopilot:
    case ROUTES.legacyTraders:
      return 'traders'
    case ROUTES.portfolio:
    case ROUTES.legacyDashboard:
      return 'trader'
    case ROUTES.strategy:
      return 'strategy'
    case ROUTES.strategyMarket:
      return 'strategy-market'
    case ROUTES.chat:
      return 'chat'
    case ROUTES.help:
    case ROUTES.legacyFaq:
      return 'faq'
    case ROUTES.login:
      return 'login'
    case ROUTES.register:
      return 'register'
    case ROUTES.leaderboard:
    case ROUTES.legacyCompetition:
      return 'competition'
    default:
      return undefined
  }
}
