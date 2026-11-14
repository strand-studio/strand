// Constants for AI model and provider configuration

export interface Claw402Model {
  id: string
  name: string
  provider: string
  desc: string
  brand: string // key for getModelIcon / getModelColor
  priceIn: number // USD per 1M input tokens (upto pay-as-you-go)
  priceOut: number // USD per 1M output tokens
  isNew?: boolean
}

export interface AIProviderConfig {
  defaultModel: string
  /** OpenAI-compatible Base URL used as the blank-field default. */
  apiUrl: string
  /** Console / signup page where the user creates an API key. */
  apiKeyUrl: string
  apiName: string
}

// Get friendly AI model display name
export function getModelDisplayName(modelId: string): string {
  switch (modelId.toLowerCase()) {
    case 'deepseek':
      return 'DeepSeek'
    case 'qwen':
      return 'Qwen'
    case 'claude':
      return 'Claude'
    default:
      return modelId.toUpperCase()
  }
}

// Extract name part after underscore
export function getShortName(fullName: string): string {
  const parts = fullName.split('_')
  return parts.length > 1 ? parts[parts.length - 1] : fullName
}

export const DEFAULT_CLAW402_MODEL = 'gpt-6-luna'

// Models available through Claw402 (x402 USDC payment protocol)
// Must stay in sync with the claw402 catalog (GET /api/v1/catalog)
// Indicative retail USD per 1M tokens, including gateway markup. DeepSeek
// shows peak rates; GPT long-context multipliers and the per-request service
// fee are separate. The live gateway quote/verified receipt is authoritative.
export const CLAW402_MODELS: Claw402Model[] = [
  {
    id: 'gpt-6',
    name: 'GPT-6 Astra',
    provider: 'OpenAI',
    desc: 'Most capable · 1.05M context',
    brand: 'openai',
    priceIn: 14.375,
    priceOut: 57.5,
    isNew: true,
  },
  {
    id: 'gpt-6.1-sol',
    name: 'GPT-6.1 Sol',
    provider: 'OpenAI',
    desc: 'Latest workhorse · 1.05M context',
    brand: 'openai',
    priceIn: 2.875,
    priceOut: 11.5,
    isNew: true,
  },
  {
    id: 'gpt-6-sol',
    name: 'GPT-6 Sol',
    provider: 'OpenAI',
    desc: 'Previous workhorse · 1.05M context',
    brand: 'openai',
    priceIn: 2.875,
    priceOut: 11.5,
    isNew: true,
  },
  {
    id: 'gpt-6-luna',
    name: 'GPT-6 Luna',
    provider: 'OpenAI',
    desc: 'Fast and cost-efficient · 1.05M context',
    brand: 'openai',
    priceIn: 0.14375,
    priceOut: 0.575,
    isNew: true,
  },
  {
    id: 'gpt-5.6',
    name: 'GPT-5.6 Sol',
    provider: 'OpenAI',
    desc: 'Previous flagship',
    brand: 'openai',
    priceIn: 5.75,
    priceOut: 34.5,
  },
  {
    id: 'gpt-5.6-terra',
    name: 'GPT-5.6 Terra',
    provider: 'OpenAI',
    desc: 'Balanced',
    brand: 'openai',
    priceIn: 2.875,
    priceOut: 17.25,
  },
  {
    id: 'gpt-5.6-luna',
    name: 'GPT-5.6 Luna',
    provider: 'OpenAI',
    desc: 'Cost-efficient',
    brand: 'openai',
    priceIn: 1.15,
    priceOut: 6.9,
  },
  {
    id: 'claude-fable',
    name: 'Claude Fable 5',
    provider: 'Anthropic',
    desc: 'Most capable',
    brand: 'claude',
    priceIn: 14.375,
    priceOut: 57.5,
  },
  {
    id: 'claude-opus',
    name: 'Claude Opus 4.8',
    provider: 'Anthropic',
    desc: 'Coding & agents flagship',
    brand: 'claude',
    priceIn: 7.1875,
    priceOut: 28.75,
  },
  {
    id: 'deepseek-v4-flash',
    name: 'DeepSeek V4.1 Flash',
    provider: 'DeepSeek',
    desc: 'Fast general model',
    brand: 'deepseek',
    priceIn: 0.345,
    priceOut: 1.38,
  },
  {
    id: 'deepseek-v4-pro',
    name: 'DeepSeek-V4 Pro',
    provider: 'DeepSeek',
    desc: 'Advanced reasoning',
    brand: 'deepseek',
    priceIn: 1.518,
    priceOut: 4.554,
  },
  {
    id: 'glm-5',
    name: 'GLM-5',
    provider: 'Z.ai',
    desc: 'Deep reasoning flagship',
    brand: 'zhipu',
    priceIn: 1.15,
    priceOut: 3.68,
  },
]

