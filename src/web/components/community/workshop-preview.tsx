'use client'
import { useSessionUser } from '@/hooks/use-session-user'
import { useEffect, useState } from 'react'
import { CalendarDays, MapPin, Users, ArrowUpRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useRegistration } from '@/hooks/use-registration'
import { type Workshop, type SessionUser, dateText } from './shared'

export function WorkshopPreview({ workshop }: { workshop: Workshop }) {
  const [open, setOpen] = useState(false),
    [now, setNow] = useState(Date.now()),
    [registered, setRegistered] = useState(false)
  const { handleRegister, isRegistering, regStatus, waitingPosition } =
    useRegistration(workshop.id)
  const user = useSessionUser() as SessionUser | null
  useEffect(() => {
    const tick = setInterval(() => setNow(Date.now()), 1000)
    const success = () => setRegistered(true)
    window.addEventListener(`workshop-reg-success-${workshop.id}`, success)
    return () => {
      clearInterval(tick)
      window.removeEventListener(`workshop-reg-success-${workshop.id}`, success)
    }
  }, [workshop.id])
  const before = now < new Date(workshop.registrationStartTime).getTime()
  const ended = now > new Date(workshop.registrationEndTime).getTime()
  const allowed = user?.role === 'STUDENT' || user?.role === 'ADMIN'
  const eligible =
    allowed &&
    workshop.status === 'PUBLISHED' &&
    !before &&
    !ended &&
    workshop.availableSeats > 0 &&
    !registered
  let label = 'Đăng ký workshop'
  if (!allowed) label = 'Đăng ký dành cho sinh viên'
  else if (registered) label = 'Đã đăng ký'
  else if (workshop.status === 'CANCELLED') label = 'Sự kiện đã hủy'
  else if (workshop.status !== 'PUBLISHED' || ended) label = 'Đã đóng đăng ký'
  else if (before) label = 'Chưa mở đăng ký'
  else if (workshop.availableSeats < 1) label = 'Đã đủ chỗ'
  return (
    <>
      <div className="rounded-xl border border-indigo-100 bg-indigo-50/60 p-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="text-[11px] font-bold uppercase tracking-wider text-indigo-600">
              Workshop {workshop.companyName && `· ${workshop.companyName}`}
            </p>
            <h3 className="mt-1 font-bold text-slate-900">{workshop.title}</h3>
          </div>
          <CalendarDays className="h-5 w-5 shrink-0 text-indigo-500" />
        </div>
        <div className="mt-3 space-y-1 text-sm text-slate-600">
          <p className="flex items-center gap-2">
            <CalendarDays className="h-3.5 w-3.5" />
            {dateText(workshop.startTime)}
          </p>
          <p className="flex items-center gap-2">
            <MapPin className="h-3.5 w-3.5" />
            {workshop.room}
          </p>
          <p className="flex items-center gap-2">
            <Users className="h-3.5 w-3.5" />
            {workshop.availableSeats} chỗ còn lại / {workshop.capacity} · Miễn
            phí
          </p>
        </div>
        <Button
          size="sm"
          variant="outline"
          className="mt-4 border-indigo-200 text-indigo-700"
          onClick={() => setOpen(true)}
        >
          Xem chi tiết
          <ArrowUpRight className="ml-1 h-4 w-4" />
        </Button>
      </div>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{workshop.title}</DialogTitle>
            <DialogDescription>
              {dateText(workshop.startTime)} – {dateText(workshop.endTime)} ·{' '}
              {workshop.room}
            </DialogDescription>
          </DialogHeader>
          {workshop.coverUrl && (
            <img
              src={workshop.coverUrl}
              alt={workshop.title}
              className="max-h-64 w-full rounded-xl object-cover"
            />
          )}
          <div className="space-y-4 text-sm">
            {[
              ['Giới thiệu', workshop.description || workshop.summary],
              ['Chương trình có gì?', workshop.agenda],
              ['Diễn giả', workshop.speaker],
              ['Phù hợp với ai?', workshop.audience],
              ['Bạn sẽ nhận được gì?', workshop.benefits],
              ['Cần chuẩn bị', workshop.preparation],
            ].map(
              ([title, value]) =>
                value && (
                  <section key={title}>
                    <h3 className="font-semibold text-slate-900">{title}</h3>
                    <p className="mt-1 whitespace-pre-wrap leading-6 text-slate-600">
                      {value}
                    </p>
                  </section>
                )
            )}
            <p className="rounded-lg bg-slate-50 p-3">
              Cổng đăng ký: {dateText(workshop.registrationStartTime)} –{' '}
              {dateText(workshop.registrationEndTime)}
              <br />
              Còn {workshop.availableSeats} chỗ. Thích hoặc lưu bài không giữ
              ghế.
            </p>
          </div>
          <Button
            disabled={!eligible || isRegistering}
            onClick={() => void handleRegister()}
          >
            {isRegistering ? regStatus || 'Đang xử lý…' : label}
            {waitingPosition ? ` · Vị trí ${waitingPosition}` : ''}
          </Button>
        </DialogContent>
      </Dialog>
    </>
  )
}
