import { useEffect, useState } from 'react'

/** No existing connection-status detection anywhere in this app --
 * built from scratch. navigator.onLine reflects network *interface*
 * state (e.g. wifi connected), not whether the API is actually
 * reachable, so this is a coarse signal -- TakeExamPage combines it with
 * its own autosave-failure/retry state for the fuller "Saved / Saving… /
 * Offline / Connection lost / Retrying…" picture the exam UI needs. */
export function useOnlineStatus(): boolean {
  const [online, setOnline] = useState(() => (typeof navigator === 'undefined' ? true : navigator.onLine))

  useEffect(() => {
    const goOnline = () => setOnline(true)
    const goOffline = () => setOnline(false)
    window.addEventListener('online', goOnline)
    window.addEventListener('offline', goOffline)
    return () => {
      window.removeEventListener('online', goOnline)
      window.removeEventListener('offline', goOffline)
    }
  }, [])

  return online
}
