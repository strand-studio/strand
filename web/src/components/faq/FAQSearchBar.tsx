import { Search, X } from 'lucide-react'

interface FAQSearchBarProps {
  searchTerm: string
  onSearchChange: (value: string) => void
  placeholder?: string
}

export function FAQSearchBar({
  searchTerm,
  onSearchChange,
  placeholder = 'Search Help…',
}: FAQSearchBarProps) {
  return (
    <div className="relative group">
      <Search className="absolute left-4 top-1/2 transform -translate-y-1/2 w-5 h-5 text-strand-text-muted group-focus-within:text-strand-gold transition-colors" />
      <input
        type="text"
        value={searchTerm}
        onChange={(e) => onSearchChange(e.target.value)}
        placeholder={placeholder}
        className="w-full pl-12 pr-12 py-3 rounded-lg text-base transition-all focus:outline-hidden bg-strand-bg-lighter border border-[rgba(25,36,52,0.14)] text-strand-text placeholder-strand-text-muted/50 focus:border-strand-gold/50 focus:ring-1 focus:ring-strand-gold/20 hover:border-strand-gold/30 font-sans"
      />
      {searchTerm && (
        <button
          onClick={() => onSearchChange('')}
          className="absolute right-4 top-1/2 transform -translate-y-1/2 text-strand-text-muted hover:text-strand-text transition-colors"
        >
          <X className="w-5 h-5" />
        </button>
      )}
    </div>
  )
}
