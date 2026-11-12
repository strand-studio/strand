import { motion, AnimatePresence } from 'framer-motion'
import { LogIn, UserPlus, X } from 'lucide-react'
import { DeepVoidBackground } from '../common/DeepVoidBackground'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { Link } from 'react-router'

interface LoginRequiredOverlayProps {
  isOpen: boolean
  onClose: () => void
  featureName?: string
}

/** Soft sign-in prompt. Prefer navigating to /login with returnUrl for nav gates. */
export function LoginRequiredOverlay({
  isOpen,
  onClose,
  featureName,
}: LoginRequiredOverlayProps) {
  const { language } = useLanguage()

  const tr = (key: string, params?: Record<string, string | number>) =>
    t(`loginRequired.${key}`, language, params)

  const subtitle = featureName
    ? tr('subtitleWithFeature', { featureName })
    : tr('subtitleDefault')

  const benefits = [tr('benefit1'), tr('benefit2'), tr('benefit4')]

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-50"
        >
          <DeepVoidBackground
            className="w-full h-full bg-strand-bg/95 backdrop-blur-md flex items-center justify-center p-4 text-strand-text"
            disableAnimation
            onClick={onClose}
          >
            <motion.div
              initial={{ opacity: 0, scale: 0.96, y: 8 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.96, y: 8 }}
              transition={{ type: 'spring', damping: 22, stiffness: 320 }}
              className="relative max-w-md w-full overflow-hidden bg-strand-bg-lighter border border-strand-border shadow-lg rounded-sm"
              onClick={(e) => e.stopPropagation()}
              role="dialog"
              aria-modal="true"
              aria-labelledby="login-required-title"
            >
              <div className="flex items-center justify-between px-4 py-3 border-b border-strand-border">
                <span className="text-xs text-strand-text-muted tracking-wide">
                  {tr('badge')}
                </span>
                <button
                  type="button"
                  onClick={onClose}
                  className="text-strand-text-muted hover:text-strand-text transition-colors"
                  aria-label={tr('abort')}
                >
                  <X size={16} />
                </button>
              </div>

              <div className="p-6 sm:p-8">
                <div className="space-y-3 mb-6">
                  <h2
                    id="login-required-title"
                    className="text-xl font-semibold text-strand-text tracking-tight"
                  >
                    {tr('title')}
                  </h2>
                  <p className="text-sm text-strand-text-secondary leading-relaxed">
                    {subtitle}
                  </p>
                  <p className="text-sm text-strand-text-muted leading-relaxed">
                    {tr('description')}
                  </p>
                </div>

                <ul className="grid gap-2 mb-6">
                  {benefits.map((benefit) => (
                    <li
                      key={benefit}
                      className="flex items-start gap-2 text-sm text-strand-text-secondary"
                    >
                      <span className="text-strand-gold mt-0.5" aria-hidden>
                        ✓
                      </span>
                      <span>{benefit}</span>
                    </li>
                  ))}
                </ul>

                <div className="space-y-3">
                  <Link
                    to="/login"
                    className="flex items-center justify-center gap-2 w-full py-3 bg-strand-gold text-strand-bg font-semibold text-sm hover:bg-strand-gold-highlight transition-colors"
                  >
                    <LogIn size={16} />
                    <span>{tr('loginButton')}</span>
                  </Link>

                  <Link
                    to="/register"
                    className="flex items-center justify-center gap-2 w-full py-3 border border-strand-border text-strand-text-secondary hover:text-strand-text hover:border-strand-gold/40 font-medium text-sm transition-colors"
                  >
                    <UserPlus size={16} />
                    <span>{tr('registerButton')}</span>
                  </Link>
                </div>

                <div className="mt-4 text-center">
                  <button
                    type="button"
                    onClick={onClose}
                    className="text-xs text-strand-text-muted hover:text-strand-text underline-offset-2 hover:underline"
                  >
                    {tr('abort')}
                  </button>
                </div>
              </div>
            </motion.div>
          </DeepVoidBackground>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
