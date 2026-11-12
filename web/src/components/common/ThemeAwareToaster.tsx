import { Toaster } from 'sonner'
import { useTheme } from '../../contexts/ThemeContext'

export function ThemeAwareToaster() {
  const { resolvedTheme } = useTheme()
  return (
    <Toaster
      theme={resolvedTheme}
      closeButton
      position="top-center"
      duration={2200}
      toastOptions={{
        className: 'strand-toast',
      }}
    />
  )
}
