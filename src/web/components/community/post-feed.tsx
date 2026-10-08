'use client'
import { useSessionUser } from '@/hooks/use-session-user'
import Link from 'next/link'
import { useCallback, useEffect, useState } from 'react'
import {
  Heart,
  MessageCircle,
  Bookmark,
  Share2,
  Plus,
  Flag,
  Pencil,
  Trash2,
} from 'lucide-react'
import { api } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { PostComposer } from './post-composer'
import {
  type Post,
  type FeedPage,
  type SessionUser,
  CommunityShell,
  dateText,
  errorText,
  mediaURL,
  Status,
  LoadState,
} from './shared'
import { WorkshopPreview } from './workshop-preview'

export function PostCard({
  post,
  refresh,
}: {
  post: Post
  refresh: () => void
}) {
  const user = useSessionUser() as SessionUser | null
  const [editing, setEditing] = useState(false),
    [busy, setBusy] = useState(false)
  async function interact(kind: 'like' | 'bookmark') {
    setBusy(true)
    try {
      if (kind === 'like' ? post.liked : post.bookmarked)
        await api.delete(`/api/v1/posts/${post.id}/${kind}`)
      else await api.put(`/api/v1/posts/${post.id}/${kind}`)
      refresh()
    } catch (e) {
      toast.error(errorText(e))
    } finally {
      setBusy(false)
    }
  }
  async function remove() {
    if (!window.confirm('Xóa bài viết này?')) return
    try {
      await api.delete(`/api/v1/posts/${post.id}`)
      toast.success('Đã xóa bài')
      refresh()
    } catch (e) {
      toast.error(errorText(e))
    }
  }
  async function report() {
    const reason = window.prompt('Lý do báo cáo bài viết')
    if (!reason?.trim()) return
    try {
      await api.post('/api/v1/reports', { postId: post.id, reason })
      toast.success('Đã gửi báo cáo cho admin')
    } catch (e) {
      toast.error(errorText(e))
    }
  }
  return (
    <article className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div className="p-5 sm:p-6">
        <header className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-full bg-indigo-100 text-lg font-bold text-indigo-600">
              {(post.companyName || post.authorName).slice(0, 1)}
            </div>
            <div>
              {post.companySlug ? (
                <Link
                  className="font-semibold text-slate-900 hover:text-indigo-600"
                  href={`/companies/${post.companySlug}`}
                >
                  {post.companyName}
                </Link>
              ) : (
                <p className="font-semibold text-slate-900">
                  {post.authorName}
                </p>
              )}
              <p className="mt-0.5 text-xs text-slate-500">
                {dateText(post.publishedAt || post.createdAt)}
                {post.topic && ` · ${post.topic}`}
              </p>
            </div>
          </div>
          {post.status !== 'PUBLISHED' && <Status value={post.status} />}
        </header>
        <Link href={`/posts/${post.id}`}>
          <h2 className="mt-5 text-xl font-bold leading-snug text-slate-900 hover:text-indigo-600">
            {post.title}
          </h2>
        </Link>
        <p className="mt-3 whitespace-pre-wrap break-words text-sm leading-7 text-slate-600">
          {post.content}
        </p>
        {post.mediaIds.length > 0 && (
          <div
            className={`mt-4 grid gap-2 ${post.mediaIds.length > 1 ? 'grid-cols-2' : 'grid-cols-1'}`}
          >
            {post.mediaIds.map((id) => (
              <img
                key={id}
                src={mediaURL(id)}
                alt={post.title}
                className="max-h-96 w-full rounded-xl object-cover"
                loading="lazy"
              />
            ))}
          </div>
        )}
        {post.workshop && (
          <div className="mt-5">
            <WorkshopPreview workshop={post.workshop} />
          </div>
        )}
        <div className="mt-5 flex flex-wrap items-center gap-1 border-t border-slate-100 pt-3">
          <Button
            size="sm"
            variant="ghost"
            disabled={busy || post.status !== 'PUBLISHED'}
            className={post.liked ? 'text-rose-600' : 'text-slate-500'}
            onClick={() => void interact('like')}
          >
            <Heart
              className={`mr-1.5 h-4 w-4 ${post.liked ? 'fill-current' : ''}`}
            />
            {post.likes}
          </Button>
          <Button size="sm" variant="ghost" asChild>
            <Link href={`/posts/${post.id}`}>
              <MessageCircle className="mr-1.5 h-4 w-4" />
              {post.comments}
            </Link>
          </Button>
          <Button
            size="sm"
            variant="ghost"
            disabled={busy || post.status !== 'PUBLISHED'}
            onClick={() => void interact('bookmark')}
            className={post.bookmarked ? 'text-indigo-600' : ''}
          >
            <Bookmark
              className={`mr-1.5 h-4 w-4 ${post.bookmarked ? 'fill-current' : ''}`}
            />
            Lưu
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              void navigator.clipboard
                .writeText(`${window.location.origin}/posts/${post.id}`)
                .then(() => toast.success('Đã sao chép liên kết'))
                .catch(() => toast.error('Không thể sao chép liên kết'))
            }}
          >
            <Share2 className="mr-1.5 h-4 w-4" />
            Chia sẻ
          </Button>
          {user?.id === post.authorUserId && (
            <>
              <Button
                size="sm"
                variant="ghost"
                disabled={post.status === 'HIDDEN'}
                onClick={() => setEditing(true)}
              >
                <Pencil className="mr-1 h-4 w-4" />
                Sửa
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="text-rose-600"
                onClick={() => void remove()}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </>
          )}
          {post.status === 'PUBLISHED' && (
            <Button
              size="sm"
              variant="ghost"
              className="ml-auto text-slate-400"
              aria-label="Báo cáo bài viết"
              onClick={() => void report()}
            >
              <Flag className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>
      <PostComposer
        open={editing}
        onOpenChange={setEditing}
        initial={post}
        onSaved={refresh}
      />
    </article>
  )
}
export function PostFeed({
  fixedMode,
  companyId,
}: {
  fixedMode?: string
  companyId?: string
}) {
  const [mode, setMode] = useState(fixedMode || ''),
    [topic, setTopic] = useState(''),
    [posts, setPosts] = useState<Post[]>([]),
    [cursor, setCursor] = useState(''),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(''),
    [composing, setComposing] = useState(false)
  const user = useSessionUser() as SessionUser | null
  const load = useCallback(
    async (next = '') => {
      setLoading(true)
      setError('')
      const query = new URLSearchParams({ mode, topic, cursor: next })
      const endpoint = companyId
        ? `/api/v1/companies/${companyId}/posts`
        : '/api/v1/feed'
      try {
        const response = await api.get<FeedPage>(`${endpoint}?${query}`)
        const data = response.data
        if (data) {
          setPosts((list) => (next ? [...list, ...data.items] : data.items))
          setCursor(data.nextCursor)
        }
      } catch (e) {
        setError(errorText(e))
      } finally {
        setLoading(false)
      }
    },
    [mode, topic, companyId]
  )
  useEffect(() => {
    void load()
  }, [load])
  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center gap-2">
        {!fixedMode && !companyId && (
          <>
            <Button
              variant={mode === '' ? 'default' : 'outline'}
              onClick={() => setMode('')}
            >
              Mới nhất
            </Button>
            <Button
              variant={mode === 'following' ? 'default' : 'outline'}
              onClick={() => setMode('following')}
            >
              Đang theo dõi
            </Button>
            <Button
              variant={mode === 'mine' ? 'default' : 'outline'}
              onClick={() => setMode('mine')}
            >
              Bài của tôi
            </Button>
          </>
        )}
        {!companyId &&
          ['STUDENT', 'BUSINESS', 'ADMIN'].includes(user?.role || '') && (
            <Button className="ml-auto" onClick={() => setComposing(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Viết bài
            </Button>
          )}
      </div>
      <label className="block text-sm text-slate-600">
        Lọc chủ đề
        <select
          aria-label="Lọc chủ đề"
          className="ml-3 rounded-lg border bg-white p-2"
          value={topic}
          onChange={(e) => setTopic(e.target.value)}
        >
          <option value="">Tất cả</option>
          <option>Công nghệ</option>
          <option>Nghề nghiệp</option>
          <option>Kỹ năng</option>
        </select>
      </label>
      {posts.map((post) => (
        <PostCard key={post.id} post={post} refresh={() => void load()} />
      ))}
      <LoadState loading={loading} error={error} retry={() => void load()} />
      {!loading && !error && posts.length === 0 && (
        <div className="rounded-2xl border border-dashed bg-white p-12 text-center text-slate-500">
          {mode === 'following'
            ? 'Theo dõi doanh nghiệp để cập nhật bài viết mới ở đây.'
            : 'Chưa có bài viết. Hãy chia sẻ điều đầu tiên với cộng đồng.'}
        </div>
      )}
      {cursor && !loading && (
        <Button
          variant="outline"
          className="w-full"
          onClick={() => void load(cursor)}
        >
          Xem thêm bài viết
        </Button>
      )}
      <PostComposer
        open={composing}
        onOpenChange={setComposing}
        onSaved={() => void load()}
      />
    </div>
  )
}
export function FeedScreen({ saved = false }: { saved?: boolean }) {
  return (
    <CommunityShell
      title={saved ? 'Bài viết đã lưu' : 'Cộng đồng UniHub'}
      description={
        saved
          ? 'Những ý tưởng và workshop bạn muốn quay lại.'
          : 'Chia sẻ kiến thức, gặp gỡ doanh nghiệp và khám phá workshop sắp diễn ra.'
      }
    >
      <div className="mx-auto max-w-3xl">
        <PostFeed fixedMode={saved ? 'saved' : undefined} />
      </div>
    </CommunityShell>
  )
}
