import { useMemo, useState } from 'react'
import { HelpCircle } from 'lucide-react'
import { DeepVoidBackground } from '../common/DeepVoidBackground'
import { FAQSearchBar } from './FAQSearchBar'
import { FAQSidebar } from './FAQSidebar'
import { FAQContent } from './FAQContent'
import { faqCategories, faqItemSearchText } from '../../data/faqData'
import type { FAQCategory } from '../../data/faqData'
import { OFFICIAL_LINKS } from '../../constants/branding'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'

export function FAQLayout() {
  const { language } = useLanguage()
  const [searchTerm, setSearchTerm] = useState('')
  const [activeItemId, setActiveItemId] = useState<string | null>(null)

  const filteredCategories = useMemo(() => {
    if (!searchTerm.trim()) return faqCategories

    const term = searchTerm.toLowerCase()
    const filtered: FAQCategory[] = []
    faqCategories.forEach((category) => {
      const matchingItems = category.items.filter((item) =>
        faqItemSearchText(item).includes(term)
      )
      if (matchingItems.length > 0) {
        filtered.push({ ...category, items: matchingItems })
      }
    })
    return filtered
  }, [searchTerm])

  const totalItems = useMemo(
    () =>
      faqCategories.reduce((sum, category) => sum + category.items.length, 0),
    []
  )

  const handleItemClick = (_categoryId: string, itemId: string) => {
    const element = document.getElementById(itemId)
    if (!element) return
    const offset = 104
    const top =
      element.getBoundingClientRect().top + window.pageYOffset - offset
    window.scrollTo({ top, behavior: 'smooth' })
  }

  return (
    <DeepVoidBackground className="py-8" disableAnimation>
      <div className="mx-auto w-full max-w-6xl px-4 md:px-8">
        {/* page header — shared workspace title and search */}
        <div className="mb-8 border-b border-strand-gold/20 pb-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl border border-strand-gold/30 bg-strand-bg-lighter text-strand-gold md:h-14 md:w-14">
                <HelpCircle className="h-6 w-6 md:h-7 md:w-7" />
              </div>
              <div>
                <h1 className="font-sans text-2xl font-bold tracking-tight text-strand-text md:text-3xl">
                  {t('pageHelpTitle', language)}
                </h1>
                <p className="mt-1 font-sans text-xs tracking-normal text-strand-text-muted">
                  {totalItems} {t('pageHelpDesc', language)}
                </p>
              </div>
            </div>
            <div className="w-full md:w-80">
              <FAQSearchBar
                searchTerm={searchTerm}
                onSearchChange={setSearchTerm}
              />
            </div>
          </div>
        </div>

        {/* content */}
        <div className="flex gap-8">
          <aside className="hidden w-64 shrink-0 lg:block">
            <FAQSidebar
              categories={filteredCategories}
              activeItemId={activeItemId}
              onItemClick={handleItemClick}
            />
          </aside>

          <main className="min-w-0 flex-1">
            {filteredCategories.length > 0 ? (
              <FAQContent
                categories={filteredCategories}
                onActiveItemChange={setActiveItemId}
              />
            ) : (
              <div className="rounded-xl border border-strand-gold/20 bg-strand-bg-lighter py-16 text-center">
                <p className="font-sans text-sm text-strand-text-muted">
                  {t('helpNoMatch', language, { term: searchTerm })}
                </p>
                <button
                  onClick={() => setSearchTerm('')}
                  className="mt-4 rounded-lg border border-strand-gold/30 bg-strand-gold/10 px-5 py-2 font-sans text-xs font-bold tracking-normal text-strand-gold hover:bg-strand-gold/20"
                >
                  {t('helpClearSearch', language)}
                </button>
              </div>
            )}
          </main>
        </div>

        {/* still stuck */}
        <div className="mt-12 rounded-xl border border-strand-gold/20 bg-strand-bg-lighter p-6 text-center md:p-8">
          <h3 className="font-sans text-sm font-bold tracking-normal text-strand-text">
            {t('helpStillTitle', language)}
          </h3>
          <p className="mt-2 text-sm text-strand-text-muted">
            {t('helpStillBody', language)}
          </p>
          <div className="mt-5 flex items-center justify-center gap-3">
            {OFFICIAL_LINKS.github ? (
              <a
                href={OFFICIAL_LINKS.github}
                target="_blank"
                rel="noopener noreferrer"
                className="rounded-lg border border-[rgba(25,36,52,0.14)] bg-strand-bg-deeper px-5 py-2.5 font-sans text-xs font-bold tracking-normal text-strand-text hover:border-strand-gold/40"
              >
                GitHub
              </a>
            ) : null}
            {OFFICIAL_LINKS.telegram ? (
              <a
                href={OFFICIAL_LINKS.telegram}
                target="_blank"
                rel="noopener noreferrer"
                className="rounded-lg bg-strand-gold px-5 py-2.5 font-sans text-xs font-bold tracking-normal text-white hover:bg-strand-accent"
              >
                {t('helpTelegramCta', language)}
              </a>
            ) : null}
          </div>
        </div>
      </div>
    </DeepVoidBackground>
  )
}
