"use client"

import { useState, useEffect } from "react"
import {
  Calendar,
  MapPin,
  User,
  Clock,
  Info,
  Map as MapIcon,
  X
} from "lucide-react"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { type Workshop } from "./workshop-card"
import { useRegistration } from "@/hooks/use-registration"
import { PaymentDialog } from "./payment-dialog"
import { toast } from "sonner"
import { cn } from "@/lib/utils"

interface WorkshopDetailDialogProps {
  workshop: Workshop | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function WorkshopDetailDialog({
  workshop,
  open,
  onOpenChange
}: WorkshopDetailDialogProps) {
  const [showMap, setShowMap] = useState(false)
  const [hasJustRegistered, setHasJustRegistered] = useState(false)
  const { 
    isRegistering, 
    regStatus, 
    waitingPosition, 
    handleRegister,
    showPaymentDialog,
    setShowPaymentDialog,
    paymentInfo
  } = useRegistration(workshop?.id || "")

  useEffect(() => {
    if (!open) {
      setShowMap(false)
    }
    
    // Lắng nghe sự kiện đăng ký thành công cho RIÊNG workshop này
    const handleSuccess = () => {
      setHasJustRegistered(true)
    }
    const eventName = workshop ? `workshop-reg-success-${workshop.id}` : ''
    if (eventName) {
      window.addEventListener(eventName, handleSuccess)
    }
    return () => {
      if (eventName) {
        window.removeEventListener(eventName, handleSuccess)
      }
    }
  }, [open, workshop?.id])

  if (!workshop) return null

  const filledPercentage = Math.round(
    ((workshop.capacity - workshop.availableSeats) / workshop.capacity) * 100
  )
  const isFull = workshop.availableSeats === 0
  const isAlmostFull = filledPercentage >= 80

  // Cấu hình nút tương tự WorkshopCard
  const getButtonConfig = () => {
    if (workshop.status === "DELETED") {
      return { 
        label: "ĐÃ HỦY", 
        variant: "destructive" as const, 
        disabled: true 
      }
    }
    if (workshop.status === "CLOSED") {
      return { 
        label: "ĐÃ ĐÓNG ĐĂNG KÝ", 
        variant: "outline" as const, 
        className: "border-amber-500 text-amber-600 bg-amber-50 hover:bg-amber-50 shadow-none",
        disabled: true 
      }
    }
    if (workshop.isRegistered || hasJustRegistered) {
      return { 
        label: "ĐÃ ĐĂNG KÝ", 
        variant: "secondary" as const, 
        disabled: true 
      }
    }
    if (isFull) {
      return { 
        label: "HẾT CHỖ", 
        variant: "outline" as const, 
        disabled: true 
      }
    }
    if (isRegistering) {
      return { 
        label: regStatus || "ĐANG XỬ LÝ...", 
        variant: "default" as const, 
        disabled: true 
      }
    }
    return { 
      label: "ĐĂNG KÝ NGAY", 
      variant: "default" as const, 
      disabled: false 
    }
  }

  const btnConfig = getButtonConfig()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-7xl w-[95vw] sm:w-[90vw] h-[90vh] sm:h-[85vh] overflow-hidden p-0 rounded-none sm:rounded-3xl border-none shadow-2xl bg-white">
        <div className="sr-only">
          <DialogHeader>
            <DialogTitle>{workshop.title}</DialogTitle>
            <DialogDescription>Chi tiết thông tin về workshop {workshop.title}</DialogDescription>
          </DialogHeader>
        </div>

        <div className="flex flex-col md:flex-row h-full min-h-0 overflow-hidden">
          {/* Left Side: Visual & Quick Info */}
          <div className="w-full md:w-[35%] bg-slate-50 border-r flex flex-col relative overflow-hidden h-full">
            {showMap && workshop.roomLayoutUrl && (
              <div className="absolute inset-0 z-20 bg-white p-8 animate-in fade-in zoom-in duration-300 flex flex-col">
                <div className="flex items-center justify-between mb-6">
                  <h4 className="font-bold text-xl flex items-center gap-2 text-slate-900">
                    <MapIcon className="h-5 w-5 text-primary" />
                    Sơ đồ: {workshop.location}
                  </h4>
                  <Button variant="outline" size="icon" className="rounded-full hover:bg-slate-100" onClick={() => setShowMap(false)}>
                    <X className="h-5 w-5" />
                  </Button>
                </div>
                <div className="flex-1 relative w-full rounded-2xl border-2 border-dashed border-slate-200 bg-slate-50 overflow-hidden flex items-center justify-center">
                  <img
                    src={workshop.roomLayoutUrl}
                    alt="Sơ đồ phòng"
                    className="max-w-full max-h-full object-contain p-6"
                  />
                </div>
              </div>
            )}

            <div className="h-[40%] w-full bg-gradient-to-br from-primary/30 via-primary/10 to-slate-50 flex flex-col items-center justify-center p-10 text-center shrink-0">
              <Badge className="mb-6 bg-primary text-white hover:bg-primary/90 shadow-lg px-6 py-1.5 text-sm">
                {workshop.category}
              </Badge>
              <h2 className="text-3xl font-black text-slate-900 tracking-tight leading-tight">
                {workshop.title}
              </h2>
            </div>

            <div className="flex-1 p-10 space-y-8 overflow-y-auto custom-scrollbar bg-slate-50/50">
              <div className="grid grid-cols-2 gap-6">
                <div className="space-y-2">
                  <p className="text-xs uppercase tracking-widest text-muted-foreground font-extrabold">Ngày tổ chức</p>
                  <p className="text-base font-bold flex items-center gap-2 text-slate-800">
                    <Calendar className="h-4 w-4 text-primary" />
                    {workshop.date}
                  </p>
                </div>
                <div className="space-y-2">
                  <p className="text-xs uppercase tracking-widest text-muted-foreground font-extrabold">Khung giờ</p>
                  <p className="text-base font-bold flex items-center gap-2 text-slate-800">
                    <Clock className="h-4 w-4 text-primary" />
                    {workshop.time}
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <p className="text-xs uppercase tracking-widest text-muted-foreground font-extrabold">Địa điểm</p>
                <div className="flex items-center justify-between p-4 rounded-2xl bg-white border border-slate-200 shadow-sm">
                  <p className="text-base font-bold flex items-center gap-2 text-slate-800">
                    <MapPin className="h-4 w-4 text-primary" />
                    {workshop.location}
                  </p>
                  <Button
                    variant="secondary"
                    size="sm"
                    className="font-bold text-primary border border-primary/20 hover:bg-primary/10 shadow-sm"
                    onClick={() => setShowMap(true)}
                  >
                    <MapIcon className="h-3.5 w-3.5 mr-1.5" />
                    Mở sơ đồ
                  </Button>
                </div>
              </div>

              <div className="pt-4 space-y-4">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm font-bold text-slate-600">Trạng thái đăng ký</span>
                  <span className={cn(
                    "text-sm font-black",
                    isFull ? "text-destructive" : isAlmostFull ? "text-amber-500" : "text-emerald-600"
                  )}>
                    {workshop.capacity - workshop.availableSeats}/{workshop.capacity} ghế
                  </span>
                </div>
                <Progress 
                  value={filledPercentage} 
                  className={cn(
                    "h-3 bg-slate-200",
                    isFull ? "[&>div]:bg-destructive" : isAlmostFull ? "[&>div]:bg-amber-500" : "[&>div]:bg-emerald-500"
                  )} 
                />
              </div>
            </div>
          </div>

          {/* Right Side: Detailed Content & Actions */}
          <div className="flex-1 flex flex-col bg-white overflow-hidden h-full min-h-0">
            <div className="flex-1 p-10 md:p-14 overflow-y-auto custom-scrollbar">
              <section className="space-y-6">
                <div className="flex items-center gap-3 font-black text-2xl text-slate-900">
                  <div className="h-8 w-2 bg-primary rounded-full" />
                  <h3>Nội dung & Chương trình</h3>
                </div>
                
                <div className="space-y-6 text-lg text-slate-600 leading-relaxed whitespace-pre-line font-medium bg-slate-50/30 p-10 rounded-[2rem] border border-slate-100 shadow-inner">
                  {workshop.summary ? (
                    <div>{workshop.summary}</div>
                  ) : (
                    <div className="text-slate-400 italic">Đang cập nhật nội dung chương trình...</div>
                  )}
                </div>
              </section>

              <div className="mt-12 pb-10">
                <section className="flex items-center gap-6 p-8 rounded-3xl bg-slate-50/50 border-2 border-slate-100 group">
                  <div className="flex h-20 w-20 shrink-0 items-center justify-center rounded-2xl bg-white shadow-md text-primary">
                    <User className="h-10 w-10" />
                  </div>
                  <div>
                    <h4 className="text-2xl font-black text-slate-900">{workshop.speaker}</h4>
                    <p className="text-sm font-bold text-primary uppercase tracking-widest">{workshop.speakerTitle || "Expert Speaker"}</p>
                  </div>
                </section>
              </div>
            </div>

            {/* Sticky Actions Footer */}
            <div className="p-10 md:px-14 md:py-10 border-t flex items-center justify-between gap-10 bg-white shrink-0 shadow-[0_-4px_20px_rgba(0,0,0,0.03)]">
              <div className="hidden sm:block">
                <p className="text-sm font-bold text-slate-400 uppercase tracking-[0.2em] mb-2">
                  {workshop.isRegistered ? "Trạng thái" : (regStatus || "Thông tin vé")}
                </p>
                <p className="text-4xl font-black text-slate-900 tracking-tighter">
                  {workshop.isRegistered ? "ĐÃ ĐĂNG KÝ" : waitingPosition ? `#${waitingPosition}` : (workshop.ticketType === "free" ? "MIỄN PHÍ" : `${workshop.price?.toLocaleString('vi-VN')}đ`)}
                </p>
              </div>
              <Button 
                size="lg" 
                className={cn(
                  "flex-1 sm:flex-none sm:min-w-[320px] h-18 text-2xl font-black shadow-2xl shadow-primary/30 hover:shadow-primary/40 transition-all hover:translate-y-[-4px] active:translate-y-[2px] rounded-2xl",
                  btnConfig.className
                )}
                disabled={btnConfig.disabled}
                variant={btnConfig.variant}
                onClick={handleRegister}
              >
                {btnConfig.label}
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
      <PaymentDialog
        open={showPaymentDialog}
        onOpenChange={setShowPaymentDialog}
        amount={paymentInfo?.amount || 0}
        paymentUrl={paymentInfo?.url || ""}
        workshopTitle={workshop.title}
      />
      <style dangerouslySetInnerHTML={{ __html: customScrollbarStyles }} />
    </Dialog>
  )
}

const customScrollbarStyles = `
  .custom-scrollbar::-webkit-scrollbar {
    width: 10px;
  }
  .custom-scrollbar::-webkit-scrollbar-track {
    background: #f1f5f9;
    border-radius: 5px;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb {
    background: #475569;
    border-radius: 5px;
    border: 2px solid #f1f5f9;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb:hover {
    background: #1e293b;
  }
  .custom-scrollbar {
    scrollbar-width: thin;
    scrollbar-color: #475569 #f1f5f9;
  }
`
