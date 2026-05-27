import { useState, useCallback, useRef, useEffect } from 'react'
import { api, APIError } from '@/lib/api-client'
import { toast } from 'sonner'

export function useRegistration(workshopId: string) {
  const [isRegistering, setIsRegistering] = useState(false)
  const [regStatus, setRegStatus] = useState<string | null>(null)
  const [waitingPosition, setWaitingPosition] = useState<number | null>(null)
  const [showPaymentDialog, setShowPaymentDialog] = useState(false)
  const [paymentInfo, setPaymentInfo] = useState<{ url: string, amount: number, title: string } | null>(null)

  // Dùng Ref để phá vỡ vòng lặp phụ thuộc giữa handleRegister và pollWaitingRoom
  const handleRegisterRef = useRef<() => Promise<void>>()

  // 1. Polling Status
  const pollRegistrationStatus = useCallback(async (correlationId: string) => {
    const interval = setInterval(async () => {
      try {
        const response = await api.get<any>(`/api/v1/registrations/status/${correlationId}`)

        if (response.data?.status !== 'PROCESSING') {
          clearInterval(interval)
          setIsRegistering(false)
          setRegStatus(null)

          if (response.data?.status === 'SUCCESS') {
            toast.success('Đăng ký thành công!', {
              id: `reg-success-${workshopId}`,
              description: 'Vui lòng kiểm tra email để nhận thông tin vé và hướng dẫn tham gia.'
            })
            window.dispatchEvent(new CustomEvent('registration-success', { detail: { workshopId } }))
            window.dispatchEvent(new CustomEvent(`workshop-reg-success-${workshopId}`))
          } else if (response.data?.status === 'PENDING_PAYMENT') {
            // Không hiện toast ở đây vì QR dialog sẽ hiện ra
            setPaymentInfo({
              url: response.data.paymentUrl,
              amount: response.data.paymentAmount,
              title: response.data.message || 'Thanh toán đăng ký Workshop'
            })
            setShowPaymentDialog(true)
            window.dispatchEvent(new CustomEvent('registration-success', { detail: { workshopId } }))
            window.dispatchEvent(new CustomEvent(`workshop-reg-success-${workshopId}`))
          } else {
            toast.error('Yêu cầu bị từ chối', {
              id: `reg-error-${workshopId}`,
              description: response.data?.message || 'Không thể hoàn tất đăng ký lúc này.'
            })
          }
        }
      } catch (error) {
        console.error('Lỗi Polling Status:', error)
      }
    }, 500)
    return () => clearInterval(interval)
  }, [workshopId])

  // 2. Polling Waiting Room
  const pollWaitingRoom = useCallback(async () => {
    const interval = setInterval(async () => {
      try {
        const response = await api.get<any>(`/api/v1/registrations/waiting-room/${workshopId}`)

        if (response.data?.status === 1) { // GRANTED
          clearInterval(interval)
          if (handleRegisterRef.current) {
            handleRegisterRef.current()
          }
        } else if (response.data?.position) {
          setWaitingPosition(response.data.position)
        }
      } catch (error) {
        console.error('Lỗi Polling Waiting Room:', error)
      }
    }, 5000)
    return () => clearInterval(interval)
  }, [workshopId])

  // 3. Handle Register
  const handleRegister = useCallback(async () => {
    setIsRegistering(true)
    setRegStatus('Đang gửi yêu cầu...')

    try {
      const response = await api.post<any>('/api/v1/registrations', {
        workshop_id: workshopId
      })

      if (response.data?.correlationId) {
        setRegStatus('Hệ thống đang xử lý...')
        pollRegistrationStatus(response.data.correlationId)
      }
    } catch (error: any) {
      const errorMsg = error.message || ""
      
      if (errorMsg.includes("bảo trì") || errorMsg.includes("maintenance") || errorMsg.includes("outage") || errorMsg.includes("circuit breaker")) {
        toast.error('Cổng thanh toán đang bảo trì', {
          id: `reg-maint-${workshopId}`,
          description: 'Hệ thống thanh toán hiện đang bảo trì để nâng cấp. Vui lòng quay lại sau ít phút.'
        })
      } else if (error.status === 401) {
        toast.error('Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại.', { id: 'auth-error' })
      } else if (error.status === 429) {
        setRegStatus('Đang trong phòng chờ...')
        pollWaitingRoom()
      } else {
        toast.error('Đăng ký thất bại', {
          id: `reg-error-${workshopId}`,
          description: errorMsg || 'Vui lòng thử lại sau.'
        })
      }
      setIsRegistering(false)
      setRegStatus(null)
    }
  }, [workshopId, pollRegistrationStatus, pollWaitingRoom])

  // Cập nhật ref mỗi khi handleRegister thay đổi
  useEffect(() => {
    handleRegisterRef.current = handleRegister
  }, [handleRegister])

  return {
    isRegistering,
    regStatus,
    waitingPosition,
    showPaymentDialog,
    setShowPaymentDialog,
    paymentInfo,
    handleRegister
  }
}
