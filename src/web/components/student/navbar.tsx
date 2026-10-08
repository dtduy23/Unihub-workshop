'use client'
import { useSessionUser } from '@/hooks/use-session-user'
import { useState } from 'react'
import Link from 'next/link'
import { useRouter, usePathname } from 'next/navigation'
import { LogOut, KeyRound } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { auth } from '@/lib/api-client'
import { NotificationBell } from './notification-bell'
import { ChangePasswordDialog } from './change-password-dialog'

export function Navbar() {
  const router = useRouter(),
    path = usePathname()
  const [passwordOpen, setPasswordOpen] = useState(false)
  const user = useSessionUser()
  const role = String(user?.role || '')
  const links = [
    { href: '/', label: 'Workshop' },
    { href: '/feed', label: 'Cộng đồng' },
    { href: '/companies', label: 'Doanh nghiệp' },
    { href: '/saved', label: 'Đã lưu' },
  ]
  if (role === 'BUSINESS')
    links.push({ href: '/business', label: 'Quản lý doanh nghiệp' })
  if (role === 'ADMIN') links.push({ href: '/admin', label: 'Quản trị' })
  if (role === 'STAFF')
    links.push({ href: '/staff/checkin', label: 'Check-in' })
  async function logout() {
    await auth.clearSession()
    router.push('/login')
    router.refresh()
  }
  return (
    <header className="sticky top-0 z-40 border-b bg-white/95 backdrop-blur">
      <div className="mx-auto flex max-w-7xl flex-wrap items-center gap-4 px-4 py-3 sm:px-6">
        <Link href="/feed" className="flex items-center gap-2">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-indigo-600 font-bold text-white">
            U
          </span>
          <span className="text-lg font-bold text-slate-900">UniHub</span>
        </Link>
        <nav className="order-3 flex w-full gap-1 overflow-x-auto sm:order-none sm:w-auto">
          {links.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className={`whitespace-nowrap rounded-lg px-3 py-2 text-sm font-medium ${path === link.href ? 'bg-indigo-50 text-indigo-700' : 'text-slate-500 hover:bg-slate-50'}`}
            >
              {link.label}
            </Link>
          ))}
        </nav>
        <div className="ml-auto flex items-center gap-1">
          <span className="mr-2 hidden max-w-32 truncate text-xs text-slate-500 md:block">
            {String(user?.fullName || '')}
          </span>
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
      </div>
      <ChangePasswordDialog
        open={passwordOpen}
        onOpenChange={setPasswordOpen}
      />
    </header>
  )
}
