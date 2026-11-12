import type { UserMode } from '../../lib/onboarding'

interface OnboardingModeSelectorProps {
  language: string
  mode: UserMode
  onChange: (mode: UserMode) => void
}

export function OnboardingModeSelector({
  language,
  mode,
  onChange,
}: OnboardingModeSelectorProps) {
  const isZh = language === 'zh'

  const options: Array<{
    id: UserMode
    title: string
    badge?: string
    description: string
  }> = [
    {
      id: 'beginner',
      title: isZh ? 'Beginner Mode' : 'Beginner Mode',
      badge: isZh ? 'Recommended' : 'Recommended',
      description: isZh
        ? 'Generate a Base wallet automatically and start with Claw402 + GLM by default.'
        : 'Generate a Base wallet automatically and start with Claw402 + GLM by default.',
    },
    {
      id: 'advanced',
      title: isZh ? 'Advanced Mode' : 'Advanced Mode',
      description: isZh
        ? 'Keep the full manual flow and configure models, wallets, and exchanges yourself.'
        : 'Keep the full manual flow and configure models, wallets, and exchanges yourself.',
    },
  ]

  return (
    <div className="space-y-2">
      <div className="text-xs font-medium text-strand-text-muted">
        {isZh ? 'Experience' : 'Experience'}
      </div>
      <div className="grid grid-cols-1 gap-2">
        {options.map((option) => {
          const selected = option.id === mode
          return (
            <button
              key={option.id}
              type="button"
              onClick={() => onChange(option.id)}
              className={`w-full rounded-xl border px-4 py-3 text-left transition-all ${
                selected
                  ? 'border-strand-gold/60 bg-strand-gold/10'
                  : 'border-[rgba(25,36,52,0.14)] bg-strand-bg-lighter hover:border-strand-gold/40'
              }`}
            >
              <div className="flex items-center gap-2 text-sm font-semibold text-strand-text">
                <span>{option.title}</span>
                {option.badge ? (
                  <span className="rounded-full bg-strand-gold px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-strand-bg">
                    {option.badge}
                  </span>
                ) : null}
              </div>
              <p className="mt-1 text-xs leading-5 text-strand-text-muted">
                {option.description}
              </p>
            </button>
          )
        })}
      </div>
    </div>
  )
}
