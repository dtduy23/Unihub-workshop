'use client'
import { useMemo, useSyncExternalStore } from 'react'
import type { User } from '@/lib/types'

function subscribe(callback: () => void) {
  window.addEventListener('auth-session-change', callback)
  window.addEventListener('focus', callback)
  return () => {
    window.removeEventListener('auth-session-change', callback)
    window.removeEventListener('focus', callback)
  }
}
function snapshot() {
  return document.cookie.match(/(?:^|;\s*)unihub_session=([^;]*)/)?.[1] || ''
}

// Match the server's initial render; read display information after hydration.
// API authorization always uses the server's HttpOnly token.
export function useSessionUser(): User | null {
  const cookie = useSyncExternalStore(subscribe, snapshot, () => '')
  return useMemo(() => {
    if (!cookie) return null
    try {
      return JSON.parse(decodeURIComponent(cookie)) as User
    } catch {
      return null
    }
  }, [cookie])
}
