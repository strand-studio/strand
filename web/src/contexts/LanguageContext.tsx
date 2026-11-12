import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import type { Language } from '../i18n/translations'

interface LanguageContextType {
  language: Language
  setLanguage: (language: Language) => void
}

const LanguageContext = createContext<LanguageContextType | undefined>(
  undefined
)

const STORAGE_KEY = 'language'

function readStoredLanguage(): Language {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (raw === 'zh' || raw === 'en' || raw === 'id') return raw
  // Prefer browser language on first visit
  const nav = (navigator.language || '').toLowerCase()
  if (nav.startsWith('zh')) return 'zh'
  if (nav.startsWith('id')) return 'id'
  return 'en'
}

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [language, setLanguageState] = useState<Language>(() =>
    readStoredLanguage()
  )

  const setLanguage = (lang: Language) => {
    localStorage.setItem(STORAGE_KEY, lang)
    setLanguageState(lang)
  }

  useEffect(() => {
    document.documentElement.lang =
      language === 'zh' ? 'zh-CN' : language === 'id' ? 'id' : 'en'
  }, [language])

  return (
    <LanguageContext.Provider value={{ language, setLanguage }}>
      {children}
    </LanguageContext.Provider>
  )
}

export function useLanguage() {
  const context = useContext(LanguageContext)
  if (!context) {
    throw new Error('useLanguage must be used within LanguageProvider')
  }
  return context
}
