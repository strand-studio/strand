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

type HealthAware = {
  id: string
  enabled?: boolean
  last_test_ok?: boolean
  tested?: boolean
}

/** Prefer healthy+enabled, then enabled, then any. Honors preferred id when eligible. */
export function pickPreferredModel<T extends HealthAware>(
  models: T[],
  preferredId?: string
): T | undefined {
  if (!models.length) return undefined
  const preferred = (preferredId || getPreferredModelId()).trim()
  const enabled = models.filter((m) => m.enabled !== false)
  const healthy = enabled.filter(
    (m) => m.tested !== true || m.last_test_ok !== false
  )
  const pool = healthy.length > 0 ? healthy : enabled.length > 0 ? enabled : models
  if (preferred) {
    const hit = pool.find((m) => m.id === preferred)
    if (hit) return hit
  }
  return pool[0]
}

/** Filter resources suitable for business pickers (enabled + not known-failing). */
export function selectableResources<T extends HealthAware>(items: T[]): T[] {
  const enabled = items.filter((i) => i.enabled !== false)
  const healthy = enabled.filter(
    (i) => i.tested !== true || i.last_test_ok !== false
  )
  return healthy.length > 0 ? healthy : enabled
}
