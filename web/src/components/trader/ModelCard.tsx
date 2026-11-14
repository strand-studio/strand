import { Check } from 'lucide-react'
import type { AIModel } from '../../types'
import { getModelIcon } from '../common/ModelIcons'
import { getShortName } from './model-constants'

interface ModelCardProps {
  model: AIModel
  selected: boolean
  onClick: () => void
  configured?: boolean
}

export function ModelCard({
  model,
  selected,
  onClick,
  configured,
}: ModelCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex flex-col items-center gap-2 p-4 rounded-xl transition-all hover:scale-105"
      style={{
        background: selected ? 'var(--brand-soft)' : 'var(--panel-bg)',
        border: selected
          ? '2px solid var(--strand-gold)'
          : '2px solid var(--panel-border)',
      }}
    >
      <div className="relative">
        <div className="w-12 h-12 rounded-xl flex items-center justify-center bg-strand-bg-deeper border border-strand-border">
          {getModelIcon(model.provider || model.id, {
            width: 32,
            height: 32,
          }) || (
            <span className="text-lg font-bold" style={{ color: 'var(--strand-gold)' }}>
              {model.name[0]}
            </span>
          )}
        </div>
        {configured && (
          <div
            className="absolute -top-1 -right-1 w-4 h-4 rounded-full flex items-center justify-center"
            style={{ background: 'var(--strand-gold)' }}
          >
            <Check className="w-2.5 h-2.5 text-white" />
          </div>
        )}
      </div>
      <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
        {getShortName(model.name)}
      </span>
      <span
        className="text-[10px] px-2 py-0.5 rounded-full uppercase tracking-wide"
        style={{ background: 'rgba(41,91,232, 0.18)', color: 'var(--strand-gold)' }}
      >
        {model.provider}
      </span>
    </button>
  )
}