// AI Provider configuration - default models and API endpoints.
// Keep in sync with resolveAIEndpoint in internal/api/config_credential.go.
export const AI_PROVIDER_CONFIG: Record<string, AIProviderConfig> = {
  claw402: {
    defaultModel: DEFAULT_CLAW402_MODEL,
    apiUrl: 'https://claw402.ai',
    apiKeyUrl: 'https://claw402.ai',
    apiName: 'Claw402',
  },
  deepseek: {
    defaultModel: 'deepseek-flash',
    apiUrl: 'https://api.deepseek.com',
    apiKeyUrl: 'https://platform.deepseek.com/api_keys',
    apiName: 'DeepSeek',
  },
  openai: {
    defaultModel: 'gpt-6-luna',
    apiUrl: 'https://api.openai.com/v1',
    apiKeyUrl: 'https://platform.openai.com/api-keys',
    apiName: 'OpenAI',
  },
  qwen: {
    defaultModel: 'qwen3-max',
    apiUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    apiKeyUrl: 'https://bailian.console.aliyun.com/?tab=model#/api-key',
    apiName: 'Qwen',
  },
  claude: {
    defaultModel: 'claude-sonnet-5-5',
    apiUrl: 'https://api.anthropic.com/v1',
    apiKeyUrl: 'https://console.anthropic.com/settings/keys',
    apiName: 'Claude',
  },
  gemini: {
    defaultModel: 'gemini-3.6-flash',
    apiUrl: 'https://generativelanguage.googleapis.com/v1beta/openai',
    apiKeyUrl: 'https://aistudio.google.com/apikey',
    apiName: 'Gemini',
  },
  grok: {
    defaultModel: 'grok-4.3',
    apiUrl: 'https://api.x.ai/v1',
    apiKeyUrl: 'https://console.x.ai/',
    apiName: 'Grok',
  },
  kimi: {
    defaultModel: 'kimi-k3',
    // Base URL stays .cn for regional defaults; key signup must be
    // international (moonshot.ai) — China-region keys are incompatible.
    apiUrl: 'https://api.moonshot.cn/v1',
    apiKeyUrl: 'https://platform.moonshot.ai/console/api-keys',
    apiName: 'Kimi',
  },
}

/** Resolve provider defaults for Base URL / model placeholders. */
export function getProviderDefaults(provider?: string): AIProviderConfig | null {
  if (!provider) return null
  return AI_PROVIDER_CONFIG[provider.toLowerCase()] || null
}

// Helper function to get exchange display name from exchange ID (UUID)
export function getExchangeDisplayName(
  exchangeId: string | undefined,
  exchanges: {
    id: string
    exchange_type?: string
    name: string
    account_name?: string
  }[]
): string {
  if (!exchangeId) return 'Unknown'
  const exchange = exchanges.find((e) => e.id === exchangeId)
  if (!exchange) return exchangeId.substring(0, 8).toUpperCase() + '...' // Show truncated UUID if not found
  const typeName = exchange.exchange_type?.toUpperCase() || exchange.name
  return exchange.account_name
    ? `${typeName} - ${exchange.account_name}`
    : typeName
}

// Helper function to check if exchange is a perp-dex type (wallet-based)
export function isPerpDexExchange(exchangeType: string | undefined): boolean {
  if (!exchangeType) return false
  const perpDexTypes = ['hyperliquid', 'lighter', 'aster']
  return perpDexTypes.includes(exchangeType.toLowerCase())
}

// Helper function to get wallet address for perp-dex exchanges
export function getWalletAddress(
  exchange:
    | {
        exchange_type?: string
        hyperliquidWalletAddr?: string
        lighterWalletAddr?: string
        asterSigner?: string
      }
    | undefined
): string | undefined {
  if (!exchange) return undefined
  const type = exchange.exchange_type?.toLowerCase()
  switch (type) {
    case 'hyperliquid':
      return exchange.hyperliquidWalletAddr
    case 'lighter':
      return exchange.lighterWalletAddr
    case 'aster':
      return exchange.asterSigner
    default:
      return undefined
  }
}

// Helper function to truncate wallet address for display
export function truncateAddress(
  address: string,
  startLen = 6,
  endLen = 4
): string {
  if (address.length <= startLen + endLen + 3) return address
  return `${address.slice(0, startLen)}...${address.slice(-endLen)}`
}
