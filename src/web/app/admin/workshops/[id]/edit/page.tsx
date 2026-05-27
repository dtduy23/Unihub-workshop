"use client"

import { useEffect, useState } from "react"
import { useRouter, useParams } from "next/navigation"
import { PageHeader } from "@/components/admin/page-header"
import { WorkshopForm } from "@/components/admin/workshop-form"
import { api } from "@/lib/api-client"
import { toast } from "sonner"
import { Loader2, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"

export default function EditWorkshopPage() {
  const router = useRouter()
  const params = useParams()
  const workshopId = params.id as string

  const [workshop, setWorkshop] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchWorkshop = async () => {
      try {
        const response = await api.get<any>(`/api/v1/workshops/${workshopId}`)
        if (response.success && response.data) {
          setWorkshop(response.data)
        }
      } catch (err) {
        console.error("Fetch workshop error:", err)
        setError("Không thể tải thông tin workshop.")
      } finally {
        setLoading(false)
      }
    }

    if (workshopId) {
      fetchWorkshop()
    }
  }, [workshopId])

  const handleSubmit = async (values: any) => {
    try {
      const response = await api.put(`/api/v1/workshops/${workshopId}`, values)
      if (response.success) {
        toast.success("Đã cập nhật Workshop thành công!")
        router.push("/admin/workshops")
      }
    } catch (error) {
      console.error("Update workshop error:", error)
      toast.error("Không thể cập nhật Workshop. Vui lòng thử lại.")
    }
  }

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-[80vh] gap-4">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
        <p className="text-muted-foreground font-medium">Đang tải thông tin workshop...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-[80vh] gap-4 p-8">
        <div className="bg-rose-50 text-rose-600 p-4 rounded-2xl flex items-center gap-3 border border-rose-100">
          <AlertCircle className="h-6 w-6" />
          <p className="font-bold">{error}</p>
        </div>
        <Button onClick={() => window.location.reload()} variant="outline">Thử lại</Button>
      </div>
    )
  }

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-8">
      <PageHeader
        title="Chỉnh sửa Workshop"
        description={`Cập nhật thông tin cho: ${workshop?.title}`}
      />

      <div className="mt-8">
        <WorkshopForm 
          initialData={workshop} 
          onSubmit={handleSubmit} 
          isEditing={true} 
        />
      </div>
    </div>
  )
}
