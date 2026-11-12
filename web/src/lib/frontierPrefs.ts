const KEY = 'strand.frontier.prefs.v1'

export type FrontierPrefs = {
  venue?: string
  symbol?: string
  interval?: string
  watchlist: string[] // `${venue}:${symbol}`
}

const defaults: FrontierPrefs = { watchlist: [] }

export function loadFrontierPrefs(): FrontierPrefs {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return { ...defaults, watchlist: [] }
    const parsed = JSON.parse(raw) as Partial<FrontierPrefs>
    return {
      venue: parsed.venue,
      symbol: parsed.symbol,
      interval: parsed.interval,
      watchlist: Array.isArray(parsed.watchlist) ? parsed.watchlist : [],
    }
  } catch {
    return { ...defaults, watchlist: [] }
  }
}

export function saveFrontierPrefs(next: FrontierPrefs) {
  localStorage.setItem(KEY, JSON.stringify(next))
}

export function watchKey(venue: string, symbol: string) {
  return `${venue}:${symbol}`
}

export function toggleWatch(
  prefs: FrontierPrefs,
  venue: string,
  symbol: string
): FrontierPrefs {
  const key = watchKey(venue, symbol)
  const has = prefs.watchlist.includes(key)
  const watchlist = has
    ? prefs.watchlist.filter((k) => k !== key)
    : [...prefs.watchlist, key]
  const next = { ...prefs, watchlist }
  saveFrontierPrefs(next)
  return next
}

export function rememberMarket(
  prefs: FrontierPrefs,
  venue: string,
  symbol: string,
  interval?: string
): FrontierPrefs {
  const next = {
    ...prefs,
    venue,
    symbol,
    interval: interval || prefs.interval,
  }
  saveFrontierPrefs(next)
  return next
}
