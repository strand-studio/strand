import { useLanguage } from '../contexts/LanguageContext'
import { t } from '../i18n/translations'

// Strategy Market — embedded vergex.trade/explore (legacy route helper).
// Markets live trading data is now on /markets (MarketsPage).
//
// vergex.trade may allow local Strand origins (e.g. localhost:3000) via
// Content-Security-Policy frame-ancestors so cross-origin embedding works.
export function StrategyMarketPage() {
  const { language } = useLanguage()

  return (
    <div className="h-[calc(100vh-64px)] w-full">
      <iframe
        src="https://vergex.trade/explore"
        title={t('strategyMarket', language) || 'Strategy Market'}
        className="h-full w-full border-0"
        // Permission policy: keep minimal. `clipboard-write` is intentionally
        // omitted so the iframe cannot silently overwrite the clipboard.
        allow="fullscreen"
        // Sandbox grants vergex.trade only what it actually needs to render
        // the explore page: run scripts, talk to its own origin / wallet
        // providers (allow-same-origin), submit search forms, open external
        // links in new tabs. Notably absent:
        //   - allow-top-navigation: prevents the iframe from navigating the
        //     parent Strand shell to an arbitrary URL.
        //   - allow-modals / allow-pointer-lock / allow-orientation-lock:
        //     not needed for a strategy list view.
        //   - allow-storage-access-by-user-activation: keeps any third-party
        //     storage access prompts out of the embedded surface.
        sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox"
        referrerPolicy="strict-origin-when-cross-origin"
      />
    </div>
  )
}
