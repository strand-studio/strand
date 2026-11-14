import { useState } from 'react'
import { useNavigate } from 'react-router'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { Header } from '../common/Header'
import { ArrowLeft, KeyRound, Copy, Check } from 'lucide-react'
import { toast } from 'sonner'

const RESET_PASSWORD_COMMAND = 'strand reset-password --email you@example.com'

export function ResetPasswordPage() {
  const { language } = useLanguage()
  const navigate = useNavigate()
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(RESET_PASSWORD_COMMAND)
      setCopied(true)
      toast.success(t('copy', language))
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error(t('copy', language))
    }
  }

  return (
    <div className="min-h-screen bg-strand-bg text-strand-text">
      <Header simple />

      <div
        className="flex items-center justify-center"
        style={{ minHeight: 'calc(100vh - 80px)' }}
      >
        <div className="w-full max-w-md px-4">
          <button
            onClick={() => navigate('/login')}
            className="flex items-center gap-2 mb-6 text-sm text-strand-text-muted hover:text-strand-gold transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            {t('backToLogin', language)}
          </button>

          <div className="text-center mb-8">
            <div className="w-16 h-16 mx-auto mb-4 flex items-center justify-center rounded-full bg-strand-gold/10">
              <KeyRound className="w-8 h-8 text-strand-gold" />
            </div>
            <h1 className="text-2xl font-bold text-strand-text">
              {t('resetPasswordTitle', language)}
            </h1>
          </div>

          <div className="rounded-lg p-6 bg-strand-bg-lighter border border-strand-border">
            <p className="text-sm leading-relaxed mb-4 text-strand-text">
              {t('resetPasswordCliIntro', language)}
            </p>

            <div className="flex items-center justify-between gap-3 rounded-sm px-3 py-3 font-mono text-xs bg-strand-bg-deeper border border-strand-border">
              <code className="break-all text-strand-gold">
                {RESET_PASSWORD_COMMAND}
              </code>
              <button
                type="button"
                onClick={handleCopy}
                className="shrink-0 btn-icon text-strand-text-muted"
                aria-label={t('copy', language)}
              >
                {copied ? (
                  <Check className="w-4 h-4 text-strand-success" />
                ) : (
                  <Copy className="w-4 h-4" />
                )}
              </button>
            </div>

            <p className="text-xs leading-relaxed mt-4 text-strand-text-muted">
              {t('resetPasswordCliSecurityNote', language)}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
