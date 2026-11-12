import { BrandLogo } from '../common/BrandLogo'
import React, { useState, useEffect } from 'react'
import { Eye, EyeOff, Loader2, ArrowRight } from 'lucide-react'
import { useNavigate } from 'react-router'
import { toast } from 'sonner'
import { useAuth } from '../../contexts/AuthContext'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { DeepVoidBackground } from '../common/DeepVoidBackground'

export function LoginPage() {
  const { language } = useLanguage()
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [expiredToastId, setExpiredToastId] = useState<string | number | null>(
    null
  )

  useEffect(() => {
    // Only wipe local auth when we arrived here because a request returned 401.
    // Clearing on every /login visit made the UI look "logged in" then suddenly
    // "please sign in" after wallet/nav actions bounced the user here.
    if (sessionStorage.getItem('from401') === 'true') {
      localStorage.removeItem('auth_token')
      localStorage.removeItem('auth_user')
      localStorage.removeItem('user_id')
      const id = toast.warning(t('sessionExpired', language), {
        duration: Infinity,
      })
      setExpiredToastId(id)
      sessionStorage.removeItem('from401')
    }
  }, [language])

  // Account wipe was removed from the public API (it was an unauthenticated
  // destructive endpoint). It now runs as a local CLI command on the server,
  // so we surface the instruction instead of calling an endpoint.
  const handleResetAccount = () => {
    toast(t('resetAccountCliIntro', language), {
      description: 'strand reset-account',
      duration: 10000,
    })
  }

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    const result = await login(email, password)
    setLoading(false)
    if (result.success) {
      if (expiredToastId) toast.dismiss(expiredToastId)
    } else {
      const msg = result.message || t('loginFailed', language)
      setError(msg)
      toast.error(msg)
    }
  }

  return (
    <DeepVoidBackground className="strand-auth" disableAnimation>
      {/* Self-contained centering grid — works regardless of parent flex setup */}
      <main className="flex-1 grid lg:grid-cols-2">
        {/* ───────── LEFT: brand panel (desktop only) ───────── */}
        <section className="hidden lg:flex flex-col justify-between p-12 xl:p-16 relative overflow-hidden">
          {/* Soft brand background */}
          <div className="absolute -left-32 top-1/3 w-md h-112 bg-strand-gold/6 rounded-full blur-3xl pointer-events-none" />
          <div className="absolute -right-16 bottom-0 w-72 h-72 bg-strand-accent/4 rounded-full blur-3xl pointer-events-none" />

          {/* Brand mark */}
          <div className="flex items-center gap-3 relative">
            <BrandLogo />
          </div>

          {/* Headline */}
          <div className="relative max-w-lg">
            <div className="inline-flex items-center gap-2 mb-7 px-3 py-1 rounded-full border border-strand-gold/20 bg-strand-gold/6">
              <div className="w-1.5 h-1.5 rounded-full bg-strand-gold" />
              <span className="text-xs font-medium text-strand-gold">
                Your trading workspace
              </span>
            </div>
            <h2 className="text-4xl xl:text-5xl font-bold tracking-tight text-strand-text leading-[1.05]">
              {language === 'zh' ? (
                <>
                  自托管
                  <br />
                  <span className="text-strand-gold">AI 交易终端</span>
                </>
              ) : language === 'id' ? (
                <>
                  Terminal trading AI
                  <br />
                  <span className="text-strand-gold">yang Anda host sendiri</span>
                </>
              ) : (
                <>
                  Self-hosted
                  <br />
                  <span className="text-strand-gold">AI trading terminal</span>
                </>
              )}
            </h2>
            <p className="mt-5 text-strand-text-muted text-base leading-relaxed max-w-md">
              {language === 'zh'
                ? '自动驾驶跟随 Claw402 方向板；交易页对比计划后再确认。支持 Hyperliquid 与主流 CEX——密钥留在你自己的机器上。'
                : language === 'id'
                  ? 'Autopilot mengikuti papan Claw402; Trade membandingkan rencana sebelum Anda konfirmasi. Hyperliquid dan CEX utama — kunci tetap di mesin Anda.'
                  : 'Autopilot follows the Claw402 board; Trade compares plans before you confirm. Hyperliquid and major CEXs — keys stay on your machine.'}
            </p>
          </div>

          {/* Stats strip */}
          <div className="relative grid grid-cols-3 gap-8 max-w-md">
            <Stat
              value="10+"
              label={
                language === 'zh'
                  ? 'Exchanges'
                  : language === 'id'
                    ? 'Bursa'
                    : 'Exchanges'
              }
            />
            <Stat
              value="7"
              label={
                language === 'zh'
                  ? 'AI Models'
                  : language === 'id'
                    ? 'Model AI'
                    : 'AI Models'
              }
            />
            <Stat
              value="24/7"
              label={
                language === 'zh'
                  ? 'Always On'
                  : language === 'id'
                    ? 'Sepanjang Waktu'
                    : 'Always On'
              }
            />
          </div>
        </section>

        {/* ───────── RIGHT: form panel ───────── */}
        <section className="flex items-center justify-center p-6 sm:p-12 relative">
          <div className="w-full max-w-sm">
            {/* Mobile brand */}
            <div className="lg:hidden flex flex-col items-center gap-3 mb-10">
              <BrandLogo />
            </div>

            {/* Form header */}
            <div className="mb-7">
              <h1 className="text-[26px] sm:text-3xl font-bold tracking-tight text-strand-text">
                {t('signIn', language)}
              </h1>
              <p className="mt-1.5 text-sm text-strand-text-muted">
                {language === 'zh'
                  ? 'Continue with your email'
                  : language === 'id'
                    ? 'Lanjutkan dengan email Anda'
                    : 'Continue with your email'}
              </p>
            </div>

            {/* Form */}
            <form onSubmit={handleLogin} className="space-y-4">
              {/* Email */}
              <div>
                <label className="block text-[10.5px] font-medium uppercase tracking-[0.14em] text-strand-text-muted mb-2">
                  {t('email', language)}
                </label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full bg-strand-bg-lighter border border-[rgba(25,36,52,0.14)] rounded-lg px-4 py-[11px] text-[14px] text-strand-text placeholder-strand-text-muted focus:outline-hidden focus:border-strand-gold/50 focus:bg-strand-bg-lighter focus:ring-2 focus:ring-strand-gold/20 transition-all"
                  placeholder="you@example.com"
                  required
                  autoFocus
                  autoComplete="email"
                />
              </div>

              {/* Password */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-[10.5px] font-medium uppercase tracking-[0.14em] text-strand-text-muted">
                    {t('password', language)}
                  </label>
                  <button
                    type="button"
                    onClick={() => navigate('/reset-password')}
                    className="text-xs text-strand-text-muted hover:text-strand-gold transition-colors"
                  >
                    {t('forgotPassword', language)}
                  </button>
                </div>
                <div className="relative">
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full bg-strand-bg-lighter border border-[rgba(25,36,52,0.14)] rounded-lg px-4 py-[11px] pr-11 text-[14px] text-strand-text placeholder-strand-text-muted focus:outline-hidden focus:border-strand-gold/50 focus:bg-strand-bg-lighter focus:ring-2 focus:ring-strand-gold/20 transition-all"
                    placeholder="••••••••"
                    required
                    autoComplete="current-password"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-strand-text-muted hover:text-strand-text transition-colors"
                    aria-label={
                      showPassword ? 'Hide password' : 'Show password'
                    }
                  >
                    {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                  </button>
                </div>
              </div>

              {/* Error banner */}
              {error && (
                <div className="flex items-start gap-2 rounded-lg border border-strand-danger/25 bg-strand-danger/8 px-3 py-2.5 text-xs text-strand-danger">
                  <span className="text-strand-danger font-bold mt-px">!</span>
                  <span className="leading-relaxed">{error}</span>
                </div>
              )}

              {/* Submit */}
              <button
                type="submit"
                disabled={loading}
                className="group mt-2 flex w-full items-center justify-center gap-2 rounded-lg bg-strand-gold py-[11px] text-sm font-semibold text-strand-bg transition-all hover:bg-strand-gold-highlight active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {loading ? (
                  <>
                    <Loader2 size={16} className="animate-spin" />
                    {t('loggingIn', language) || 'Signing in...'}
                  </>
                ) : (
                  <>
                    {t('signIn', language)}
                    <ArrowRight
                      size={16}
                      className="transition-transform group-hover:translate-x-0.5"
                    />
                  </>
                )}
              </button>
            </form>

            {/* Footer */}
            <div className="mt-8 pt-5 border-t border-[rgba(25,36,52,0.14)] flex items-center justify-between text-[11px]">
              <span className="text-strand-text-muted">Local workspace</span>
              <button
                type="button"
                onClick={handleResetAccount}
                className="text-strand-text-muted transition-colors hover:text-strand-danger"
              >
                {t('forgotAccount', language)}
              </button>
            </div>
          </div>
        </section>
      </main>
    </DeepVoidBackground>
  )
}

function Stat({ value, label }: { value: string; label: string }) {
  return (
    <div>
      <div className="font-sans text-2xl xl:text-3xl font-bold text-strand-text">
        {value}
      </div>
      <div className="mt-1 text-[10.5px] uppercase tracking-[0.14em] text-strand-text-muted">
        {label}
      </div>
    </div>
  )
}
