'use client'
import { useState } from 'react'
import { useParams } from 'next/navigation'
import { api } from '@/lib/api-client'
import { useSessionUser } from '@/hooks/use-session-user'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { PostCard } from './post-feed'
import {
  type Post,
  type Comment,
  CommunityShell,
  useResource,
  LoadState,
  dateText,
  errorText,
} from './shared'

function Comments({ postId }: { postId: string }) {
  const { data, loading, error, refresh } = useResource<Comment[]>(
    `/api/v1/posts/${postId}/comments`
  )
  const user = useSessionUser()
  const [text, setText] = useState(''),
    [parent, setParent] = useState<Comment | null>(null),
    [busy, setBusy] = useState(false)
  async function add(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    try {
      await api.post(`/api/v1/posts/${postId}/comments`, {
        content: text,
        parentId: parent?.id || null,
      })
      setText('')
      setParent(null)
      await refresh()
    } catch (e) {
      toast.error(errorText(e))
    } finally {
      setBusy(false)
    }
  }
  async function edit(comment: Comment) {
    const content = window.prompt('Sửa bình luận', comment.content)
    if (!content?.trim()) return
    try {
      await api.patch(`/api/v1/comments/${comment.id}`, { content })
      await refresh()
    } catch (e) {
      toast.error(errorText(e))
    }
  }
  async function remove(comment: Comment) {
    if (!window.confirm('Xóa bình luận này?')) return
    try {
      await api.delete(`/api/v1/comments/${comment.id}`)
      await refresh()
    } catch (e) {
      toast.error(errorText(e))
    }
  }
  async function report(comment: Comment) {
    const reason = window.prompt('Lý do báo cáo bình luận')
    if (!reason?.trim()) return
    try {
      await api.post('/api/v1/reports', { commentId: comment.id, reason })
      toast.success('Đã gửi báo cáo')
    } catch (e) {
      toast.error(errorText(e))
    }
  }
  const render = (comment: Comment, reply = false) => (
    <div
      key={comment.id}
      className={`${reply ? 'ml-6 border-l-2 border-indigo-100 pl-4' : ''} mt-4`}
    >
      <div className="rounded-xl bg-slate-50 p-4">
        <p className="text-sm font-semibold">
          {comment.authorName}
          <span className="ml-2 text-xs font-normal text-slate-400">
            {dateText(comment.createdAt)}
          </span>
        </p>
        <p className="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-slate-600">
          {comment.content}
        </p>
      </div>
      <div className="mt-1 flex gap-2">
        {!reply && (
          <Button variant="ghost" size="sm" onClick={() => setParent(comment)}>
            Trả lời
          </Button>
        )}
        {comment.authorUserId === user?.id && (
          <>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => void edit(comment)}
            >
              Sửa
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => void remove(comment)}
            >
              Xóa
            </Button>
          </>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="text-slate-400"
          onClick={() => void report(comment)}
        >
          Báo cáo
        </Button>
      </div>
    </div>
  )
  return (
    <section className="rounded-2xl border bg-white p-5 sm:p-6">
      <h2 className="text-lg font-bold">Bình luận</h2>
      <LoadState loading={loading} error={error} retry={() => void refresh()} />
      {data
        ?.filter((c) => !c.parentId)
        .map((c) => (
          <div key={c.id}>
            {render(c)}
            {data
              .filter((reply) => reply.parentId === c.id)
              .map((reply) => render(reply, true))}
          </div>
        ))}
      {data
        ?.filter(
          (c) => c.parentId && !data.some((parent) => parent.id === c.parentId)
        )
        .map((c) => render(c, true))}
      <form onSubmit={add} className="mt-5 space-y-3">
        {parent && (
          <div className="flex items-center justify-between rounded-lg bg-indigo-50 p-3 text-xs text-indigo-700">
            Trả lời {parent.authorName}
            <button type="button" onClick={() => setParent(null)}>
              Hủy
            </button>
          </div>
        )}
        <textarea
          aria-label="Viết bình luận"
          className="min-h-24 w-full rounded-lg border p-3 text-sm"
          required
          maxLength={4000}
          value={text}
          placeholder="Chia sẻ suy nghĩ hoặc đặt câu hỏi…"
          onChange={(e) => setText(e.target.value)}
        />
        <Button type="submit" disabled={busy}>
          {busy ? 'Đang gửi…' : 'Gửi bình luận'}
        </Button>
      </form>
    </section>
  )
}
export function PostDetailScreen() {
  const { id } = useParams<{ id: string }>()
  const { data, loading, error, refresh } = useResource<Post>(
    `/api/v1/posts/${id}`
  )
  return (
    <CommunityShell title="Chi tiết bài viết">
      <div className="mx-auto max-w-3xl space-y-6">
        <LoadState
          loading={loading}
          error={error}
          retry={() => void refresh()}
        />
        {data && (
          <>
            <PostCard post={data} refresh={() => void refresh()} />
            {data.status === 'PUBLISHED' && <Comments postId={id} />}
          </>
        )}
      </div>
    </CommunityShell>
  )
}
