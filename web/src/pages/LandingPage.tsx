import FlowLanding from '../components/landing/FlowLanding'
import { useState } from 'react'
import HeaderBar from '../components/common/HeaderBar'
import LoginModal from '../components/landing/LoginModal'
import FooterSection from '../components/landing/FooterSection'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'

export function LandingPage() {
  const [showLoginModal, setShowLoginModal] = useState(false)
  const { user, token, logout } = useAuth()
  const { language, setLanguage } = useLanguage()
  const isLoggedIn = Boolean(user && token)

  return (
    <>
      <HeaderBar
        onLoginClick={() => setShowLoginModal(true)}
        isLoggedIn={isLoggedIn}
        isHomePage={true}
        language={language}
        onLanguageChange={setLanguage}
        user={user}
        onLogout={logout}
      />
      <div className="min-h-screen bg-strand-bg text-strand-text font-sans selection:bg-strand-gold selection:text-strand-bg">
        <FlowLanding />

        <FooterSection language={language} />

        {showLoginModal && (
          <LoginModal
            onClose={() => setShowLoginModal(false)}
            language={language}
          />
        )}
      </div>
    </>
  )
}
