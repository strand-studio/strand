import { ConfirmDialogProvider } from './components/common/ConfirmDialog'
import { ThemeAwareToaster } from './components/common/ThemeAwareToaster'
import { AuthProvider } from './contexts/AuthContext'
import { LanguageProvider } from './contexts/LanguageContext'
import { ThemeProvider } from './contexts/ThemeContext'
import { AppRoutes } from './router/AppRoutes'

export default function App() {
  return (
    <ThemeProvider>
      <LanguageProvider>
        <AuthProvider>
          <ConfirmDialogProvider>
            <ThemeAwareToaster />
            <AppRoutes />
          </ConfirmDialogProvider>
        </AuthProvider>
      </LanguageProvider>
    </ThemeProvider>
  )
}