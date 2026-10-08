'use client'
import { useSessionUser } from '@/hooks/use-session-user'
import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { KeyRound, LogOut } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { auth } from '@/lib/api-client'
import { NotificationBell } from '@/components/student/notification-bell'
import { ChangePasswordDialog } from '@/components/student/change-password-dialog'

export function AdminHeader() {
  const router = useRouter()
  const [passwordOpen, setPasswordOpen] = useState(false)
  const user = useSessionUser()
  async function logout() {
    await auth.clearSession()
    router.push('/login')
    router.refresh()
  }
  return (
    <header className="sticky top-0 z-30 flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 bg-white px-6 py-3">
      <span className="text-sm font-semibold text-slate-700">
        Quản trị UniHub
      </span>
      <div className="flex items-center gap-2">
        <span className="hidden text-xs text-slate-500 md:block">
          {String(user?.fullName || '')}
        </span>
        <Button asChild variant="outline" size="sm">
          <Link href="/feed">Cộng đồng</Link>
        </Button>
        <NotificationBell />
        <Button
          variant="ghost"
          size="icon"
          aria-label="Đổi mật khẩu"
          onClick={() => setPasswordOpen(true)}
        >
          <KeyRound className="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          aria-label="Đăng xuất"
          onClick={() => void logout()}
        >
          <LogOut className="h-4 w-4" />
        </Button>
      </div>
      <ChangePasswordDialog
        open={passwordOpen}
        onOpenChange={setPasswordOpen}
      />
    </header>
  )
}
