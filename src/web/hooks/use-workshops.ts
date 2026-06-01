import { useState, useEffect, useCallback } from 'react'
import { api, auth } from '@/lib/api-client'
import { toast } from 'sonner'

export interface Workshop {
  id: string
  title: string
  description: string
  speaker: string
  room: string
  startTime: string
  endTime: string
  registrationStartTime: string
  registrationEndTime: string
  capacity: number
  availableSeats: number
  price: number
  summary: string
  status: string
  roomLayoutUrl?: string
  datetime: string // Thêm trường này cho Admin
  registered: number // Thêm trường này cho Admin
  // Các trường helper cho UI
  date: string
  time: string
  ticketType: 'free' | 'paid'
  isRegistered?: boolean // Thêm trường này để UI biết
}


export function useWorkshops() {
  const [workshops, setWorkshops] = useState<Workshop[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isOffline, setIsOffline] = useState(false)
  const [registeredIds, setRegisteredIds] = useState<Set<string>>(new Set())
  
  const user = auth.getUser() as any
  const userId = user?.id || 'guest'
  const cacheKey = `unihub_workshops_cache_${userId}`

  // Hàm chuyển đổi dữ liệu từ Backend sang format UI
  const formatWorkshop = (w: any): Workshop => {
    const start = new Date(w.startTime)
    const end = new Date(w.endTime)
    
    return {
      ...w,
      location: w.room || 'Chưa xác định',
      date: start.toLocaleDateString('vi-VN'),
      time: `${start.getHours().toString().padStart(2, '0')}:${start.getMinutes().toString().padStart(2, '0')} - ${end.getHours().toString().padStart(2, '0')}:${end.getMinutes().toString().padStart(2, '0')}`,
      ticketType: w.price > 0 ? 'paid' : 'free',
      category: w.category || 'Công nghệ', 
      speakerTitle: 'Giảng viên/Chuyên gia',
      datetime: w.startTime, // Sử dụng startTime (đã được đổi từ start_time)
      registered: (w.capacity || 0) - (w.availableSeats || 0) // Sử dụng availableSeats
    }
  }

  const fetchWorkshops = useCallback(async () => {
    setIsLoading(true)
    try {
      const [wsResponse, regResponse] = await Promise.all([
        api.get<any[]>('/api/v1/workshops'),
        auth.isAuthenticated() ? api.get<any[]>('/api/v1/registrations/my') : Promise.resolve({ success: true, data: [] })
      ])

      let regIds = new Set<string>()
      if (regResponse.success && regResponse.data) {
        // Lọc bỏ các trạng thái không còn hiệu lực (Đã hủy, thất bại, từ chối)
        const activeRegs = regResponse.data.filter((r: any) => 
          r.status !== 'CANCELLED' && 
          r.status !== 'REJECTED' && 
          r.status !== 'FAILED'
        )
        regIds = new Set(activeRegs.map((r: any) => r.workshopId))
        setRegisteredIds(regIds)
      }

      if (wsResponse.success && wsResponse.data) {
        const formatted = wsResponse.data.map(w => ({
          ...formatWorkshop(w),
          isRegistered: regIds.has(w.id)
        }))
        setWorkshops(formatted)
        // Lưu vào cache riêng của user này
        localStorage.setItem(cacheKey, JSON.stringify(formatted))
        setIsOffline(false)
      }
    } catch (error) {
      console.error('Fetch workshops error:', error)
      // Thử lấy từ cache của user này nếu lỗi (hoặc mất mạng)
      const cached = localStorage.getItem(cacheKey)
      if (cached) {
        setWorkshops(JSON.parse(cached))
        setIsOffline(true)
        toast.info('Bạn đang xem dữ liệu ngoại tuyến (offline)')
      } else {
        toast.error('Không thể tải danh sách workshop')
      }
    } finally {
      setIsLoading(true)
      // Giả lập loading mượt mà
      setTimeout(() => setIsLoading(false), 300)
    }
  }, [cacheKey])

  useEffect(() => {
    // Khi user thay đổi, xóa sạch state cũ trước khi fetch mới
    setWorkshops([])
    setRegisteredIds(new Set())
    
    fetchWorkshops()

    // Theo dõi trạng thái mạng
    const handleOnline = () => {
      setIsOffline(false)
      fetchWorkshops()
      toast.success('Đã khôi phục kết nối mạng')
    }
    const handleOffline = () => {
      setIsOffline(true)
      toast.warning('Mất kết nối mạng. Đang chạy ở chế độ offline.')
    }

    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)

    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
    }
  }, [fetchWorkshops, userId]) // Theo dõi userId

  return {
    workshops,
    loading: isLoading,
    error: isOffline ? 'Mất kết nối mạng' : null,
    isOffline,
    refresh: fetchWorkshops
  }
}
