const STORAGE_KEY = 'strand.preferred_model_id'

/** Client-side preferred AI model for Chat / Autopilot model pickers. */
export function getPreferredModelId(): string {
  try {
    return localStorage.getItem(STORAGE_KEY)?.trim() || ''
  } catch {
    return ''
  }
}

export function setPreferredModelId(modelId: string): void {
  try {
    const id = modelId.trim()
    if (!id) {
      localStorage.removeItem(STORAGE_KEY)
      return
    }
    localStorage.setItem(STORAGE_KEY, id)
  } catch {
    // ignore quota / private mode
  }
}

/** Pick preferred enabled model, else first enabled, else first configured. */
export function pickPreferredModel<
  T extends { id: string; enabled?: boolean },
>(models: T[], preferredId?: string): T | undefined {
  if (!models.length) return undefined
  const preferred = (preferredId || getPreferredModelId()).trim()
  const enabled = models.filter((m) => m.enabled !== false)
  const pool = enabled.length > 0 ? enabled : models
  if (preferred) {
    const hit = pool.find((m) => m.id === preferred)
    if (hit) return hit
  }
  return pool[0]
}
