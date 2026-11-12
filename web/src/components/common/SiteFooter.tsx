import { OFFICIAL_LINKS } from '../../constants/branding'
import { t, type Language } from '../../i18n/translations'
import { BrandLogo } from './BrandLogo'

export function SiteFooter({ language }: { language: Language }) {
  const links = [
    OFFICIAL_LINKS.github
      ? { label: 'GitHub ↗', href: OFFICIAL_LINKS.github }
      : null,
    OFFICIAL_LINKS.twitter
      ? { label: 'X / Twitter ↗', href: OFFICIAL_LINKS.twitter }
      : null,
    OFFICIAL_LINKS.telegram
      ? { label: 'Community ↗', href: OFFICIAL_LINKS.telegram }
      : null,
  ].filter(Boolean) as { label: string; href: string }[]

  return (
    <footer className="strand-site-footer">
      <div className="strand-site-footer-inner">
        <BrandLogo />
        <div>
          <p>{t('footerTitle', language)}</p>
          <p>{t('footerWarning', language)}</p>
        </div>
        {links.length > 0 ? (
          <div className="strand-site-footer-links">
            {links.map((link) => (
              <a
                key={link.href}
                href={link.href}
                target="_blank"
                rel="noopener noreferrer"
              >
                {link.label}
              </a>
            ))}
          </div>
        ) : null}
      </div>
    </footer>
  )
}
