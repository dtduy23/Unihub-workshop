'use client'
import Link from 'next/link'
import { api } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import {
  type Company,
  useResource,
  LoadState,
  Status,
  errorText,
} from './shared'

export function BusinessDashboard() {
  const profile = useResource<Company>('/api/v1/business/profile'),
    stats = useResource<Record<string, number>>('/api/v1/business/stats')
  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">
        Xin chào, {profile.data?.name || 'doanh nghiệp'}
      </h1>
      <LoadState
        loading={profile.loading}
        error={profile.error}
        retry={() => void profile.refresh()}
      />
      {profile.data && (
        <section className="rounded-2xl border bg-white p-6">
          <div className="flex flex-wrap items-center gap-3">
            <h2 className="text-lg font-semibold">Trạng thái hồ sơ</h2>
            <Status value={profile.data.status} />
          </div>
          <p className="mt-3 text-sm text-slate-500">
            {profile.data.status === 'APPROVED'
              ? 'Bạn có thể tạo workshop và chia sẻ bài viết với cộng đồng.'
              : 'Hoàn thiện hồ sơ và chờ admin duyệt để tạo workshop, đăng bài.'}
          </p>
          {profile.data.reviewReason && (
            <p className="mt-3 rounded-lg bg-slate-50 p-3 text-sm">
              Ghi chú: {profile.data.reviewReason}
            </p>
          )}
          <Button asChild variant="outline" className="mt-4">
            <Link href="/business/profile">Xem hồ sơ</Link>
          </Button>
        </section>
      )}
      <LoadState
        loading={stats.loading}
        error={stats.error}
        retry={() => void stats.refresh()}
      />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {[
          ['workshops', 'Workshop'],
          ['posts', 'Bài đăng'],
          ['followers', 'Người theo dõi'],
          ['registrations', 'Vé hợp lệ'],
          ['checkins', 'Đã check-in'],
        ].map(([key, label]) => (
          <div className="rounded-2xl border bg-white p-5" key={key}>
            <p className="text-sm text-slate-500">{label}</p>
            <p className="mt-2 text-3xl font-bold text-slate-900">
              {stats.data?.[key] ?? 0}
            </p>
          </div>
        ))}
      </div>
      <div className="flex gap-3">
        <Button asChild>
          <Link href="/business/workshops">Quản lý workshop</Link>
        </Button>
        <Button variant="outline" asChild>
          <Link href="/business/posts">Soạn bài giới thiệu</Link>
        </Button>
      </div>
    </div>
  )
}
export function BusinessProfile() {
  const { data, loading, error, refresh } = useResource<Company>(
    '/api/v1/business/profile'
  )
  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    try {
      await api.patch('/api/v1/business/profile', Object.fromEntries(form))
      toast.success('Đã lưu hồ sơ')
      await refresh()
    } catch (e) {
      toast.error(errorText(e))
    }
  }
  async function upload(
    input: HTMLInputElement,
    target: HTMLInputElement | null
  ) {
    const file = input.files?.[0]
    if (!file || !target) return
    const body = new FormData()
    body.append('file', file)
    try {
      const response = await api.upload<{ url: string }>('/api/v1/media', body)
      if (response.data) target.value = response.data.url
      toast.success('Đã tải ảnh. Nhấn lưu hồ sơ để cập nhật.')
    } catch (e) {
      toast.error(errorText(e))
    } finally {
      input.value = ''
    }
  }
  return (
    <section>
      <h1 className="mb-6 text-3xl font-bold">Hồ sơ doanh nghiệp</h1>
      <LoadState loading={loading} error={error} retry={() => void refresh()} />
      {data && (
        <form
          key={data.id + data.status}
          onSubmit={save}
          className="max-w-3xl space-y-5 rounded-2xl border bg-white p-6"
        >
          <Status value={data.status} />
          {data.reviewReason && (
            <p className="text-sm text-slate-600">
              Ghi chú xét duyệt: {data.reviewReason}
            </p>
          )}
          {[
            ['name', 'Tên doanh nghiệp', data.name],
            ['industry', 'Lĩnh vực', data.industry],
            ['website', 'Website', data.website],
            ['address', 'Địa chỉ', data.address],
            ['contact', 'Liên hệ công khai', data.contact],
          ].map(([name, title, value]) => (
            <label key={name} className="block text-sm font-medium">
              {title}
              <input
                aria-label={title}
                name={name}
                defaultValue={value}
                required={name === 'name'}
                maxLength={500}
                className="mt-2 w-full rounded-lg border p-3"
              />
            </label>
          ))}
          <label className="block text-sm font-medium">
            Giới thiệu
            <textarea
              aria-label="Giới thiệu doanh nghiệp"
              name="description"
              defaultValue={data.description}
              maxLength={12000}
              className="mt-2 min-h-32 w-full rounded-lg border p-3"
            />
          </label>
          {[
            ['logoUrl', 'Logo', data.logoUrl],
            ['coverUrl', 'Ảnh bìa', data.coverUrl],
          ].map(([name, title, value]) => (
            <label key={name} className="block text-sm font-medium">
              {title}
              <input
                name={name}
                aria-label={`${title} URL`}
                defaultValue={value}
                className="mt-2 w-full rounded-lg border p-3"
              />
              <input
                type="file"
                accept="image/png,image/jpeg,image/webp"
                className="mt-2 text-xs"
                onChange={(e) => {
                  const target = e.currentTarget.form?.elements.namedItem(
                    name
                  ) as HTMLInputElement | null
                  void upload(e.currentTarget, target)
                }}
              />
            </label>
          ))}
          <Button type="submit" disabled={data.status === 'SUSPENDED'}>
            Lưu hồ sơ{data.status === 'REJECTED' ? ' và gửi lại' : ''}
          </Button>
        </form>
      )}
    </section>
  )
}
