'use client'
import { useSessionUser } from '@/hooks/use-session-user'
import { useEffect, useState, type FormEvent } from 'react'
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
  type Post,
  type Workshop,
  type SessionUser,
  errorText,
  mediaURL,
} from './shared'

export function PostComposer({
  open,
  onOpenChange,
  initial,
  onSaved,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  initial?: Post
  onSaved: () => void
}) {
  const user = useSessionUser() as SessionUser | null
  const [title, setTitle] = useState(''),
    [content, setContent] = useState(''),
    [topic, setTopic] = useState('')
  const [kind, setKind] = useState('COMMUNITY'),
    [workshopId, setWorkshopId] = useState(''),
    [images, setImages] = useState<string[]>([])
  const [workshops, setWorkshops] = useState<Workshop[]>([]),
    [busy, setBusy] = useState(false),
    [preview, setPreview] = useState(false)
  useEffect(() => {
    if (!open) return
    setTitle(initial?.title || '')
    setContent(initial?.content || '')
    setTopic(initial?.topic || '')
    setKind(initial?.kind || 'COMMUNITY')
    setWorkshopId(initial?.workshopId || '')
    setImages(initial?.mediaIds || [])
    setPreview(false)
    if (user?.role === 'BUSINESS' || user?.role === 'ADMIN') {
      void api
        .get<Workshop[]>(
          user.role === 'BUSINESS'
            ? '/api/v1/business/workshops'
            : '/api/v1/workshops'
        )
        .then((r) =>
          setWorkshops((r.data || []).filter((w) => w.status === 'PUBLISHED'))
        )
        .catch((e) => toast.error(errorText(e)))
    }
  }, [open, initial, user?.role])
  async function upload(file?: File) {
    if (!file) return
    if (images.length >= 4 || file.size > 5 * 1024 * 1024) {
      toast.error('Tối đa 4 ảnh; mỗi ảnh tối đa 5 MB.')
      return
    }
    setBusy(true)
    try {
      const form = new FormData()
      form.append('file', file)
      const response = await api.upload<{ id: string }>('/api/v1/media', form)
      if (response.data) setImages((list) => [...list, response.data!.id])
    } catch (e) {
      toast.error(errorText(e))
    } finally {
      setBusy(false)
    }
  }
  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const submitter = (event.nativeEvent as SubmitEvent)
      .submitter as HTMLButtonElement | null
    const status = submitter?.value || 'PUBLISHED'
    setBusy(true)
    try {
      const data = {
        title,
        content,
        topic,
        kind,
        workshopId: kind === 'WORKSHOP_ANNOUNCEMENT' ? workshopId : null,
        mediaIds: images,
        status,
      }
      if (initial) await api.patch(`/api/v1/posts/${initial.id}`, data)
      else await api.post('/api/v1/posts', data)
      toast.success(status === 'DRAFT' ? 'Đã lưu bản nháp' : 'Đã đăng bài')
      onOpenChange(false)
      onSaved()
    } catch (e) {
      toast.error(errorText(e))
    } finally {
      setBusy(false)
    }
  }
  const field =
    'w-full rounded-lg border border-slate-200 bg-white px-3 py-2.5 text-sm outline-none focus:border-indigo-500'
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {initial ? 'Chỉnh sửa bài viết' : 'Chia sẻ với cộng đồng'}
          </DialogTitle>
          <DialogDescription>
            Chia sẻ kiến thức hoặc giới thiệu một workshop đã được duyệt.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={save} className="space-y-4">
          {['BUSINESS', 'ADMIN'].includes(user?.role || '') && (
            <label className="block text-sm font-medium">
              Loại bài
              <select
                aria-label="Loại bài"
                className={`${field} mt-1`}
                value={kind}
                onChange={(e) => setKind(e.target.value)}
              >
                <option value="COMMUNITY">Bài cộng đồng</option>
                <option value="WORKSHOP_ANNOUNCEMENT">
                  Giới thiệu workshop
                </option>
              </select>
            </label>
          )}
          {kind === 'WORKSHOP_ANNOUNCEMENT' && (
            <label className="block text-sm font-medium">
              Workshop đã duyệt
              <select
                aria-label="Workshop đã duyệt"
                className={`${field} mt-1`}
                required
                value={workshopId}
                onChange={(e) => setWorkshopId(e.target.value)}
              >
                <option value="">Chọn workshop của bạn</option>
                {workshops.map((w) => (
                  <option key={w.id} value={w.id}>
                    {w.title}
                  </option>
                ))}
              </select>
              {workshops.length === 0 && (
                <p className="mt-2 text-xs text-amber-700">
                  Chưa có workshop đã duyệt. Hãy gửi workshop để admin duyệt
                  trước.
                </p>
              )}
            </label>
          )}
          <label className="block text-sm font-medium">
            Tiêu đề
            <input
              aria-label="Tiêu đề bài viết"
              className={`${field} mt-1`}
              required
              minLength={2}
              maxLength={200}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </label>
          <label className="block text-sm font-medium">
            Nội dung
            <textarea
              aria-label="Nội dung bài viết"
              className={`${field} mt-1 min-h-36`}
              required
              maxLength={12000}
              placeholder="Workshop này có gì? Bạn muốn chia sẻ điều gì?"
              value={content}
              onChange={(e) => setContent(e.target.value)}
            />
          </label>
          <label className="block text-sm font-medium">
            Chủ đề
            <input
              aria-label="Chủ đề bài viết"
              className={`${field} mt-1`}
              maxLength={80}
              placeholder="Công nghệ, nghề nghiệp, kỹ năng…"
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
            />
          </label>
          <div className="flex flex-wrap gap-3">
            {images.map((id) => (
              <div key={id} className="relative">
                <img
                  src={mediaURL(id)}
                  alt="Ảnh đính kèm"
                  className="h-24 w-24 rounded-lg object-cover"
                />
                <button
                  type="button"
                  className="absolute right-1 top-1 rounded bg-white px-2"
                  aria-label="Xóa ảnh"
                  onClick={() =>
                    setImages(images.filter((item) => item !== id))
                  }
                >
                  ×
                </button>
              </div>
            ))}
          </div>
          <label className="block text-sm text-slate-600">
            Thêm ảnh (PNG/JPG/WebP, tối đa 5 MB)
            <input
              type="file"
              accept="image/png,image/jpeg,image/webp"
              disabled={busy || images.length >= 4}
              className="mt-2 block w-full text-sm"
              onChange={(e) => {
                void upload(e.target.files?.[0])
                e.target.value = ''
              }}
            />
          </label>
          {preview && (
            <article className="rounded-xl bg-slate-50 p-5">
              <h3 className="font-bold">{title || 'Tiêu đề bài viết'}</h3>
              <p className="mt-3 whitespace-pre-wrap text-sm">{content}</p>
              {workshopId && (
                <p className="mt-4 text-sm text-indigo-700">
                  Workshop: {workshops.find((w) => w.id === workshopId)?.title}
                </p>
              )}
            </article>
          )}
          <div className="flex flex-wrap justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setPreview(!preview)}
            >
              Xem trước
            </Button>
            <Button
              type="submit"
              value="DRAFT"
              variant="outline"
              disabled={busy}
            >
              Lưu nháp
            </Button>
            <Button type="submit" value="PUBLISHED" disabled={busy}>
              {busy ? 'Đang lưu…' : 'Đăng bài'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
