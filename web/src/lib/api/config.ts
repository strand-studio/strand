import type {
  AIModel,
  Exchange,
  ExchangeAccountStateResponse,
  UpdateModelConfigRequest,
  UpdateExchangeConfigRequest,
  CreateExchangeRequest,
  BeginnerOnboardingResponse,
  CurrentBeginnerWalletResponse,
} from '../../types'
import { API_BASE, httpClient, CryptoService } from './helpers'

export type StrandBackup = {
  format: string
  version: number
  exported_at?: string
  include_secrets?: boolean
  models?: unknown[]
  exchanges?: unknown[]
  strategies?: unknown[]
  traders?: unknown[]
}

export const configApi = {
  async exportBackup(includeSecrets = false): Promise<StrandBackup> {
    const q = includeSecrets ? '?include_secrets=1' : ''
    const result = await httpClient.get<StrandBackup>(
      `${API_BASE}/backup/export${q}`
    )
    if (!result.success || !result.data) {
      throw new Error(result.message || 'Failed to export backup')
    }
    return result.data
  },

  async importBackup(
    backup: StrandBackup & { restore_secrets?: boolean }
  ): Promise<{
    restored_strategies?: number
    restored_models?: number
    restored_exchanges?: number
  }> {
    const result = await httpClient.post<{
      restored_strategies?: number
      restored_models?: number
      restored_exchanges?: number
    }>(`${API_BASE}/backup/import`, backup)
    if (!result.success) {
      throw new Error(result.message || 'Failed to import backup')
    }
    return result.data || {}
  },

  async getModelConfigs(): Promise<AIModel[]> {
    const result = await httpClient.get<AIModel[]>(`${API_BASE}/models`)
    if (!result.success) throw new Error('Failed to fetch model configs')
    if (!Array.isArray(result.data)) return []
    return result.data.map((model) =>
      model.provider === 'claw402'
        ? { ...model, name: model.name.replace(/^claw402\b/i, 'Claw402') }
        : model
    )
  },

  async getSupportedModels(): Promise<AIModel[]> {
    const result = await httpClient.get<AIModel[]>(
      `${API_BASE}/supported-models`
    )
    if (!result.success) throw new Error('Failed to fetch supported models')
    return result.data!
  },

  async getPromptTemplates(): Promise<string[]> {
    const res = await fetch(`${API_BASE}/prompt-templates`)
    if (!res.ok) throw new Error('Failed to fetch prompt templates')
    const data = await res.json()
    if (Array.isArray(data.templates)) {
      return data.templates.map((item: { name: string }) => item.name)
    }
    return []
  },

  async testModelConfig(request: {
    provider: string
    model_id?: string
    api_key: string
    custom_api_url?: string
    custom_model_name?: string
  }): Promise<void> {
    const result = await httpClient.request<{ ok?: boolean; error?: string }>(
      `${API_BASE}/models/test`,
      { method: 'POST', data: request, timeout: 60000, silent: true }
    )
    if (!result.success) {
      throw new Error(result.message || 'AI connection test failed')
    }
  },

  async testExchangeConfig(request: {
    exchange_id?: string
    exchange_type: string
    api_key?: string
    secret_key?: string
    passphrase?: string
    testnet?: boolean
    hyperliquid_wallet_addr?: string
    hyperliquid_unified_account?: boolean
    aster_user?: string
    aster_signer?: string
    aster_private_key?: string
    lighter_wallet_addr?: string
    lighter_private_key?: string
    lighter_api_key_private_key?: string
    lighter_api_key_index?: number
  }): Promise<void> {
    const result = await httpClient.request<{ ok?: boolean; error?: string }>(
      `${API_BASE}/exchanges/test`,
      { method: 'POST', data: request, timeout: 60000, silent: true }
    )
    if (!result.success) {
      throw new Error(result.message || 'Exchange connection test failed')
    }
  },

  async updateModelConfigs(request: UpdateModelConfigRequest): Promise<void> {
    // Check if transport encryption is enabled
    const config = await CryptoService.fetchCryptoConfig()

    if (!config.transport_encryption) {
      // Transport encryption disabled, send plaintext
      const result = await httpClient.put(`${API_BASE}/models`, request)
      if (!result.success)
        throw new Error(result.message || 'Failed to update model configs')
      return
    }

    // Fetch RSA public key
    const publicKey = await CryptoService.fetchPublicKey()

    // Initialize crypto service
    await CryptoService.initialize(publicKey)

    // Get user info from localStorage
    const userId = localStorage.getItem('user_id') || ''
    const sessionId = sessionStorage.getItem('session_id') || ''

    // Encrypt sensitive data
    const encryptedPayload = await CryptoService.encryptSensitiveData(
      JSON.stringify(request),
      userId,
      sessionId
    )

    // Send encrypted data
    const result = await httpClient.put(`${API_BASE}/models`, encryptedPayload)
    if (!result.success)
      throw new Error(result.message || 'Failed to update model configs')
  },

  async getExchangeConfigs(): Promise<Exchange[]> {
    const result = await httpClient.get<Exchange[]>(`${API_BASE}/exchanges`)
    if (!result.success) throw new Error('Failed to fetch exchange configs')
    return result.data!
  },

  async getExchangeAccountState(): Promise<ExchangeAccountStateResponse> {
    const result = await httpClient.get<ExchangeAccountStateResponse>(
      `${API_BASE}/exchanges/account-state`
    )
    if (!result.success || !result.data) {
      throw new Error('Failed to fetch exchange account states')
    }
    return result.data
  },

  async getSupportedExchanges(): Promise<Exchange[]> {
    const result = await httpClient.get<Exchange[]>(
      `${API_BASE}/supported-exchanges`
    )
    if (!result.success) throw new Error('Failed to fetch supported exchanges')
    return result.data!
  },

  async updateExchangeConfigs(
    request: UpdateExchangeConfigRequest
  ): Promise<void> {
    const result = await httpClient.put(`${API_BASE}/exchanges`, request)
    if (!result.success)
      throw new Error(result.message || 'Failed to update exchange configs')
  },

  async createExchange(
    request: CreateExchangeRequest
  ): Promise<{ id: string }> {
    const result = await httpClient.post<{ id: string }>(
      `${API_BASE}/exchanges`,
      request
    )
    if (!result.success)
      throw new Error(result.message || 'Failed to create exchange account')
    return result.data!
  },

  async createExchangeEncrypted(
    request: CreateExchangeRequest
  ): Promise<{ id: string }> {
    // Check if transport encryption is enabled
    const config = await CryptoService.fetchCryptoConfig()

    if (!config.transport_encryption) {
      // Transport encryption disabled, send plaintext
      const result = await httpClient.post<{ id: string }>(
        `${API_BASE}/exchanges`,
        request
      )
      if (!result.success)
        throw new Error(result.message || 'Failed to create exchange account')
      return result.data!
    }

    // Fetch RSA public key
    const publicKey = await CryptoService.fetchPublicKey()

    // Initialize crypto service
    await CryptoService.initialize(publicKey)

    // Get user info
    const userId = localStorage.getItem('user_id') || ''
    const sessionId = sessionStorage.getItem('session_id') || ''

    // Encrypt sensitive data
    const encryptedPayload = await CryptoService.encryptSensitiveData(
      JSON.stringify(request),
      userId,
      sessionId
    )

    // Send encrypted data
    const result = await httpClient.post<{ id: string }>(
      `${API_BASE}/exchanges`,
      encryptedPayload
    )
    if (!result.success)
      throw new Error(result.message || 'Failed to create exchange account')
    return result.data!
  },

  async deleteExchange(exchangeId: string): Promise<void> {
    const result = await httpClient.delete(
      `${API_BASE}/exchanges/${exchangeId}`
    )
    if (!result.success) throw new Error('Failed to delete exchange account')
  },

  async updateExchangeConfigsEncrypted(
    request: UpdateExchangeConfigRequest
  ): Promise<void> {
    // Check if transport encryption is enabled
    const config = await CryptoService.fetchCryptoConfig()

    if (!config.transport_encryption) {
      // Transport encryption disabled, send plaintext
      const result = await httpClient.put(`${API_BASE}/exchanges`, request)
      if (!result.success)
        throw new Error(result.message || 'Failed to update exchange configs')
      return
    }

    // Fetch RSA public key
    const publicKey = await CryptoService.fetchPublicKey()

    // Initialize crypto service
    await CryptoService.initialize(publicKey)

    // Get user info from localStorage
    const userId = localStorage.getItem('user_id') || ''
    const sessionId = sessionStorage.getItem('session_id') || ''

    // Encrypt sensitive data
    const encryptedPayload = await CryptoService.encryptSensitiveData(
      JSON.stringify(request),
      userId,
      sessionId
    )

    // Send encrypted data
    const result = await httpClient.put(
      `${API_BASE}/exchanges`,
      encryptedPayload
    )
    if (!result.success)
      throw new Error(result.message || 'Failed to update exchange configs')
  },

  async getServerIP(): Promise<{
    public_ip: string
    message: string
  }> {
    const result = await httpClient.get<{
      public_ip: string
      message: string
    }>(`${API_BASE}/server-ip`)
    if (!result.success) throw new Error('Failed to fetch server IP')
    return result.data!
  },

  async prepareBeginnerOnboarding(): Promise<BeginnerOnboardingResponse> {
    const result = await httpClient.post<BeginnerOnboardingResponse>(
      `${API_BASE}/onboarding/beginner`
    )
    if (!result.success || !result.data) {
      throw new Error(result.message || 'Failed to prepare beginner onboarding')
    }
    return result.data
  },

  async getCurrentBeginnerWallet(): Promise<CurrentBeginnerWalletResponse> {
    const result = await httpClient.get<CurrentBeginnerWalletResponse>(
      `${API_BASE}/onboarding/beginner/current`
    )
    if (!result.success || !result.data) {
      throw new Error(
        result.message || 'Failed to fetch current beginner wallet'
      )
    }
    return result.data
  },
}
