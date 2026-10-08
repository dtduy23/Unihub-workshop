'use client'
import Link from 'next/link'
import { useState } from 'react'
import { api } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
export default function Page() {
  const [password, setPassword] = useState(''),
    [confirmation, setConfirmation] = useState(''),
    [busy, setBusy] = useState(false),
    [done, setDone] = useState(false)
  async function reset(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (password !== confirmation) {
      toast.error('Hai mật khẩu không khớp')
      return
    }
    setBusy(true)
    try {
      const token = new URLSearchParams(window.location.search).get('token')
      await api.post('/api/v1/auth/reset-password', { token, password })
      setDone(true)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Liên kết không hợp lệ')
    } finally {
      setBusy(false)
    }
  }
  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-50 p-4">
      <section className="w-full max-w-md rounded-2xl border bg-white p-8 shadow-sm">
        <h1 className="text-2xl font-bold">Đặt lại mật khẩu</h1>
        {done ? (
          <>
            <p className="my-5">
              Mật khẩu đã cập nhật. Các phiên cũ đã hết hiệu lực.
            </p>
            <Button asChild>
              <Link href="/login">Đăng nhập</Link>
            </Button>
          </>
        ) : (
          <form onSubmit={reset} className="mt-6 space-y-4">
            <p className="text-sm text-slate-500">
              Liên kết có hiệu lực trong 30 phút và chỉ dùng một lần.
            </p>
            <label className="block text-sm">
              Mật khẩu mới
              <input
                aria-label="Mật khẩu mới"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                maxLength={72}
                className="mt-2 w-full rounded-lg border p-3"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </label>
            <label className="block text-sm">
              Nhập lại mật khẩu
              <input
                aria-label="Nhập lại mật khẩu"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                maxLength={72}
                className="mt-2 w-full rounded-lg border p-3"
                value={confirmation}
                onChange={(e) => setConfirmation(e.target.value)}
              />
            </label>
            <Button className="w-full" disabled={busy}>
              {busy ? 'Đang xử lý…' : 'Cập nhật mật khẩu'}
            </Button>
          </form>
        )}
      </section>
    </main>
  )
}
