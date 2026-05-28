import type { Metadata } from 'next'

export const metadata: Metadata = {
  title: 'UniHub Staff - Check-in',
  description: 'Hệ thống điểm danh dành cho nhân viên UniHub',
}

export default function StaffLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="min-h-screen bg-slate-50">
      {children}
    </div>
  )
}
