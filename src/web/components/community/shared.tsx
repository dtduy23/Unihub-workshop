'use client'
import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { api } from '@/lib/api-client'
import { Navbar } from '@/components/student/navbar'
import { Button } from '@/components/ui/button'

export type SessionUser = {
  id: string
  fullName: string
  studentId: string
  role: 'STUDENT' | 'STAFF' | 'BUSINESS' | 'ADMIN'
}
export type Company = {
  id: string
  ownerUserId: string
  name: string
  slug: string
  description: string
  industry: string
  website: string
  address: string
  contact: string
  logoUrl: string
  coverUrl: string
  status: string
  reviewReason: string
  followers: number
  following: boolean
}
export type Workshop = {
  id: string
  title: string
  description?: string
  companyName?: string
  speaker: string
  room: string
  startTime: string
  endTime: string
  registrationStartTime: string
  registrationEndTime: string
  capacity: number
  availableSeats: number
  status: string
  reviewReason: string
  price: number
  summary?: string
  coverUrl: string
  audience: string
  benefits: string
  preparation: string
  agenda: string
  format: string
}
export type Post = {
  id: string
  authorUserId: string
  authorName: string
  companyId?: string
  companyName: string
  companySlug: string
  workshopId?: string
  workshop?: Workshop
  kind: string
  title: string
  content: string
  topic: string
  mediaIds: string[]
  status: string
  likes: number
  comments: number
  liked: boolean
  bookmarked: boolean
  createdAt: string
  publishedAt?: string
}
export type FeedPage = { items: Post[]; nextCursor: string }
export type Comment = {
  id: string
  authorUserId: string
  authorName: string
  parentId?: string
  content: string
  createdAt: string
}
export type Revision = {
  id: string
  workshopId: string
  workshopTitle: string
  companyName: string
  payload: Workshop
  cancellation: boolean
  status: string
  reason: string
}
export type Report = {
  id: string
  postId?: string
  commentId?: string
  content: string
  reason: string
  status: string
  resolution: string
}

export const labels: Record<string, string> = {
  STUDENT: 'Sinh viên',
  STAFF: 'Nhân sự',
  BUSINESS: 'Doanh nghiệp',
  ADMIN: 'Quản trị',
  PENDING: 'Chờ xét duyệt',
  APPROVED: 'Đã duyệt',
  REJECTED: 'Cần chỉnh sửa',
  SUSPENDED: 'Đình chỉ',
  DRAFT: 'Bản nháp',
  PENDING_REVIEW: 'Chờ duyệt',
  PUBLISHED: 'Đã đăng',
  CLOSED: 'Đã đóng',
  CANCELLED: 'Đã hủy',
  DELETED: 'Đã xóa',
  HIDDEN: 'Đã ẩn',
  RESOLVED: 'Đã xử lý',
}
export const dateText = (value: string) =>
  new Date(value).toLocaleString('vi-VN', {
    timeZone: 'Asia/Ho_Chi_Minh',
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
export const errorText = (error: unknown) =>
  error instanceof Error ? error.message : 'Không thể xử lý. Vui lòng thử lại.'
export const mediaURL = (id: string) => `/api/backend/api/v1/media/${id}`
export function Status({ value }: { value: string }) {
  return (
    <span
      className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold ${['APPROVED', 'PUBLISHED'].includes(value) ? 'bg-emerald-50 text-emerald-700' : ['SUSPENDED', 'REJECTED', 'CANCELLED', 'HIDDEN'].includes(value) ? 'bg-rose-50 text-rose-700' : 'bg-indigo-50 text-indigo-700'}`}
    >
      {labels[value] || value}
    </span>
  )
}
export function useResource<T>(endpoint: string) {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const refresh = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const response = await api.get<T>(endpoint)
      setData(response.data ?? null)
    } catch (error) {
      setError(errorText(error))
    } finally {
      setLoading(false)
    }
  }, [endpoint])
  useEffect(() => {
    void refresh()
  }, [refresh])
  return { data, loading, error, refresh }
}
export function LoadState({
  loading,
  error,
  retry,
}: {
  loading: boolean
  error: string
  retry: () => void
}) {
  if (loading)
    return <p className="py-12 text-center text-slate-500">Đang tải…</p>
  if (error)
    return (
      <div
        role="alert"
        className="rounded-xl border border-rose-200 bg-rose-50 p-5 text-rose-700"
      >
        <p>{error}</p>
        <Button variant="outline" onClick={retry} className="mt-3">
          Thử lại
        </Button>
      </div>
    )
  return null
}
export function CommunityShell({
  title,
  description,
  children,
}: {
  title: string
  description?: string
  children: ReactNode
}) {
  return (
    <div className="min-h-screen bg-slate-50">
      <Navbar />
      <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
        <div className="mb-8">
          <p className="mb-2 text-xs font-semibold uppercase tracking-widest text-indigo-600">
            UniHub / Kết nối & học hỏi
          </p>
          <h1 className="text-3xl font-bold tracking-tight text-slate-900">
            {title}
          </h1>
          {description && <p className="mt-2 text-slate-500">{description}</p>}
        </div>
        {children}
      </main>
    </div>
  )
}
