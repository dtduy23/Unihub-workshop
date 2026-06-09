"use client"

import { useState } from "react"
import { Pencil, Trash2, ChevronLeft, ChevronRight, MoreHorizontal, QrCode, DownloadCloud } from "lucide-react"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { api } from "@/lib/api-client"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { cn } from "@/lib/utils"

export interface Workshop {
  id: string
  title: string
  speaker: string
  date: string
  time: string
  capacity: number
  registered: number
  price: number
  status: "PUBLISHED" | "CLOSED" | "DELETED"
}

interface WorkshopTableProps {
  workshops: Workshop[]
  onEdit?: (workshop: Workshop) => void
  onDelete?: (workshop: Workshop) => void
  showActions?: boolean
}

const statusConfig = {
  PUBLISHED: {
    label: "Đang mở",
    className: "bg-emerald-50 text-emerald-700 border-emerald-200",
  },
  CLOSED: {
    label: "Đã đóng",
    className: "bg-amber-50 text-amber-700 border-amber-200",
  },
  DELETED: {
    label: "Đã xóa",
    className: "bg-rose-50 text-rose-700 border-rose-200",
  },
}

export function WorkshopTable({ workshops, onEdit, onDelete, showActions = true }: WorkshopTableProps) {
  const [currentPage, setCurrentPage] = useState(1)
  const itemsPerPage = 10
  const totalPages = Math.ceil(workshops.length / itemsPerPage)

  const paginatedWorkshops = workshops.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage
  )

  const formatPrice = (price: number) => {
    if (price === 0) return "Miễn phí"
    return new Intl.NumberFormat("vi-VN", {
      style: "currency",
      currency: "VND",
    }).format(price)
  }

  const formatDate = (datetime: string) => {
    if (!datetime) return "Chưa xác định"
    const date = new Date(datetime)
    if (isNaN(date.getTime())) return "Ngày không hợp lệ"
    
    return new Intl.DateTimeFormat("vi-VN", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }).format(date)
  }

  const handleDownloadCSV = async (workshop: Workshop, type: 'registered' | 'attended') => {
    try {
      toast.info("Đang chuẩn bị file CSV...")
      const filename = `workshop_${workshop.id}_${type}.csv`
      await api.download(`/api/v1/registrations/workshop/${workshop.id}/export?type=${type}`, filename)
      toast.success("Tải file thành công")
    } catch (err) {
      toast.error("Lỗi khi tải file CSV")
    }
  }

  return (
    <div className="rounded-2xl border border-slate-200 bg-white overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent bg-slate-50/50">
            <TableHead className="w-[300px] text-sm font-semibold text-slate-700 py-4">
              Tiêu đề
            </TableHead>
            <TableHead className="text-sm font-semibold text-slate-700">
              Diễn giả
            </TableHead>
            <TableHead className="text-sm font-semibold text-slate-700">
              Thời gian
            </TableHead>
            <TableHead className="text-sm font-semibold text-slate-700">
              Số chỗ
            </TableHead>
            <TableHead className="text-sm font-semibold text-slate-700">
              Giá vé
            </TableHead>
            <TableHead className="text-sm font-semibold text-slate-700">
              Trạng thái
            </TableHead>
            {showActions && (
              <TableHead className="w-[100px] text-right text-sm font-semibold text-slate-700">
                Hành động
              </TableHead>
            )}
          </TableRow>
        </TableHeader>
        <TableBody>
          {paginatedWorkshops.length === 0 ? (
            <TableRow>
              <TableCell colSpan={showActions ? 7 : 6} className="h-32 text-center text-muted-foreground">
                Không có dữ liệu hiển thị.
              </TableCell>
            </TableRow>
          ) : (
            paginatedWorkshops.map((workshop) => {
              const status = statusConfig[workshop.status as keyof typeof statusConfig] || statusConfig.PUBLISHED
              const fillPercentage = (workshop.registered / workshop.capacity) * 100

              return (
                <TableRow key={workshop.id} className="group hover:bg-slate-50/50 transition-colors">
                  <TableCell className="font-bold text-slate-900">
                    {workshop.title}
                  </TableCell>
                  <TableCell className="text-slate-600 font-medium">{workshop.speaker}</TableCell>
                  <TableCell className="text-sm text-slate-600 font-medium">
                    {workshop.date} <br />
                    <span className="text-xs text-slate-400">({workshop.time})</span>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1.5 min-w-[120px]">
                      <div className="flex items-center justify-between text-[11px] font-bold">
                        <span className="text-slate-500">
                          {workshop.registered}/{workshop.capacity}
                        </span>
                        <span className={cn(
                          workshop.registered >= workshop.capacity ? "text-rose-600" : 
                          (workshop.registered / workshop.capacity) >= 0.8 ? "text-amber-600" : "text-emerald-600"
                        )}>
                          {Math.round((workshop.registered / workshop.capacity) * 100)}%
                        </span>
                      </div>
                      <div className="h-1.5 w-full bg-slate-100 rounded-full overflow-hidden">
                        <div 
                          className={cn(
                            "h-full transition-all duration-500",
                            workshop.registered >= workshop.capacity ? "bg-rose-500" : 
                            (workshop.registered / workshop.capacity) >= 0.8 ? "bg-amber-500" : "bg-emerald-500"
                          )}
                          style={{ width: `${Math.min(100, (workshop.registered / workshop.capacity) * 100)}%` }}
                        />
                      </div>
                    </div>
                  </TableCell>
                  <TableCell className="text-slate-600 font-medium text-sm">
                    {formatPrice(workshop.price)}
                  </TableCell>
                  <TableCell>
                    <span
                      className={cn(
                        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wider",
                        status.className
                      )}
                    >
                      {status.label}
                    </span>
                  </TableCell>
                  {showActions && (
                    <TableCell className="text-right">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity"
                          >
                            <MoreHorizontal className="h-4 w-4 text-slate-500" />
                            <span className="sr-only">Mở menu</span>
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="rounded-xl shadow-xl border-slate-200">
                          <DropdownMenuItem
                            onClick={() => onEdit?.(workshop)}
                            className="cursor-pointer font-medium"
                          >
                            <Pencil className="mr-2 h-4 w-4" />
                            Chỉnh sửa
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => (window as any).onViewQR?.(workshop)}
                            className="cursor-pointer font-medium"
                          >
                            <QrCode className="mr-2 h-4 w-4" />
                            Xem mã QR
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => handleDownloadCSV(workshop, 'registered')}
                            className="cursor-pointer font-medium"
                          >
                            <DownloadCloud className="mr-2 h-4 w-4" />
                            Xuất DS Đăng ký
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => handleDownloadCSV(workshop, 'attended')}
                            className="cursor-pointer font-medium"
                          >
                            <DownloadCloud className="mr-2 h-4 w-4" />
                            Xuất DS Điểm danh
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => onDelete?.(workshop)}
                            className="cursor-pointer text-rose-600 focus:text-rose-600 font-medium"
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Xóa
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  )}
                </TableRow>
              )
            })
          )}
        </TableBody>
      </Table>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between border-t border-slate-100 px-6 py-4 bg-slate-50/30">
          <p className="text-xs font-medium text-slate-500 italic">
            Hiển thị {(currentPage - 1) * itemsPerPage + 1} -{" "}
            {Math.min(currentPage * itemsPerPage, workshops.length)} / {workshops.length}
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              disabled={currentPage === 1}
              className="h-8 rounded-lg"
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            {Array.from({ length: totalPages }, (_, i) => i + 1).map((page) => (
              <Button
                key={page}
                variant={currentPage === page ? "default" : "outline"}
                size="sm"
                onClick={() => setCurrentPage(page)}
                className={cn(
                  "h-8 w-8 rounded-lg font-bold text-xs",
                  currentPage === page ? "bg-primary hover:bg-primary/90 shadow-md shadow-primary/20" : "text-slate-600"
                )}
              >
                {page}
              </Button>
            ))}
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
              disabled={currentPage === totalPages}
              className="h-8 rounded-lg"
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
