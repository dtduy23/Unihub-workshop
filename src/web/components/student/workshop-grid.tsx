"use client"

import { WorkshopCard } from "@/components/student/workshop-card"
import { useWorkshops } from "@/hooks/use-workshops"
import { Loader2, SearchX, WifiOff } from "lucide-react"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"

interface WorkshopGridProps {
  searchQuery: string
  categoryFilter: string
  ticketTypeFilter: string
}

export function WorkshopGrid({ 
  searchQuery, 
  categoryFilter, 
  ticketTypeFilter 
}: WorkshopGridProps) {
  const { workshops, isLoading, isOffline } = useWorkshops()

  // Logic lọc dữ liệu (Hoạt động cả khi offline)
  const filteredWorkshops = workshops.filter((workshop) => {
    // 1. Lọc theo search (Tiêu đề hoặc Diễn giả)
    const matchesSearch = 
      workshop.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      workshop.speaker.toLowerCase().includes(searchQuery.toLowerCase())

    // 2. Lọc theo chủ đề
    // Ánh xạ category value sang text hiển thị (hoặc dùng Enum nếu Backend trả về chuẩn)
    const matchesCategory = categoryFilter === "all" || 
      (categoryFilter === "tech" && workshop.category?.toLowerCase().includes("nghệ")) ||
      (categoryFilter === "design" && workshop.category?.toLowerCase().includes("kế")) ||
      (categoryFilter === "business" && workshop.category?.toLowerCase().includes("doanh")) ||
      (categoryFilter === "marketing" && workshop.category?.toLowerCase().includes("marketing")) ||
      (categoryFilter === "softskill" && workshop.category?.toLowerCase().includes("kỹ năng"))

    // 3. Lọc theo loại vé
    const matchesTicket = ticketTypeFilter === "all" || 
      (ticketTypeFilter === "registered" && workshop.isRegistered) ||
      (workshop.ticketType === ticketTypeFilter)

    // 4. Lọc theo trạng thái (Không hiện workshop đã xóa)
    const isNotDeleted = workshop.status !== "DELETED"

    return matchesSearch && matchesCategory && matchesTicket && isNotDeleted
  })

  if (isLoading && workshops.length === 0) {
    return (
      <div className="flex h-64 w-full flex-col items-center justify-center gap-4">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
        <p className="text-muted-foreground">Đang tải danh sách workshop...</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {isOffline && (
        <Alert variant="warning" className="bg-amber-50 border-amber-200">
          <WifiOff className="h-4 w-4 text-amber-600" />
          <AlertTitle className="text-amber-800">Chế độ Offline</AlertTitle>
          <AlertDescription className="text-amber-700">
            Bạn đang xem dữ liệu đã được lưu trong bộ nhớ tạm. Vui lòng kết nối mạng để nhận thông tin mới nhất.
          </AlertDescription>
        </Alert>
      )}

      {filteredWorkshops.length === 0 ? (
        <div className="flex h-64 w-full flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed bg-muted/30">
          <SearchX className="h-12 w-12 text-muted-foreground" />
          <h3 className="text-lg font-medium text-foreground">Không tìm thấy workshop nào</h3>
          <p className="text-sm text-muted-foreground text-center max-w-xs">
            Thử thay đổi từ khóa tìm kiếm hoặc bộ lọc để thấy kết quả khác
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {filteredWorkshops.map((workshop) => (
            <WorkshopCard key={workshop.id} workshop={workshop} />
          ))}
        </div>
      )}
    </div>
  )
}
