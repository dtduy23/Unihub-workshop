import { useCallback, useEffect, useRef, useState } from 'react'
import { api, APIError } from '@/lib/api-client'
import type { RegistrationStatusResponse } from '@/lib/types'
import { toast } from 'sonner'

type WaitingRoom = { status: number; position?: number }

export function useRegistration(workshopId: string) {
  const [isRegistering, setIsRegistering] = useState(false)
  const [regStatus, setRegStatus] = useState<string | null>(null)
  const [waitingPosition, setWaitingPosition] = useState<number | null>(null)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const generation = useRef(0)
  const busy = useRef(false)

  useEffect(() => {
    busy.current = false
    setIsRegistering(false)
    setRegStatus(null)
    setWaitingPosition(null)
    return () => {
      generation.current++
      if (timer.current) clearTimeout(timer.current)
      busy.current = false
    }
  }, [workshopId])

  const handleRegister = useCallback(async () => {
    if (busy.current) return
    busy.current = true
    const run = ++generation.current
    const started = Date.now()
    let errors = 0
    setIsRegistering(true)
    setRegStatus('Đang gửi yêu cầu…')
    setWaitingPosition(null)
    const active = () => run === generation.current
    const finish = () => {
      busy.current = false
      setIsRegistering(false)
      setRegStatus(null)
      setWaitingPosition(null)
    }
    const fail = (error: unknown) => {
      if (!active()) return
      finish()
      toast.error(
        error instanceof Error ? error.message : 'Không thể đăng ký workshop'
      )
    }
    const schedule = (callback: () => Promise<void>, delay: number) => {
      if (!active()) return
      if (Date.now() - started > 180_000) {
        finish()
        toast.info(
          'Yêu cầu đang chờ xử lý. Kiểm tra lại thông báo và vé của bạn sau ít phút.'
        )
        return
      }
      timer.current = setTimeout(() => {
        void callback()
      }, delay)
    }
    const retry = (error: unknown, callback: () => Promise<void>) => {
      if (!active()) return
      if (
        error instanceof APIError &&
        error.status !== 429 &&
        error.status < 500
      ) {
        fail(error)
        return
      }
      if (++errors > 5) {
        fail(
          new Error(
            'Kết nối bị gián đoạn. Kiểm tra vé của bạn trước khi thử lại.'
          )
        )
        return
      }
      schedule(
        callback,
        error instanceof APIError ? (error.retryAfter || 5) * 1000 : 5000
      )
    }
    const pollStatus = async (id: string): Promise<void> => {
      try {
        const response = await api.get<RegistrationStatusResponse>(
          `/api/v1/registrations/status/${id}`
        )
        if (!active()) return
        errors = 0
        const status = response.data
        if (!status || status.status === 'PROCESSING') {
          schedule(() => pollStatus(id), 1500)
          return
        }
        finish()
        if (status.status === 'SUCCESS') {
          toast.success(
            'Đăng ký thành công! Vé của bạn đã sẵn sàng trong mục thông báo.'
          )
          window.dispatchEvent(
            new CustomEvent('registration-success', { detail: { workshopId } })
          )
          window.dispatchEvent(
            new CustomEvent(`workshop-reg-success-${workshopId}`)
          )
        } else toast.error(status.message || 'Không thể hoàn tất đăng ký')
      } catch (error) {
        retry(error, () => pollStatus(id))
      }
    }
    const pollWaiting = async (): Promise<void> => {
      try {
        const response = await api.get<WaitingRoom>(
          `/api/v1/registrations/waiting-room/${workshopId}`
        )
        if (!active()) return
        errors = 0
        if ([1, 3].includes(response.data?.status || 0)) {
          await submit()
          return
        }
        setWaitingPosition(response.data?.position || null)
        schedule(pollWaiting, 5000)
      } catch (error) {
        retry(error, pollWaiting)
      }
    }
    const submit = async (): Promise<void> => {
      try {
        const response = await api.post<{ correlationId: string }>(
          '/api/v1/registrations',
          { workshopId }
        )
        if (!active()) return
        if (!response.data?.correlationId) {
          fail(new Error('Máy chủ chưa xác nhận yêu cầu'))
          return
        }
        setRegStatus('Đang xử lý đăng ký…')
        schedule(() => pollStatus(response.data!.correlationId), 1000)
      } catch (error) {
        if (!active()) return
        if (error instanceof APIError && error.status === 429) {
          const room = error.data as WaitingRoom | undefined
          if (room && [0, 2].includes(room.status)) {
            setRegStatus('Đang trong phòng chờ…')
            setWaitingPosition(room.position || null)
            schedule(pollWaiting, 5000)
          } else {
            setRegStatus('Vui lòng đợi…')
            retry(error, submit)
          }
        } else fail(error)
      }
    }
    await submit()
  }, [workshopId])

  return { isRegistering, regStatus, waitingPosition, handleRegister }
}
