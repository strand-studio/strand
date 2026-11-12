interface BrandLogoProps {
  compact?: boolean
  className?: string
}

/** The approved Flow identity, with a vector mark for crisp small-size use. */
export function BrandLogo({ compact = false, className = '' }: BrandLogoProps) {
  return (
    <span className={`strand-brand ${className}`} aria-label="Strand">
      <img src="/icons/strand-flow.svg" alt="" width="38" height="33" />
      {!compact && (
        <span className="strand-wordmark" aria-hidden="true">
          strand
        </span>
      )}
    </span>
  )
}
