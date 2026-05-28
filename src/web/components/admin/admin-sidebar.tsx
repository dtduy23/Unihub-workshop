'use client'

import Link from 'next/link'
import { useRouter, usePathname } from 'next/navigation'
import { 
  LayoutDashboard, 
  CalendarRange, 
  Database, 
  Settings, 
  ChevronRight,
  Sparkles,
  ScanLine
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'

const navItems = [
  { 
    name: 'Dashboard', 
    href: '/admin', 
    icon: LayoutDashboard 
  },
  { 
    name: 'Quản lý Workshop', 
    href: '/admin/workshops', 
    icon: CalendarRange 
  },
  { 
    name: 'Check-in', 
    href: '/staff/checkin', 
    icon: ScanLine 
  },
  { 
    name: 'Trung tâm Dữ liệu', 
    href: '/admin/data-sync', 
    icon: Database 
  },
  { 
    name: 'Cài đặt', 
    href: '/admin/settings', 
    icon: Settings 
  },
]

export function AdminSidebar() {
  const pathname = usePathname()
  const router = useRouter()

  const handleLogout = () => {
    // 1. Xóa session cookie
    document.cookie = "unihub_session=; path=/; max-age=0; SameSite=Lax"
    
    // 2. Thông báo và điều hướng mượt mà
    toast.success('Đăng xuất thành công')
    router.push('/login')
  }

  return (
    <aside className="w-72 h-screen bg-white border-r border-slate-200 flex flex-col sticky top-0">
      {/* Logo Section */}
      <div className="p-8">
        <Link href="/admin" className="flex items-center gap-3">
          <div className="w-10 h-10 bg-primary rounded-xl flex items-center justify-center shadow-lg shadow-indigo-500/20">
            <Sparkles className="text-white h-6 w-6" />
          </div>
          <div>
            <span className="text-xl font-bold tracking-tight text-slate-900">UniHub</span>
            <span className="block text-[10px] font-bold text-primary tracking-widest uppercase">Admin Portal</span>
          </div>
        </Link>
      </div>

      {/* Navigation Links */}
      <nav className="flex-1 px-4 space-y-1">
        {navItems.map((item) => {
          const isActive = pathname === item.href
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center justify-between px-4 py-3.5 rounded-xl transition-all duration-200 group",
                isActive 
                  ? "bg-primary text-white shadow-md shadow-indigo-500/20" 
                  : "text-slate-500 hover:bg-indigo-50 hover:text-primary"
              )}
            >
              <div className="flex items-center gap-3">
                <item.icon className={cn("h-5 w-5", isActive ? "text-white" : "text-slate-400 group-hover:text-primary transition-colors")} />
                <span className="font-semibold text-sm">{item.name}</span>
              </div>
              {isActive && <ChevronRight className="h-4 w-4" />}
            </Link>
          )
        })}
      </nav>

      {/* Footer / Logout */}
      <div className="p-4 border-t border-slate-100">
        <button 
          onClick={handleLogout}
          className="flex items-center justify-center w-full px-4 py-3 rounded-xl text-slate-500 hover:bg-red-50 hover:text-red-600 transition-all duration-200 font-bold text-sm"
        >
          Đăng xuất
        </button>
      </div>
    </aside>
  )
}
