import readme from "../../../../ref/nips/README.md?raw"

export type NostrKindMeta = {
  kind: string
  description: string
  nip: string
}

function buildKindMap() {
  const map = new Map<string, NostrKindMeta>()
  const lines = readme.split("\n")
  let inTable = false

  for (const line of lines) {
    if (line.startsWith("## Event Kinds")) {
      inTable = true
      continue
    }
    if (inTable && line.startsWith("## ")) {
      break
    }
    if (!inTable || !line.startsWith("| `")) {
      continue
    }

    const parts = line.split("|").map((part) => part.trim())
    if (parts.length < 4) {
      continue
    }

    const kind = parts[1]?.replace(/`/g, "")
    const description = parts[2] || ""
    const nip = parts[3] || ""
    if (kind) {
      map.set(kind, { kind, description, nip })
    }
  }

  return map
}

const kindMap = buildKindMap()

export function getNostrKindMeta(kind: number): NostrKindMeta | null {
  const exact = kindMap.get(String(kind))
  if (exact) {
    return exact
  }

  for (const [key, value] of kindMap.entries()) {
    if (key.includes("-")) {
      const range = key.replace(/[^0-9-]/g, "")
      const [startRaw, endRaw] = range.split("-")
      const start = Number(startRaw)
      const end = Number(endRaw)
      if (!Number.isNaN(start) && !Number.isNaN(end) && kind >= start && kind <= end) {
        return value
      }
    }
  }

  return null
}
