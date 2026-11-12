// Strand brand links and metadata.

function b64(s: string): string {
  return btoa(s)
}

const ENCODED_LINKS = {
  twitter: b64('https://x.com/vergex_ai'),
  telegram: b64('https://t.me/strand_dev_community'),
  github: b64('https://github.com/Strand/strand'),
}

function decodeLink(key: keyof typeof ENCODED_LINKS): string {
  try {
    return atob(ENCODED_LINKS[key])
  } catch {
    const fallbacks: Record<string, string> = {
      twitter: 'https://x.com/vergex_ai',
      telegram: 'https://t.me/strand_dev_community',
      github: 'https://github.com/Strand/strand',
    }
    return fallbacks[key] || ''
  }
}

export const OFFICIAL_LINKS = {
  get twitter() {
    return decodeLink('twitter')
  },
  get telegram() {
    return decodeLink('telegram')
  },
  get github() {
    return decodeLink('github')
  },
} as const

export const BRAND_INFO = {
  name: 'STRAND',
  tagline: 'Self-hosted AI trading terminal',
  version: '1.0.0',
  social: {
    x: () => OFFICIAL_LINKS.twitter,
    tg: () => OFFICIAL_LINKS.telegram,
    gh: () => OFFICIAL_LINKS.github,
  },
} as const
