import { useState, useEffect, useRef } from 'react'
import { useLocation, useNavigate } from 'react-router'
import {
  ChevronDown,
  Settings,
  LogOut,
  ExternalLink,
  Menu,
  Monitor,
  Moon,
  Sun,
  X,
} from 'lucide-react'
import { t, type Language } from '../../i18n/translations'
import { OFFICIAL_LINKS } from '../../constants/branding'
import { getCurrentPageForPath, ROUTES, type Page } from '../../router/paths'
import { useTheme } from '../../contexts/ThemeContext'
import { HyperliquidWalletConnect } from './HyperliquidWalletConnect'
import { BrandLogo } from './BrandLogo'

interface HeaderBarProps {
  onLoginClick?: () => void
  isLoggedIn?: boolean
  isHomePage?: boolean
  currentPage?: Page
  language?: Language
  onLanguageChange?: (lang: Language) => void
  user?: { email: string } | null
  onLogout?: () => void
  onPageChange?: (page: Page) => void
}

type NavTab = {
  page: Page
  path: string
  label: string
  auth: boolean
}

export default function HeaderBar({
  onLoginClick,
  isLoggedIn = false,
  currentPage,
  language = 'en',
  onLanguageChange,
  user,
  onLogout,
  onPageChange,
}: HeaderBarProps) {
  const { theme, resolvedTheme, cycleTheme } = useTheme()
  const navigate = useNavigate()
  const location = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)
  const [userOpen, setUserOpen] = useState(false)
  const [moreOpen, setMoreOpen] = useState(false)
  const userRef = useRef<HTMLDivElement>(null)
  const moreRef = useRef<HTMLDivElement>(null)
  const resolvedPage = currentPage ?? getCurrentPageForPath(location.pathname)

  // Primary IA: Markets → Trade → Portfolio first; secondary in More.
  const primaryTabs: NavTab[] = [
    {
      page: 'markets',
      path: ROUTES.markets,
      label: t('marketsNav', language),
      auth: false,
    },
    {
      page: 'planning',
      path: ROUTES.planning,
      label: t('tradeNav', language),
      auth: true,
    },
    {
      page: 'trader',
      path: ROUTES.portfolio,
      label: t('portfolioNav', language),
      auth: true,
    },
    {
      page: 'traders',
      path: ROUTES.autopilot,
      label: t('autopilotNav', language),
      // Guest sees launch guide; runners need login.
      auth: false,
    },
    {
      page: 'strategy',
      path: ROUTES.strategy,
      label: t('strategyNav', language),
      auth: true,
    },
    {
      page: 'competition',
      path: ROUTES.leaderboard,
      label: t('leaderboardNav', language),
      // Leaderboard is a public board (opt-in traders only).
      auth: false,
    },
  ]

  const moreTabs: NavTab[] = [
    {
      page: 'chat',
      path: ROUTES.chat,
      label: t('chatNav', language),
      auth: true,
    },
    {
      page: 'faq',
      path: ROUTES.help,
      label: t('helpNav', language),
      auth: false,
    },
  ]

  const moreActive = moreTabs.some((tab) => tab.page === resolvedPage)

  useEffect(() => {
    const close = (event: MouseEvent) => {
      const target = event.target as Node
      if (userRef.current && !userRef.current.contains(target)) setUserOpen(false)
      if (moreRef.current && !moreRef.current.contains(target)) setMoreOpen(false)
    }
    const escape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setUserOpen(false)
        setMoreOpen(false)
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', close)
    document.addEventListener('keydown', escape)
    return () => {
      document.removeEventListener('mousedown', close)
      document.removeEventListener('keydown', escape)
    }
  }, [])

  useEffect(() => {
    setMenuOpen(false)
    setUserOpen(false)
    setMoreOpen(false)
  }, [location.pathname])

  const open = (tab: NavTab) => {
    if (tab.auth && !isLoggedIn) {
      // Soft gate: send users to login and restore the tab after auth.
      // Avoid "ACCESS DENIED" overlays — needing an account is not an error.
      try {
        sessionStorage.setItem('returnUrl', tab.path)
      } catch {
        /* ignore quota / private mode */
      }
      navigate(ROUTES.login)
      setMoreOpen(false)
      setMenuOpen(false)
      return
    }
    onPageChange?.(tab.page)
    navigate(tab.path)
    setMoreOpen(false)
    setMenuOpen(false)
  }

  return (
    <nav className="header-bar strand-navigation" aria-label="Main navigation">
      <div className="strand-navigation-inner">
        <button
          className="strand-home"
          onClick={() => navigate(ROUTES.home)}
          aria-label="Strand home"
        >
          <BrandLogo />
        </button>

        <div className={`strand-nav-links lg:flex ${menuOpen ? 'is-open' : ''}`}>
          {primaryTabs.map((tab) => (
            <button
              key={tab.page}
              type="button"
              onClick={() => open(tab)}
              aria-current={resolvedPage === tab.page ? 'page' : undefined}
              className="strand-nav-link whitespace-nowrap"
            >
              {tab.label}
            </button>
          ))}

          <div className="strand-nav-more" ref={moreRef}>
            <button
              type="button"
              className="strand-nav-link strand-nav-more-trigger whitespace-nowrap"
              aria-expanded={moreOpen}
              aria-haspopup="menu"
              aria-controls="strand-more-menu"
              aria-current={moreActive ? 'page' : undefined}
              onClick={() => setMoreOpen((openState) => !openState)}
            >
              {t('moreNav', language)}
              <ChevronDown size={14} />
            </button>
            {moreOpen && (
              <div
                className="strand-more-menu"
                id="strand-more-menu"
                role="menu"
              >
                {moreTabs.map((tab) => (
                  <button
                    key={tab.page}
                    type="button"
                    role="menuitem"
                    aria-current={resolvedPage === tab.page ? 'page' : undefined}
                    onClick={() => open(tab)}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Mobile drawer also lists More destinations flat */}
          <div className="strand-nav-more-mobile">
            {moreTabs.map((tab) => (
              <button
                key={`mobile-${tab.page}`}
                type="button"
                onClick={() => open(tab)}
                aria-current={resolvedPage === tab.page ? 'page' : undefined}
                className="strand-nav-link whitespace-nowrap"
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        <div id="dash-header-slot" />

        <div className="strand-navigation-actions">
          <div
            className="strand-pref-controls"
            aria-label={t('preferences', language)}
          >
            <label className="sr-only" htmlFor="strand-lang-select">
              {t('language', language)}
            </label>
            <select
              id="strand-lang-select"
              className="strand-lang-select"
              value={language}
              aria-label={t('language', language)}
              onChange={(e) =>
                onLanguageChange?.(e.target.value as Language)
              }
            >
              <option value="en">EN</option>
              <option value="zh">中文</option>
              <option value="id">ID</option>
            </select>
            <button
              type="button"
              className="strand-theme-toggle"
              aria-label={
                theme === 'system'
                  ? t('themeSystem', language)
                  : resolvedTheme === 'dark'
                    ? t('themeDark', language)
                    : t('themeLight', language)
              }
              title={`${t('themeCycleHint', language)} (${
                theme === 'system'
                  ? t('themeSystem', language)
                  : resolvedTheme === 'dark'
                    ? t('themeDark', language)
                    : t('themeLight', language)
              })`}
              onClick={cycleTheme}
            >
              {theme === 'system' ? (
                <Monitor size={16} />
              ) : resolvedTheme === 'dark' ? (
                <Moon size={16} />
              ) : (
                <Sun size={16} />
              )}
            </button>
          </div>

          <div className="strand-nav-wallet">
            <HyperliquidWalletConnect
              language={language}
              isLoggedIn={isLoggedIn}
            />
          </div>

          {isLoggedIn && user ? (
            <div className="strand-user" ref={userRef}>
              <button
                className="strand-user-button"
                aria-label={`Account menu for ${user.email}`}
                aria-expanded={userOpen}
                aria-controls="strand-user-menu"
                onClick={() => setUserOpen(!userOpen)}
              >
                <span className="strand-user-avatar">
                  {user.email[0].toUpperCase()}
                </span>
                <ChevronDown size={14} />
              </button>
              {userOpen && (
                <div className="strand-user-menu" id="strand-user-menu">
                  <div className="strand-user-identity">
                    <small>Signed in as</small>
                    <strong>{user.email}</strong>
                  </div>
                  <button
                    onClick={() => {
                      navigate(ROUTES.settings)
                      setUserOpen(false)
                    }}
                  >
                    <Settings size={16} /> {t('settingsTitle', language)}
                  </button>
                  {[
                    OFFICIAL_LINKS.github
                      ? { name: 'GitHub', href: OFFICIAL_LINKS.github }
                      : null,
                    OFFICIAL_LINKS.twitter
                      ? { name: 'X / Twitter', href: OFFICIAL_LINKS.twitter }
                      : null,
                    OFFICIAL_LINKS.telegram
                      ? { name: 'Community', href: OFFICIAL_LINKS.telegram }
                      : null,
                  ]
                    .filter(Boolean)
                    .map((link) => (
                      <a
                        key={link!.name}
                        href={link!.href}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {link!.name}
                        <ExternalLink size={14} />
                      </a>
                    ))}
                  {onLogout && (
                    <button
                      className="strand-sign-out"
                      onClick={() => {
                        onLogout()
                        setUserOpen(false)
                      }}
                    >
                      <LogOut size={16} />
                      {t('exitLogin', language)}
                    </button>
                  )}
                </div>
              )}
            </div>
          ) : (
            resolvedPage !== 'login' &&
            resolvedPage !== 'register' && (
              <button
                className="strand-sign-in"
                onClick={() =>
                  onLoginClick ? onLoginClick() : navigate(ROUTES.login)
                }
              >
                {t('signIn', language)}
              </button>
            )
          )}

          <button
            className="strand-nav-toggle lg:hidden"
            aria-label={menuOpen ? 'Close navigation' : 'Open navigation'}
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen(!menuOpen)}
          >
            {menuOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
        </div>
      </div>
    </nav>
  )
}
