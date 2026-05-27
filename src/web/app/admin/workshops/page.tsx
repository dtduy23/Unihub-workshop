"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { 
  Dialog, 
  DialogContent, 
  DialogHeader, 
  DialogTitle 
} from "@/components/ui/dialog"
import { QRCodeSVG } from "qrcode.react"
import { QrCode, Loader2, AlertCircle } from "lucide-react"
import { toast } from "sonner"
import { PageHeader } from "@/components/admin/page-header"
import { WorkshopTable } from "@/components/admin/workshop-table"
import { useWorkshops } from "@/hooks/use-workshops"
import { api } from "@/lib/api-client"
import { Button } from "@/components/ui/button"

export default function AdminWorkshopsPage() {
  const router = useRouter()
  const { workshops, loading, error, refresh } = useWorkshops()
  
  const [selectedWorkshop, setSelectedWorkshop] = useState<any>(null)
  const [registrations, setRegistrations] = useState<any[]>([])
  const [loadingRegs, setLoadingRegs] = useState(false)
  const [isQRModalOpen, setIsQRModalOpen] = useState(false)

  // Đăng ký hàm xử lý Xem QR vào window để WorkshopTable có thể gọi
  useEffect(() => {
    (window as any).onViewQR = (ws: any) => {
      setSelectedWorkshop(ws)
      setIsQRModalOpen(true)
      fetchRegistrations(ws.id)
    }
    return () => { delete (window as any).onViewQR }
  }, [])

  const fetchRegistrations = async (workshopId: string) => {
    setLoadingRegs(true)
    try {
      const response = await api.get<any[]>(`/api/v1/registrations/workshop/${workshopId}`)
      if (response.success && response.data) {
        // Chỉ lấy những vé đã thanh toán thành công và có chữ ký
        const validTickets = response.data.filter(r => r.status === 'SUCCESS' && r.ticketSignature)
        setRegistrations(validTickets)
      }
    } catch (error) {
      console.error("Lỗi tải danh sách đăng ký:", error)
      toast.error("Không thể tải danh sách sinh viên đăng ký")
    } finally {
      setLoadingRegs(false)
    }
  }

  const handleCreate = () => {
    router.push("/admin/workshops/create")
  }

  const handleEdit = (ws: any) => {
    router.push(`/admin/workshops/${ws.id}/edit`)
  }

  const handleDelete = async (ws: any) => {
    if (!confirm(`Bạn có chắc chắn muốn xóa workshop "${ws.title}"?`)) return
    
    try {
      const response = await api.delete(`/api/v1/workshops/${ws.id}`)
      if (response.success) {
        toast.success("Đã xóa workshop thành công")
        refresh()
      }
    } catch (error) {
      toast.error("Không thể xóa workshop")
    }
  }

  if (loading && workshops.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-[80vh] gap-4">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
        <p className="text-muted-foreground font-medium">Đang tải danh sách workshop...</p>
      </div>
    )
  }

  return (
    <div className="p-8 space-y-8 max-w-7xl mx-auto">
      <PageHeader
        title="Quản lý Workshop"
        description="Theo dõi danh sách, chỉnh sửa nội dung và quản lý mã QR điểm danh cho sinh viên."
        actionLabel="Tạo Workshop mới"
        onAction={handleCreate}
      />

      {error ? (
        <div className="flex flex-col items-center justify-center py-20 gap-4">
          <div className="bg-rose-50 text-rose-600 p-4 rounded-2xl flex items-center gap-3 border border-rose-100">
            <AlertCircle className="h-6 w-6" />
            <p className="font-bold">{error}</p>
          </div>
          <Button onClick={() => refresh()} variant="outline">Thử lại</Button>
        </div>
      ) : (
        <WorkshopTable 
          workshops={workshops} 
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
      )}

      {/* Modal xem mã QR Sinh viên */}
      <Dialog open={isQRModalOpen} onOpenChange={setIsQRModalOpen}>
        <DialogContent className="!max-w-[95vw] lg:!max-w-[1000px] max-h-[90vh] overflow-y-auto rounded-3xl">
          <DialogHeader>
            <DialogTitle className="text-2xl font-bold text-slate-900">
              Mã QR Sinh viên - {selectedWorkshop?.title}
            </DialogTitle>
          </DialogHeader>

          {loadingRegs ? (
            <div className="flex flex-col items-center justify-center py-20 gap-4">
              <Loader2 className="h-10 w-10 animate-spin text-primary" />
              <p className="text-slate-500 font-medium">Đang chuẩn bị mã QR...</p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 p-2">
              {registrations.length === 0 ? (
                <div className="col-span-full text-center py-20">
                  <div className="mx-auto w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mb-4">
                    <QrCode className="h-8 w-8 text-slate-300" />
                  </div>
                  <p className="text-slate-500 font-medium">Chưa có sinh viên nào đăng ký thành công hoặc chưa có chữ ký số.</p>
                </div>
              ) : (
                registrations.map((reg) => (
                  <div key={reg.id} className="flex flex-col items-center p-6 border border-slate-100 rounded-3xl bg-white shadow-sm hover:shadow-md transition-shadow">
                    <div className="bg-slate-50 p-4 rounded-2xl mb-4 border border-slate-100">
                      <QRCodeSVG
                        value={JSON.stringify({
                          sid: reg.studentId, // Sử dụng dữ liệu đã map camelCase
                          uid: reg.userId,
                          wid: selectedWorkshop?.id,
                          sig: reg.ticketSignature
                        })}
                        size={180}
                        level="H"
                        includeMargin={true}
                      />
                    </div>
                    <div className="text-center space-y-2">
                      <div className="font-bold text-slate-900 text-lg leading-tight">{reg.fullName || 'Sinh viên'}</div>
                      <div className="text-sm font-mono text-indigo-600 bg-indigo-50 px-3 py-1 rounded-full inline-block font-bold">
                        {reg.studentId}
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
