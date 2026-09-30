"use client"

import { useState, useEffect, useCallback, useRef } from 'react'
import { api } from '@/lib/api-client'

interface PresenceData {
  workshopId: string
  activeUsers: number
}

/**
 * usePresenceHeartbeat
 * Hook gửi tín hiệu nhịp tim (Heartbeat) định kỳ lên Server để phục vụ
 * tính năng đếm người trực chờ và tự động mở rộng Pods (Proactive Autoscaling).
 *
 * Tính năng:
 * - Gửi GET /api/v1/workshops/:id/presence mỗi 15 giây.
 * - Nhận về số lượng active users thực tế để hiển thị lên UI.
 * - Tự động tạm dừng khi ẩn tab (Page Visibility API) để tiết kiệm pin và mạng.
 * - Tự động kích hoạt lại ngay khi người dùng quay lại tab.
 */
export function usePresenceHeartbeat(workshopId: string | null | undefined, intervalMs = 15000) {
  const [activeUsers, setActiveUsers] = useState<number>(0)
  const [isLoading, setIsLoading] = useState<boolean>(true)
  const timerRef = useRef<NodeJS.Timeout | null>(null)

  const sendHeartbeat = useCallback(async () => {
    if (!workshopId) return
    try {
      const response = await api.get<PresenceData>(`/api/v1/workshops/${workshopId}/presence`)
      if (response?.data?.activeUsers !== undefined) {
        setActiveUsers(response.data.activeUsers)
      }
    } catch {
      // Nuốt lỗi ngầm để không làm phiền trải nghiệm người dùng
    } finally {
      setIsLoading(false)
    }
  }, [workshopId])

  useEffect(() => {
    if (!workshopId) return

    // 1. Gửi nhịp tim đầu tiên ngay khi mở màn hình
    sendHeartbeat()

    // 2. Định kỳ lặp lại theo chu kỳ (mặc định 15 giây)
    timerRef.current = setInterval(sendHeartbeat, intervalMs)

    // 3. Tối ưu theo trạng thái tab: tạm dừng khi tab bị ẩn
    const handleVisibilityChange = () => {
      if (document.hidden) {
        if (timerRef.current) clearInterval(timerRef.current)
      } else {
        sendHeartbeat()
        timerRef.current = setInterval(sendHeartbeat, intervalMs)
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [workshopId, intervalMs, sendHeartbeat])

  return { activeUsers, isLoading, refreshPresence: sendHeartbeat }
}
