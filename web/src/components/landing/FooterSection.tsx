import { ExternalLink } from 'lucide-react'
import { t, type Language } from '../../i18n/translations'
import { OFFICIAL_LINKS } from '../../constants/branding'
import { BrandLogo } from '../common/BrandLogo'

interface FooterSectionProps {
  language: Language
}

export default function FooterSection({ language }: FooterSectionProps) {
  const links = {
    social: [
      { name: 'GitHub', href: OFFICIAL_LINKS.github },
      { name: 'X / Twitter', href: OFFICIAL_LINKS.twitter },
      { name: 'Telegram', href: OFFICIAL_LINKS.telegram },
    ],
    resources: [
      { name: t('marketsNav', language), href: '/markets', external: false },
      { name: t('helpNav', language), href: '/help', external: false },
      { name: t('leaderboardNav', language), href: '/leaderboard', external: false },
      { name: 'Source', href: OFFICIAL_LINKS.github, external: true },
    ],
    supporters: [
      {
        name: 'Aster DEX',
        href: 'https://www.asterdex.com/en/referral/fdfc0e',
      },
      { name: 'Lighter', href: 'https://app.lighter.xyz/?referral=68151432' },
    ],
  }

  return (
    <footer className="bg-strand-bg border-t border-strand-border/40">
      <div className="max-w-6xl mx-auto px-4 py-8 md:py-12">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8 md:gap-10 mb-8 md:mb-12">
          <div className="md:col-span-1">
            <div className="flex items-center gap-3 mb-4">
              <BrandLogo />
            </div>
            <p className="text-sm mb-6 text-strand-text-muted">
              {t('futureStandardAI', language)}
            </p>
            <div className="flex flex-wrap items-center gap-2">
              {links.social.map((link) => (
                <a
                  key={link.name}
                  href={link.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs border border-strand-border/40 bg-strand-bg-lighter text-strand-text-muted rounded-sm px-3 py-1.5 transition-all hover:border-strand-gold hover:text-strand-gold"
                >
                  {link.name}
                </a>
              ))}
            </div>
          </div>

          <div>
            <h4 className="text-sm font-semibold mb-4 text-strand-text">
              {t('links', language)}
            </h4>
            <ul className="space-y-3">
              {links.social.map((link) => (
                <li key={link.name}>
                  <a
                    href={link.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm text-strand-text-muted transition-colors hover:text-strand-gold"
                  >
                    {link.name}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h4 className="text-sm font-semibold mb-4 text-strand-text">
              {t('resources', language)}
            </h4>
            <ul className="space-y-3">
              {links.resources.map((link) => (
                <li key={link.name}>
                  <a
                    href={link.href}
                    {...(link.external
                      ? { target: '_blank', rel: 'noopener noreferrer' }
                      : {})}
                    className="text-sm text-strand-text-muted transition-colors hover:text-strand-gold inline-flex items-center gap-1"
                  >
                    {link.name}
                    {link.external ? (
                      <ExternalLink className="w-3 h-3 opacity-50" />
                    ) : null}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h4 className="text-sm font-semibold mb-4 text-strand-text">
              {t('supporters', language)}
            </h4>
            <div className="flex flex-wrap gap-2">
              {links.supporters.map((link) => (
                <a
                  key={link.name}
                  href={link.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs border border-strand-border/40 bg-strand-bg-lighter text-strand-text-muted rounded-sm px-3 py-1.5 transition-all hover:border-strand-gold hover:text-strand-gold hover:bg-strand-gold/10"
                >
                  {link.name}
                </a>
              ))}
            </div>
          </div>
        </div>

        <div className="pt-6 text-center text-xs text-strand-text-muted border-t border-strand-border/40">
          <p className="mb-2">{t('footerTitle', language)}</p>
          <p className="text-strand-text-muted/70">
            {t('footerWarning', language)}
          </p>
        </div>
      </div>
    </footer>
  )
}
