import { useState, useEffect, useCallback } from "react"
import { api, auth } from "@/lib/api-client"

export interface Notification {
  id: string
  userId: string
  registrationId?: string
  channel: "EMAIL" | "WEB"
  title: string
  content: string
  status: "PENDING" | "SENT" | "FAILED"
  createdAt: string
}

export function useNotifications() {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [loading, setLoading] = useState(false)
  const [unreadCount, setUnreadCount] = useState(0)

  const fetchNotifications = useCallback(async () => {
    if (!auth.isAuthenticated()) return
    setLoading(true)
    try {
      const response = await api.get<Notification[]>('/api/v1/notifications')
      if (response.success && response.data) {
        setNotifications(response.data)
        // Trong phiên bản này, ta coi tất cả thông báo mới load là chưa đọc 
        // nếu chưa có flag 'read' ở DB. Ở đây tạm tính theo client state.
        setUnreadCount(response.data.filter(n => n.status === 'SENT').length)
      }
    } catch (error) {
      console.error("Lỗi khi tải thông báo:", error)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchNotifications()
    
    const handleSuccess = () => fetchNotifications()
    window.addEventListener('registration-success', handleSuccess)
    
    // Polling mỗi 30 giây
    const interval = setInterval(fetchNotifications, 30000)
    return () => {
      clearInterval(interval)
      window.removeEventListener('registration-success', handleSuccess)
    }
  }, [fetchNotifications])

  return { notifications, loading, unreadCount, refresh: fetchNotifications }
}
