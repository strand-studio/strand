/** Read a CSS custom property from the document root (theme-aware). */
export function cssVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback
  const value = getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim()
  return value || fallback
}

export function chartSurfaceColors() {
  return {
    background: cssVar('--background', '#f6f8fc'),
    text: cssVar('--text-primary', '#192434'),
    grid: cssVar('--panel-border', 'rgba(25,36,52,0.08)'),
    border: cssVar('--panel-border', 'rgba(25,36,52,0.14)'),
    accent: cssVar('--strand-gold', '#295be8'),
  }
}
