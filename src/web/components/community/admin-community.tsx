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
  type Company,
  type Workshop,
  type Revision,
  type Report,
  useResource,
  LoadState,
  Status,
  dateText,
  errorText,
} from './shared'

async function review(endpoint: string, status: string, done: () => void) {
  const reason = window.prompt(
    'Ghi chú hoặc lý do xử lý',
    status === 'APPROVED' || status === 'PUBLISHED'
      ? 'Đã kiểm tra thông tin'
      : ''
  )
  if (!reason?.trim()) return
  try {
    await api.post(endpoint, { status, reason })
    toast.success('Đã cập nhật')
    done()
  } catch (e) {
    toast.error(errorText(e))
  }
}
export function AdminCompanies() {
  const { data, loading, error, refresh } = useResource<Company[]>(
    '/api/v1/admin/companies'
  )
  const [open, setOpen] = useState(false),
    [busy, setBusy] = useState(false)
  async function create(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    try {
      await api.post(
        '/api/v1/admin/business-accounts',
        Object.fromEntries(new FormData(event.currentTarget))
      )
      toast.success('Đã tạo tài khoản doanh nghiệp')
      setOpen(false)
      await refresh()
    } catch (e) {
      toast.error(errorText(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <section className="p-6 lg:p-10">
      <div className="mb-8 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-3xl font-bold">Doanh nghiệp</h1>
          <p className="mt-2 text-slate-500">
            Cấp tài khoản, xét duyệt hồ sơ và quản lý hoạt động.
          </p>
        </div>
        <Button onClick={() => setOpen(true)}>
          Cấp tài khoản doanh nghiệp
        </Button>
      </div>
      <LoadState loading={loading} error={error} retry={() => void refresh()} />
      <div className="space-y-4">
        {data?.map((c) => (
          <article key={c.id} className="rounded-2xl border bg-white p-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 className="text-xl font-bold">{c.name}</h2>
                <p className="mt-1 text-sm text-slate-500">
                  {c.slug} · {c.industry || 'Chưa cập nhật lĩnh vực'}
                </p>
              </div>
              <Status value={c.status} />
            </div>
            <p className="mt-4 whitespace-pre-wrap text-sm leading-7 text-slate-600">
              {c.description}
            </p>
            <p className="mt-3 text-sm text-slate-500">
              {c.contact} {c.website && `· ${c.website}`}
            </p>
            {c.reviewReason && (
              <p className="mt-3 text-sm">Ghi chú: {c.reviewReason}</p>
            )}
            <div className="mt-5 flex gap-2">
              <Button
                size="sm"
                disabled={c.status === 'APPROVED'}
                onClick={() =>
                  void review(
                    `/api/v1/admin/companies/${c.id}/review`,
                    'APPROVED',
                    () => void refresh()
                  )
                }
              >
                Duyệt hồ sơ
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() =>
                  void review(
                    `/api/v1/admin/companies/${c.id}/review`,
                    'REJECTED',
                    () => void refresh()
                  )
                }
              >
                Yêu cầu chỉnh sửa
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="text-rose-600"
                onClick={() =>
                  void review(
                    `/api/v1/admin/companies/${c.id}/review`,
                    'SUSPENDED',
                    () => void refresh()
                  )
                }
              >
                Đình chỉ
              </Button>
            </div>
          </article>
        ))}
      </div>
      {data?.length === 0 && (
        <p className="py-12 text-center text-slate-500">
          Chưa có tài khoản doanh nghiệp.
        </p>
      )}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Cấp tài khoản doanh nghiệp</DialogTitle>
            <DialogDescription>
              Doanh nghiệp đăng nhập bằng mã hoặc email. Hồ sơ mới sẽ ở trạng
              thái chờ duyệt.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={create} className="space-y-4">
            {[
              ['name', 'Tên doanh nghiệp', 'text'],
              ['slug', 'Slug (chữ thường, số, dấu gạch ngang)', 'text'],
              ['identifier', 'Mã đăng nhập', 'text'],
              ['email', 'Email đăng nhập', 'email'],
              ['password', 'Mật khẩu khởi tạo (ít nhất 8 ký tự)', 'password'],
            ].map(([name, label, type]) => (
              <label key={name} className="block text-sm font-medium">
                {label}
                <input
                  aria-label={label}
                  name={name}
                  type={type}
                  required
                  minLength={name === 'password' ? 8 : 1}
                  maxLength={name === 'password' ? 72 : 160}
                  autoComplete={name === 'password' ? 'new-password' : 'off'}
                  className="mt-2 w-full rounded-lg border p-3"
                />
              </label>
            ))}
            <Button type="submit" disabled={busy}>
              {busy ? 'Đang tạo…' : 'Tạo tài khoản'}
            </Button>
          </form>
        </DialogContent>
      </Dialog>
    </section>
  )
}
function StaffAssignment({ workshopId }: { workshopId: string }) {
  const { data, loading, error, refresh } = useResource<
    { id: string; fullName: string; assigned: string }[]
  >(`/api/v1/admin/workshops/${workshopId}/staff`)
  async function toggle(user: { id: string; assigned: string }) {
    try {
      if (user.assigned === 'true')
        await api.delete(
          `/api/v1/admin/workshops/${workshopId}/staff/${user.id}`
        )
      else
        await api.put(`/api/v1/admin/workshops/${workshopId}/staff/${user.id}`)
      await refresh()
    } catch (e) {
      toast.error(errorText(e))
    }
  }
  return (
    <div className="mt-4 space-y-2">
      <LoadState loading={loading} error={error} retry={() => void refresh()} />
      {data?.map((user) => (
        <label
          key={user.id}
          className="flex items-center gap-3 rounded-lg bg-slate-50 p-3 text-sm"
        >
          <input
            type="checkbox"
            checked={user.assigned === 'true'}
            onChange={() => void toggle(user)}
          />
          {user.fullName}
        </label>
      ))}
      {data?.length === 0 && (
        <p className="text-sm text-slate-500">Chưa có tài khoản staff.</p>
      )}
    </div>
  )
}
export function AdminWorkshopReviews() {
  const workshops = useResource<Workshop[]>('/api/v1/admin/workshop-reviews'),
    revisions = useResource<Revision[]>('/api/v1/admin/workshop-revisions'),
    all = useResource<Workshop[]>('/api/v1/workshops')
  const [selected, setSelected] = useState('')
  const refresh = () => {
    void workshops.refresh()
    void revisions.refresh()
    void all.refresh()
  }
  return (
    <section className="space-y-8 p-6 lg:p-10">
      <div>
        <h1 className="text-3xl font-bold">Duyệt workshop</h1>
        <p className="mt-2 text-slate-500">
          Kiểm tra chương trình và thay đổi của doanh nghiệp trước khi công bố
          cho thành viên.
        </p>
      </div>
      <LoadState
        loading={workshops.loading}
        error={workshops.error}
        retry={() => void workshops.refresh()}
      />
      <div className="space-y-4">
        {workshops.data?.map((w) => (
          <article className="rounded-2xl border bg-white p-6" key={w.id}>
            <Status value={w.status} />
            <h2 className="mt-3 text-xl font-bold">{w.title}</h2>
            <p className="mt-2 text-sm text-indigo-600">
              {w.companyName} · {w.speaker}
            </p>
            <p className="mt-2 text-sm text-slate-500">
              {dateText(w.startTime)} · {w.room} · {w.capacity} chỗ
            </p>
            {[
              ['Giới thiệu', w.description],
              ['Chương trình', w.agenda],
              ['Đối tượng', w.audience],
              ['Lợi ích', w.benefits],
              ['Chuẩn bị', w.preparation],
            ].map(
              ([label, value]) =>
                value && (
                  <p
                    key={label}
                    className="mt-3 whitespace-pre-wrap text-sm leading-6"
                  >
                    <strong>{label}: </strong>
                    {value}
                  </p>
                )
            )}
            <div className="mt-5 flex gap-2">
              <Button
                onClick={() =>
                  void review(
                    `/api/v1/admin/workshops/${w.id}/review`,
                    'PUBLISHED',
                    refresh
                  )
                }
              >
                Duyệt & công bố
              </Button>
              <Button
                variant="outline"
                onClick={() =>
                  void review(
                    `/api/v1/admin/workshops/${w.id}/review`,
                    'REJECTED',
                    refresh
                  )
                }
              >
                Trả lại để sửa
              </Button>
            </div>
          </article>
        ))}
      </div>
      {workshops.data?.length === 0 && (
        <p className="rounded-xl border border-dashed p-6 text-sm text-slate-500">
          Không có workshop đang chờ duyệt.
        </p>
      )}
      <section>
        <h2 className="mb-4 text-xl font-bold">Thay đổi / hủy sự kiện</h2>
        <LoadState
          loading={revisions.loading}
          error={revisions.error}
          retry={() => void revisions.refresh()}
        />
        {revisions.data
          ?.filter((r) => r.status === 'PENDING')
          .map((r) => (
            <article
              key={r.id}
              className="mb-4 rounded-2xl border bg-white p-6"
            >
              <h3 className="font-semibold">
                {r.workshopTitle} · {r.companyName}
              </h3>
              <p className="mt-2 text-sm text-amber-700">
                {r.cancellation
                  ? 'Đề nghị hủy sự kiện'
                  : 'Đề nghị cập nhật nội dung / lịch'}
              </p>
              {r.reason && <p className="mt-2 text-sm">Lý do: {r.reason}</p>}
              {!r.cancellation && (
                <div className="mt-4 rounded-lg bg-slate-50 p-4 text-sm">
                  <p className="font-semibold">{r.payload.title}</p>
                  <p className="mt-2">
                    {dateText(r.payload.startTime)} –{' '}
                    {dateText(r.payload.endTime)}
                  </p>
                  <p>
                    {r.payload.room} · {r.payload.capacity} chỗ
                  </p>
                  {[
                    ['Diễn giả', r.payload.speaker],
                    ['Giới thiệu', r.payload.description],
                    ['Chương trình', r.payload.agenda],
                    ['Đối tượng', r.payload.audience],
                    ['Lợi ích', r.payload.benefits],
                    ['Chuẩn bị', r.payload.preparation],
                  ].map(
                    ([label, value]) =>
                      value && (
                        <p key={label} className="mt-2 whitespace-pre-wrap">
                          <strong>{label}: </strong>
                          {value}
                        </p>
                      )
                  )}
                </div>
              )}
              <div className="mt-4 flex gap-2">
                <Button
                  onClick={() =>
                    void review(
                      `/api/v1/admin/workshop-revisions/${r.id}/review`,
                      'APPROVED',
                      refresh
                    )
                  }
                >
                  Chấp thuận
                </Button>
                <Button
                  variant="outline"
                  onClick={() =>
                    void review(
                      `/api/v1/admin/workshop-revisions/${r.id}/review`,
                      'REJECTED',
                      refresh
                    )
                  }
                >
                  Từ chối
                </Button>
              </div>
            </article>
          ))}
      </section>
      <section className="rounded-2xl border bg-white p-6">
        <h2 className="text-xl font-bold">Phân công nhân sự check-in</h2>
        <select
          aria-label="Workshop cần phân công"
          className="mt-4 w-full rounded-lg border p-3"
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
        >
          <option value="">Chọn workshop</option>
          {all.data
            ?.filter((w) => w.status !== 'DELETED')
            .map((w) => (
              <option key={w.id} value={w.id}>
                {w.title}
              </option>
            ))}
        </select>
        {selected && <StaffAssignment workshopId={selected} />}
      </section>
    </section>
  )
}
export function AdminModeration() {
  const { data, loading, error, refresh } = useResource<Report[]>(
    '/api/v1/admin/reports'
  )
  return (
    <section className="p-6 lg:p-10">
      <h1 className="text-3xl font-bold">Kiểm duyệt cộng đồng</h1>
      <p className="mb-8 mt-2 text-slate-500">
        Xử lý báo cáo và lưu lý do cho từng hành động.
      </p>
      <LoadState loading={loading} error={error} retry={() => void refresh()} />
      <div className="space-y-4">
        {data?.map((r) => (
          <article className="rounded-2xl border bg-white p-6" key={r.id}>
            <Status value={r.status} />
            <h2 className="mt-3 font-semibold">
              {r.postId ? 'Báo cáo bài viết' : 'Báo cáo bình luận'}
            </h2>
            <p className="mt-3 rounded-lg bg-slate-50 p-4 text-sm whitespace-pre-wrap">
              {r.content}
            </p>
            <p className="mt-3 text-sm">
              <strong>Lý do:</strong> {r.reason}
            </p>
            {r.resolution && (
              <p className="mt-3 text-sm text-slate-500">
                Xử lý: {r.resolution}
              </p>
            )}
            {r.status === 'PENDING' && (
              <div className="mt-4 flex gap-2">
                <Button
                  variant="outline"
                  onClick={() =>
                    void review(
                      `/api/v1/admin/${r.postId ? 'posts' : 'comments'}/${r.postId || r.commentId}/moderate`,
                      'HIDDEN',
                      () => void refresh()
                    )
                  }
                >
                  Ẩn nội dung
                </Button>
                <Button
                  onClick={() =>
                    void review(
                      `/api/v1/admin/reports/${r.id}/resolve`,
                      'RESOLVED',
                      () => void refresh()
                    )
                  }
                >
                  Hoàn tất xử lý
                </Button>
              </div>
            )}
          </article>
        ))}
      </div>
      {data?.length === 0 && (
        <p className="py-12 text-center text-slate-500">
          Chưa có báo cáo vi phạm.
        </p>
      )}
    </section>
  )
}
