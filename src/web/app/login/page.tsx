import { LoginForm } from '@/components/auth/login-form'
import Image from 'next/image'

export const metadata = {
  title: 'Đăng nhập - UniHub Workshop',
  description: 'Đăng nhập vào hệ thống quản lý workshop UniHub',
}

export default function LoginPage() {
  return (
    <div className="min-h-screen bg-slate-50 flex items-center justify-center px-4 py-12">
      <div className="w-full max-w-md">
        {/* Card Container */}
        <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-8">
          {/* Logo & Branding */}
          <div className="flex flex-col items-center gap-4 mb-8">
            <div className="flex items-center justify-center h-12 w-12 rounded-lg bg-gradient-to-br from-indigo-600 to-indigo-700">
              <span className="text-white font-bold text-lg">U</span>
            </div>
            <div className="text-center">
              <h1 className="text-2xl font-bold text-slate-900">UniHub</h1>
              <p className="text-sm text-slate-500 mt-1">Quản lý Workshop</p>
            </div>
          </div>

          {/* Form Title */}
          <div className="mb-8">
            <h2 className="text-xl font-semibold text-slate-900">Đăng nhập</h2>
            <p className="text-sm text-slate-600 mt-2">
              Nhập thông tin của bạn để truy cập hệ thống
            </p>
          </div>

          {/* Login Form */}
          <LoginForm />

        </div>

        {/* Footer Info */}
        <p className="text-center text-xs text-slate-500 mt-6">
          © 2026 UniHub. Tất cả quyền được bảo vệ.
        </p>
      </div>
    </div>
  )
}
