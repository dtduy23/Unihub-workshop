'use client'
import { useState } from 'react'
import { api } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import {
  type Workshop,
  type Company,
  type Revision,
  useResource,
  LoadState,
  Status,
  dateText,
  errorText,
} from './shared'
import { WorkshopPreview } from './workshop-preview'

const localTime = (date: string) =>
  new Date(new Date(date).getTime() + 7 * 3600000).toISOString().slice(0, 16)
export function WorkshopEditor({
  open,
  onOpenChange,
  initial,
  onSaved,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  initial?: Workshop
  onSaved: () => void
}) {
  const [busy, setBusy] = useState(false)
  const revision = initial && ['PUBLISHED', 'CLOSED'].includes(initial.status)
  const nextWeek = new Date(Date.now() + 7 * 86400000).toISOString()
  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    const form = new FormData(event.currentTarget)
    const payload: Record<string, unknown> = Object.fromEntries(form)
    for (const key of [
      'startTime',
      'endTime',
      'registrationStartTime',
      'registrationEndTime',
    ])
      payload[key] = new Date(`${payload[key]}+07:00`).toISOString()
    payload.capacity = Number(payload.capacity)
    payload.price = 0
    try {
      if (revision)
        await api.post(`/api/v1/business/workshops/${initial.id}/revisions`, {
          payload,
        })
      else if (initial)
        await api.patch(`/api/v1/business/workshops/${initial.id}`, payload)
      else await api.post('/api/v1/business/workshops', payload)
      toast.success(
        revision
          ? 'Đã gửi thay đổi để duyệt; lịch hiện tại vẫn có hiệu lực.'
          : 'Đã lưu workshop nháp'
      )
      onOpenChange(false)
      onSaved()
    } catch (e) {
      toast.error(errorText(e))
    } finally {
      setBusy(false)
    }
  }
  const text = [
    ['title', 'Tên workshop', initial?.title || ''],
    ['speaker', 'Diễn giả', initial?.speaker || ''],
    ['room', 'Địa điểm', initial?.room || ''],
  ]
  const areas = [
    ['description', 'Giới thiệu', initial?.description || ''],
    ['agenda', 'Chương trình có gì?', initial?.agenda || ''],
    ['audience', 'Đối tượng tham gia', initial?.audience || ''],
    ['benefits', 'Người tham gia nhận được gì?', initial?.benefits || ''],
    ['preparation', 'Cần chuẩn bị', initial?.preparation || ''],
  ]
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>
            {revision
              ? 'Đề nghị thay đổi workshop'
              : initial
                ? 'Sửa workshop nháp'
                : 'Tạo workshop'}
          </DialogTitle>
          <DialogDescription>
            Workshop miễn phí. Thời gian được nhập theo múi giờ Việt Nam
            (UTC+7).
          </DialogDescription>
        </DialogHeader>
        <form key={initial?.id || 'new'} onSubmit={save} className="space-y-4">
          {text.map(([name, title, value]) => (
            <label className="block text-sm font-medium" key={name}>
              {title}
              <input
                name={name}
                aria-label={title}
                defaultValue={value}
                required
                maxLength={200}
                className="mt-1 w-full rounded-lg border p-2.5"
              />
            </label>
          ))}
          <div className="grid gap-4 sm:grid-cols-2">
            {[
              ['startTime', 'Bắt đầu workshop', initial?.startTime || nextWeek],
              [
                'endTime',
                'Kết thúc workshop',
                initial?.endTime ||
                  new Date(
                    new Date(nextWeek).getTime() + 2 * 3600000
                  ).toISOString(),
              ],
              [
                'registrationStartTime',
                'Mở đăng ký',
                initial?.registrationStartTime || new Date().toISOString(),
              ],
              [
                'registrationEndTime',
                'Đóng đăng ký',
                initial?.registrationEndTime || nextWeek,
              ],
            ].map(([name, title, value]) => (
              <label className="block text-sm font-medium" key={name}>
                {title}
                <input
                  name={name}
                  aria-label={title}
                  type="datetime-local"
                  required
                  defaultValue={localTime(value)}
                  className="mt-1 w-full rounded-lg border p-2.5"
                />
              </label>
            ))}
            <label className="block text-sm font-medium">
              Sức chứa
              <input
                name="capacity"
                aria-label="Sức chứa"
                type="number"
                required
                min={1}
                max={100000}
                defaultValue={initial?.capacity || 50}
                className="mt-1 w-full rounded-lg border p-2.5"
              />
            </label>
            <label className="block text-sm font-medium">
              Hình thức
              <select
                name="format"
                aria-label="Hình thức"
                defaultValue={initial?.format || 'OFFLINE'}
                className="mt-1 w-full rounded-lg border p-2.5"
              >
                <option value="OFFLINE">Trực tiếp</option>
                <option value="ONLINE">Online</option>
                <option value="HYBRID">Kết hợp</option>
              </select>
            </label>
          </div>
          {areas.map(([name, title, value]) => (
            <label className="block text-sm font-medium" key={name}>
              {title}
              <textarea
                name={name}
                aria-label={title}
                defaultValue={value}
                maxLength={12000}
                className="mt-1 min-h-24 w-full rounded-lg border p-2.5"
              />
            </label>
          ))}
          <label className="block text-sm font-medium">
            Ảnh bìa
            <input
              name="coverUrl"
              aria-label="Ảnh bìa workshop"
              defaultValue={initial?.coverUrl || ''}
              className="mt-1 w-full rounded-lg border p-2.5"
            />
            <input
              type="file"
              accept="image/png,image/jpeg,image/webp"
              className="mt-2 text-xs"
              onChange={(e) => {
                const file = e.currentTarget.files?.[0]
                const input = e.currentTarget.form?.elements.namedItem(
                  'coverUrl'
                ) as HTMLInputElement
                if (!file) return
                const data = new FormData()
                data.append('file', file)
                void api
                  .upload<{ url: string }>('/api/v1/media', data)
                  .then((response) => {
                    if (response.data) input.value = response.data.url
                  })
                  .catch((error) => toast.error(errorText(error)))
              }}
            />
          </label>
          <Button type="submit" disabled={busy}>
            {busy
              ? 'Đang lưu…'
              : revision
                ? 'Gửi thay đổi để duyệt'
                : 'Lưu bản nháp'}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}
