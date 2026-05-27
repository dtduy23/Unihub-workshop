import { AdminSidebar } from '@/components/admin/admin-sidebar'

export const metadata = {
  title: 'UniHub Admin - Hệ thống quản lý',
  description: 'Bảng điều khiển dành cho Ban tổ chức UniHub',
}

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex bg-slate-50 min-h-screen">
      {/* Sidebar cố định */}
      <AdminSidebar />

      {/* Vùng nội dung chính */}
      <main className="flex-1 h-screen overflow-y-auto">
        <div className="min-h-full">
          {children}
        </div>
      </main>
    </div>
  )
}
