'use client'

/**
 * Hook tự động đồng bộ check-in từ IndexedDB lên backend.
 * 
 * - Khi online: sync mỗi 15 giây
 * - Khi offline: dừng sync, hiển thị trạng thái
 * - Khi online trở lại: sync ngay lập tức
 */

import { useEffect, useState, useCallback, useRef } from 'react'
import { getPendingCheckins, markAsSynced, markAsFailed } from '@/lib/checkin-db'
import { api } from '@/lib/api-client'

const SYNC_INTERVAL = 15_000 // 15 giây

export function useCheckinSync() {
  const [isOnline, setIsOnline] = useState(
    typeof navigator !== 'undefined' ? navigator.onLine : true
  )
  const [pendingCount, setPendingCount] = useState(0)
  const [lastSyncAt, setLastSyncAt] = useState<number | null>(null)
  const [isSyncing, setIsSyncing] = useState(false)
  const syncRef = useRef(false)

  // Sync logic
  const syncPending = useCallback(async () => {
    // Tránh sync chồng chéo
    if (syncRef.current) return
    if (!navigator.onLine) return

    syncRef.current = true
    setIsSyncing(true)

    try {
      const pending = await getPendingCheckins()
      setPendingCount(pending.length)

      if (pending.length === 0) {
        syncRef.current = false
        setIsSyncing(false)
        return
      }

      // Chuyển đổi format cho backend API
      const records = pending.map((r) => ({
        id: r.id,
        student_id: r.studentId,
        workshop_id: r.workshopId,
        scanned_at: Math.floor(r.scannedAt / 1000), // ms → seconds
      }))

      const response = await api.post<{ synced: string[]; failed: string[] }>(
        '/api/v1/checkin/sync',
        { records }
      )

      const data = response.data as any
      const synced = data?.synced || []
      const failed = data?.failed || []

      // Đánh dấu thành công
      for (const id of synced) {
        await markAsSynced(id)
      }

      // Đánh dấu thất bại
      for (const id of failed) {
        await markAsFailed(id)
      }

      // Cập nhật pending count
      const remaining = await getPendingCheckins()
      setPendingCount(remaining.length)
      setLastSyncAt(Date.now())

      if (synced.length > 0) {
        console.log(`[Sync] ✅ Đồng bộ ${synced.length} bản ghi thành công`)
      }
    } catch (error) {
      console.warn('[Sync] ⚠️ Lỗi đồng bộ:', error)
    } finally {
      syncRef.current = false
      setIsSyncing(false)
    }
  }, [])

  // Refresh pending count (không sync)
  const refreshCount = useCallback(async () => {
    const pending = await getPendingCheckins()
    setPendingCount(pending.length)
  }, [])

  // Lắng nghe online/offline events
  useEffect(() => {
    const handleOnline = () => {
      setIsOnline(true)
      console.log('[Sync] 🌐 Online — đồng bộ ngay...')
      syncPending()
    }

    const handleOffline = () => {
      setIsOnline(false)
      console.log('[Sync] 📴 Offline — lưu local, đợi mạng...')
    }

    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)

    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
    }
  }, [syncPending])

  // Auto-sync interval
  useEffect(() => {
    // Sync ngay khi mount
    syncPending()

    const timer = setInterval(syncPending, SYNC_INTERVAL)
    return () => clearInterval(timer)
  }, [syncPending])

  return {
    isOnline,
    pendingCount,
    lastSyncAt,
    isSyncing,
    syncNow: syncPending,
    refreshCount,
  }
}