export function BusinessWorkshops() {
  const { data, loading, error, refresh } = useResource<Workshop[]>(
    '/api/v1/business/workshops'
  )
  const profile = useResource<Company>('/api/v1/business/profile')
  const revisions = useResource<Revision[]>('/api/v1/business/revisions')
  const [open, setOpen] = useState(false),
    [editing, setEditing] = useState<Workshop>()
  function editor(workshop?: Workshop) {
    setEditing(workshop)
    setOpen(true)
  }
  async function submit(w: Workshop) {
    try {
      await api.post(`/api/v1/business/workshops/${w.id}/submit`)
      toast.success('Đã gửi workshop để duyệt')
      await refresh()
    } catch (e) {
      toast.error(errorText(e))
    }
  }
  async function cancel(w: Workshop) {
    const reason = window.prompt('Lý do đề nghị hủy sự kiện')
    if (!reason?.trim()) return
    try {
      await api.post(
        `/api/v1/business/workshops/${w.id}/cancellation-requests`,
        { reason }
      )
      toast.success('Đã gửi yêu cầu hủy')
      await revisions.refresh()
    } catch (e) {
      toast.error(errorText(e))
    }
  }
  return (
    <section className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-3xl font-bold">Workshop của doanh nghiệp</h1>
          <p className="mt-2 text-sm text-slate-500">
            Tạo chương trình sắp tới, gửi duyệt và giới thiệu đến sinh viên.
          </p>
        </div>
        <Button
          disabled={profile.data?.status !== 'APPROVED'}
          onClick={() => editor()}
        >
          Tạo workshop
        </Button>
      </div>
      {profile.data?.status !== 'APPROVED' && (
        <p className="rounded-lg bg-amber-50 p-4 text-sm text-amber-700">
          Hồ sơ doanh nghiệp phải được duyệt trước khi tạo workshop.
        </p>
      )}
      <LoadState loading={loading} error={error} retry={() => void refresh()} />
      <div className="grid gap-5 md:grid-cols-2">
        {data?.map((w) => (
          <article className="rounded-2xl border bg-white p-5" key={w.id}>
            <Status value={w.status} />
            <h2 className="mt-3 text-xl font-semibold">{w.title}</h2>
            <p className="mt-2 text-sm text-slate-500">
              {dateText(w.startTime)} · {w.room}
            </p>
            <p className="mt-1 text-sm text-slate-500">
              Đã giữ {w.capacity - w.availableSeats} / {w.capacity} chỗ
            </p>
            {w.reviewReason && (
              <p className="mt-3 rounded-lg bg-slate-50 p-3 text-sm">
                Ghi chú: {w.reviewReason}
              </p>
            )}
            <div className="mt-4 flex flex-wrap gap-2">
              {['DRAFT', 'REJECTED'].includes(w.status) && (
                <>
                  <Button variant="outline" size="sm" onClick={() => editor(w)}>
                    Sửa nháp
                  </Button>
                  <Button size="sm" onClick={() => void submit(w)}>
                    Gửi duyệt
                  </Button>
                </>
              )}
              {['PUBLISHED', 'CLOSED'].includes(w.status) && (
                <>
                  <Button variant="outline" size="sm" onClick={() => editor(w)}>
                    Đề nghị thay đổi
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-rose-600"
                    onClick={() => void cancel(w)}
                  >
                    Đề nghị hủy
                  </Button>
                </>
              )}
            </div>
            <div className="mt-4">
              <WorkshopPreview workshop={w} />
            </div>
          </article>
        ))}
      </div>
      {data?.length === 0 && (
        <div className="rounded-xl border border-dashed p-10 text-center text-slate-500">
          Chưa có workshop. Tạo workshop đầu tiên để bắt đầu.
        </div>
      )}
      {!!revisions.data?.length && (
        <section className="rounded-2xl border bg-white p-5">
          <h2 className="mb-4 text-xl font-semibold">Yêu cầu thay đổi</h2>
          {revisions.data.map((r) => (
            <div
              key={r.id}
              className="mb-3 flex flex-wrap items-center gap-3 rounded-lg bg-slate-50 p-3"
            >
              <span className="text-sm">
                {r.workshopTitle} ·{' '}
                {r.cancellation ? 'Hủy sự kiện' : 'Cập nhật nội dung'}
              </span>
              <Status value={r.status} />
              {r.reason && (
                <span className="text-xs text-slate-500">{r.reason}</span>
              )}
            </div>
          ))}
        </section>
      )}
      <WorkshopEditor
        open={open}
        onOpenChange={setOpen}
        initial={editing}
        onSaved={() => {
          void refresh()
          void revisions.refresh()
        }}
      />
    </section>
  )
}
