"use client"

import { useRouter } from "next/navigation"
import { AdminLayout } from "@/components/admin/admin-layout"
import { PageHeader } from "@/components/admin/page-header"
import { WorkshopForm } from "@/components/admin/workshop-form"
import { api } from "@/lib/api-client"
import { toast } from "sonner"

export default function CreateWorkshopPage() {
  const router = useRouter()

  const handleSubmit = async (values: any) => {
    try {
      const response = await api.post("/api/v1/workshops", values)
      if (response.success) {
        toast.success("Đã tạo Workshop thành công!")
        router.push("/admin/workshops")
      }
    } catch (error) {
      console.error("Create workshop error:", error)
      toast.error("Không thể tạo Workshop. Vui lòng kiểm tra lại thông tin.")
    }
  }

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-8">
      <PageHeader
        title="Tạo Workshop mới"
        description="Điền thông tin chi tiết để bắt đầu mở đăng ký cho sinh viên."
      />

      <div className="mt-8">
        <WorkshopForm onSubmit={handleSubmit} />
      </div>
    </div>
  )
}
