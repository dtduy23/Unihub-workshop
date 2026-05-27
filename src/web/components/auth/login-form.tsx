'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Eye, EyeOff, User, Lock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { toast } from 'sonner'
import { api, auth, APIError } from '@/lib/api-client'

/**
 * LoginForm — Đăng nhập qua Go Backend API.
 *
 * Luồng: POST /api/v1/auth/login → nhận { token, user } → lưu cookie → redirect
 * Seed data: student_id = "21127001", password = "123456"
 *
 * Tuân thủ: agent.md mục 10.4 (Auth), ui_web.md mục 3.1
 */

type LoginResponse = {
  token: string
  user: {
    id: string
    studentId: string
    fullName: string
    email: string
    role: string
  }
}

export function LoginForm() {
  const router = useRouter()
  const [showPassword, setShowPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [formData, setFormData] = useState({
    studentId: '',
    password: '',
    rememberMe: false,
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)

    try {
      // Gọi Go Backend — POST /api/v1/auth/login
      // Body dùng snake_case vì gửi trực tiếp lên Go
      const response = await api.post<LoginResponse>('/api/v1/auth/login', {
        student_id: formData.studentId,
        password: formData.password,
      })

      if (!response.data) {
        throw new Error('Phản hồi từ server không hợp lệ')
      }

      const { token, user } = response.data

      // Lưu JWT token và session vào cookie
      auth.setSession(token, {
        id: user.id,
        studentId: user.studentId,
        fullName: user.fullName,
        email: user.email,
        role: user.role,
      })

      toast.success(`Chào mừng ${user.fullName}!`)

      // Chuyển hướng theo Role từ Go Backend
      // Go Backend dùng: ORGANIZER, STAFF, STUDENT
      console.log('User Role from Backend:', user.role)
      const role = user.role?.toUpperCase()
      const isAdmin = role === 'STAFF' || role === 'ADMIN'

      if (isAdmin) {
        router.push('/admin')
      } else {
        router.push('/')
      }
    } catch (error) {
      if (error instanceof APIError) {
        toast.error(error.message || 'Đăng nhập thất bại')
      } else if (error instanceof Error) {
        toast.error(error.message)
      } else {
        toast.error('Đã xảy ra lỗi không xác định')
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value, type, checked } = e.target
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }))
  }

  return (
    <form onSubmit={handleSubmit} className="w-full space-y-6">
      {/* Student ID Field */}
      <div className="space-y-2">
        <label htmlFor="studentId" className="text-sm font-medium text-foreground">
          Mã số / Email Admin
        </label>
        <div className="relative">
          <User className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-muted-foreground" />
          <Input
            id="studentId"
            name="studentId"
            type="text"
            placeholder="Nhập mã số (VD: 21127001 hoặc admin)"
            value={formData.studentId}
            onChange={handleChange}
            className="pl-10 h-11 bg-background border-input"
            required
          />
        </div>
      </div>

      {/* Password Field */}
      <div className="space-y-2">
        <label htmlFor="password" className="text-sm font-medium text-foreground">
          Mật khẩu
        </label>
        <div className="relative">
          <Lock className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-muted-foreground" />
          <Input
            id="password"
            name="password"
            type={showPassword ? 'text' : 'password'}
            placeholder="Nhập mật khẩu"
            value={formData.password}
            onChange={handleChange}
            className="pl-10 pr-10 h-11 bg-background border-input"
            required
          />
          <button
            type="button"
            onClick={() => setShowPassword(!showPassword)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
            aria-label={showPassword ? 'Ẩn mật khẩu' : 'Hiện mật khẩu'}
          >
            {showPassword ? (
              <EyeOff className="h-5 w-5" />
            ) : (
              <Eye className="h-5 w-5" />
            )}
          </button>
        </div>
      </div>

      {/* Remember Me & Forgot Password */}
      <div className="flex items-center justify-between text-sm">
        <label className="flex items-center gap-2 cursor-pointer">
          <Checkbox
            name="rememberMe"
            checked={formData.rememberMe}
            onCheckedChange={(checked) =>
              setFormData(prev => ({
                ...prev,
                rememberMe: checked === true,
              }))
            }
          />
          <span className="text-foreground">Ghi nhớ tôi</span>
        </label>
        <a
          href="#"
          className="text-primary hover:underline font-medium"
        >
          Quên mật khẩu?
        </a>
      </div>

      {/* Submit Button */}
      <Button
        type="submit"
        disabled={isLoading}
        className="w-full h-11 text-base font-semibold rounded-md"
      >
        {isLoading ? 'Đang đăng nhập...' : 'Đăng nhập'}
      </Button>

      {/* Sign Up Link */}
      <p className="text-center text-sm text-muted-foreground">
        Chưa có tài khoản?{' '}
        <a
          href="#"
          className="text-primary hover:underline font-medium"
        >
          Đăng ký ngay
        </a>
      </p>
    </form>
  )
}
