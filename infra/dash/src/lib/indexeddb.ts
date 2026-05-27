const dbName = "relay-admin-dash"
const storeName = "blossom-mirror-history"
const version = 1

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = window.indexedDB.open(dbName, version)
    request.onerror = () => reject(request.error ?? new Error("Failed to open IndexedDB"))
    request.onupgradeneeded = () => {
      const database = request.result
      if (!database.objectStoreNames.contains(storeName)) {
        database.createObjectStore(storeName, { keyPath: "id" })
      }
    }
    request.onsuccess = () => resolve(request.result)
  })
}

export type BlossomMirrorHistoryEntry = {
  id: string
  source_url: string
  expected_sha256: string
  job_id?: string
  status: string
  created_at: string
}

export async function loadMirrorHistory(): Promise<BlossomMirrorHistoryEntry[]> {
  if (typeof window === "undefined" || !("indexedDB" in window)) {
    return []
  }
  const database = await openDatabase()
  return new Promise((resolve, reject) => {
    const transaction = database.transaction(storeName, "readonly")
    const store = transaction.objectStore(storeName)
    const request = store.getAll()
    request.onerror = () => reject(request.error ?? new Error("Failed to read mirror history"))
    request.onsuccess = () => {
      const items = (request.result as BlossomMirrorHistoryEntry[]).sort((left, right) => right.created_at.localeCompare(left.created_at))
      resolve(items)
    }
  })
}

export async function saveMirrorHistoryEntry(entry: BlossomMirrorHistoryEntry): Promise<void> {
  if (typeof window === "undefined" || !("indexedDB" in window)) {
    return
  }
  const database = await openDatabase()
  await new Promise<void>((resolve, reject) => {
    const transaction = database.transaction(storeName, "readwrite")
    transaction.oncomplete = () => resolve()
    transaction.onerror = () => reject(transaction.error ?? new Error("Failed to save mirror history"))
    transaction.objectStore(storeName).put(entry)
  })
}

export async function clearMirrorHistoryEntries(): Promise<void> {
  if (typeof window === "undefined" || !("indexedDB" in window)) {
    return
  }
  const database = await openDatabase()
  await new Promise<void>((resolve, reject) => {
    const transaction = database.transaction(storeName, "readwrite")
    transaction.oncomplete = () => resolve()
    transaction.onerror = () => reject(transaction.error ?? new Error("Failed to clear mirror history"))
    transaction.objectStore(storeName).clear()
  })
}
