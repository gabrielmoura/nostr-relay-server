import ngeohash from "ngeohash"

export function normalizeGeohashInput(value: string): string {
  const normalized = value.trim().toLowerCase()
  if (!normalized) {
    return ""
  }

  try {
    ngeohash.decode(normalized)
    return normalized
  } catch {
    return normalized
  }
}

export function isValidGeohash(value: string): boolean {
  const normalized = normalizeGeohashInput(value)
  if (!normalized) {
    return false
  }

  try {
    ngeohash.decode(normalized)
    return true
  } catch {
    return false
  }
}
