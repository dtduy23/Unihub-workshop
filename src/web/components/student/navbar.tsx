"use client"

import { useState } from "react"
import { Bell, LogOut, User, KeyRound } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { useRouter } from "next/navigation"
import { toast } from "sonner"
import { NotificationBell } from "./notification-bell"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ChangePasswordDialog } from "./change-password-dialog"

export function Navbar() {
  const router = useRouter()
  const [showChangePassword, setShowChangePassword] = useState(false)

  const handleLogout = () => {
    // 1. Xóa session cookie
    document.cookie = "unihub_session=; path=/; max-age=0; SameSite=Lax"
    
    // 2. Thông báo và điều hướng
    toast.success('Đã đăng xuất thành công')
    router.push('/login')
  }

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-card">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        {/* Logo */}
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary">
            <span className="text-sm font-bold text-primary-foreground">U</span>
          </div>
          <span className="text-xl font-semibold text-foreground">UniHub</span>
        </div>

        {/* Right Actions */}
        <div className="flex items-center gap-2">
          <NotificationBell />

          {/* User Avatar Dropdown */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="relative h-9 w-9 rounded-full bg-slate-100 hover:bg-slate-200 transition-colors">
                <User className="h-5 w-5 text-slate-600" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <div className="flex items-center gap-2 p-2">
                <Avatar className="h-8 w-8">
                  <AvatarFallback className="bg-primary/10 text-primary text-xs">
                    SV
                  </AvatarFallback>
                </Avatar>
                <div className="flex flex-col">
                  <span className="text-sm font-medium">Nguyễn Văn A</span>
                  <span className="text-xs text-muted-foreground">SV123456</span>
                </div>
              </div>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => setShowChangePassword(true)}>
                <KeyRound className="mr-2 h-4 w-4" />
                <span>Đổi mật khẩu</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem 
                className="text-destructive focus:bg-destructive/10 focus:text-destructive"
                onClick={handleLogout}
              >
                <LogOut className="mr-2 h-4 w-4" />
                <span>Đăng xuất</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      <ChangePasswordDialog open={showChangePassword} onOpenChange={setShowChangePassword} />
    </header>
  )
}
