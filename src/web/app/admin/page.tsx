"use client"

import { Plus, Search, Filter } from "lucide-react"
import { useRouter } from "next/navigation"
import { AdminLayout } from "@/components/admin/admin-layout"
import { WorkshopTable, Workshop } from "@/components/admin/workshop-table"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"


import { useWorkshops } from "@/hooks/use-workshops"
import { Loader2, AlertCircle } from "lucide-react"
import { PageHeader } from "@/components/admin/page-header"

export default function DashboardPage() {

  const router = useRouter()
  const { workshops, loading, error } = useWorkshops()

  const handleCreate = () => {
    router.push("/admin/workshops/create")
  }

  const handleEdit = (w: any) => {
    router.push(`/admin/workshops/${w.id}/edit`)
  }

  // Sắp xếp Workshop theo trạng thái: PUBLISHED > CLOSED > DELETED
  const sortedWorkshops = [...workshops].sort((a, b) => {
    const order: Record<string, number> = { PUBLISHED: 0, CLOSED: 1, DELETED: 2 }
    return (order[a.status] ?? 3) - (order[b.status] ?? 3)
  })

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-[80vh] gap-4">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
        <p className="text-muted-foreground font-medium">Đang tải dữ liệu hệ thống...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-[80vh] gap-4 p-8">
        <div className="bg-rose-50 text-rose-600 p-4 rounded-2xl flex items-center gap-3 border border-rose-100">
          <AlertCircle className="h-6 w-6" />
          <p className="font-bold">Lỗi kết nối: {error}</p>
        </div>
        <Button onClick={() => window.location.reload()} variant="outline">Thử lại</Button>
      </div>
    )
  }

  return (
    <div className="p-8 space-y-8 max-w-7xl mx-auto">
      <PageHeader 
        title="Tổng quan hệ thống"
        description="Thống kê hoạt động và quản lý các buổi Workshop UniHub"
      />

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm hover:shadow-md transition-shadow">
          <p className="text-sm font-medium text-slate-500 uppercase tracking-wider">Tổng Workshop</p>
          <p className="mt-2 text-4xl font-bold text-slate-900 leading-tight">
            {workshops.length}
          </p>
        </div>
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm hover:shadow-md transition-shadow">
          <p className="text-sm font-medium text-slate-500 uppercase tracking-wider">Đang mở đăng ký</p>
          <p className="mt-2 text-4xl font-bold text-emerald-600 leading-tight">
            {workshops.filter((w) => w.status === "PUBLISHED").length}
          </p>
        </div>
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm hover:shadow-md transition-shadow">
          <p className="text-sm font-medium text-slate-500 uppercase tracking-wider">Đã đóng / Kết thúc</p>
          <p className="mt-2 text-4xl font-bold text-amber-600 leading-tight">
            {workshops.filter((w) => w.status === "CLOSED").length}
          </p>
        </div>
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm hover:shadow-md transition-shadow">
          <p className="text-sm font-medium text-slate-500 uppercase tracking-wider">Tổng vé thành công</p>
          <p className="mt-2 text-4xl font-bold text-primary leading-tight">
            {workshops.reduce((sum, w) => sum + w.registered, 0)}
          </p>
        </div>
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm hover:shadow-md transition-shadow">
          <p className="text-sm font-medium text-slate-500 uppercase tracking-wider">
            Tỷ lệ lấp đầy
          </p>
          <p className="mt-2 text-4xl font-bold text-amber-600 leading-tight">
            {workshops.length > 0 
              ? Math.round(
                  (workshops.reduce((sum, w) => sum + w.registered, 0) /
                    workshops.reduce((sum, w) => sum + w.capacity, 0)) *
                    100
                )
              : 0}
            %
          </p>
        </div>
      </div>

      {/* Workshop Table Section */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-bold text-slate-900">Hoạt động gần đây</h2>
          <Button variant="ghost" className="text-primary hover:text-primary/80 hover:bg-indigo-50 font-bold" asChild>
            <a href="/admin/workshops">Xem chi tiết</a>
          </Button>
        </div>
        
        <WorkshopTable
          workshops={sortedWorkshops.slice(0, 10)}
          showActions={false}
        />
      </div>
    </div>
  )
}
