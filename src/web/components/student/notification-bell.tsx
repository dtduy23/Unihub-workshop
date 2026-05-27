"use client"

import { useState, useEffect, useCallback } from "react"
import { Bell, Ticket, QrCode, Clock, MapPin, X, Zap } from "lucide-react"
import { QRCodeSVG } from "qrcode.react"
import { 
  Popover, 
  PopoverContent, 
  PopoverTrigger 
} from "@/components/ui/popover"
import { 
  Dialog, 
  DialogContent, 
  DialogHeader, 
  DialogTitle,
  DialogDescription
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { Badge } from "@/components/ui/badge"
import { api, auth } from "@/lib/api-client"
import { format } from "date-fns"
import { vi } from "date-fns/locale"
import { PaymentDialog } from "./payment-dialog"

interface Registration {
  id: string
  workshopId: string
  status: string
  ticketSignature?: string
  workshopTitle: string
  workshopRoom: string
  startTime: string
}

import { useNotifications, Notification } from "@/hooks/use-notifications"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { toast } from "sonner"

export function NotificationBell() {
  const [registrations, setRegistrations] = useState<Registration[]>([])
  const [loadingRegs, setLoadingRegs] = useState(false)
  const [selectedTicket, setSelectedTicket] = useState<Registration | null>(null)
  const [isQRModalOpen, setIsQRModalOpen] = useState(false)
  const { notifications, loading: loadingNotifs, unreadCount } = useNotifications()
  const [paymentInfo, setPaymentInfo] = useState<{ amount: number; url: string; title: string } | null>(null)
  const [showPaymentDialog, setShowPaymentDialog] = useState(false)

  const fetchRegistrations = useCallback(async () => {
    if (!auth.isAuthenticated()) return
    setLoadingRegs(true)
    try {
      const response = await api.get<Registration[]>('/api/v1/registrations/my')
      if (response.success && response.data) {
        const validItems = response.data.filter(r => 
          (r.status === 'SUCCESS' || r.status === 'PUBLISHED' || r.status === 'PENDING_PAYMENT')
        )
        setRegistrations(validItems)
      }
    } catch (error) {
      console.error("Lỗi khi tải vé:", error)
    } finally {
      setLoadingRegs(false)
    }
  }, [])

  const user = auth.getUser() as any
  const userId = user?.id || 'guest'

  useEffect(() => {
    fetchRegistrations()
    
    // Lắng nghe sự kiện đăng ký thành công để cập nhật dữ liệu ngay lập tức
    const handleSuccess = () => {
      fetchRegistrations()
    }
    window.addEventListener('registration-success', handleSuccess)
    return () => window.removeEventListener('registration-success', handleSuccess)
  }, [fetchRegistrations, userId])

  // Hiển thị toast cho thông báo mới nhất nếu có
  useEffect(() => {
    if (notifications.length > 0) {
      const latest = notifications[0]
      // Chỉ show toast nếu thông báo mới (trong vòng 1 phút)
      const isNew = new Date().getTime() - new Date(latest.createdAt).getTime() < 60000
      if (isNew && latest.status === 'SENT') {
        toast.info(latest.title, {
          description: "Vui lòng kiểm tra hộp thư hoặc danh sách thông báo.",
        })
      }
    }
  }, [notifications])

  const openQR = (reg: Registration) => {
    setSelectedTicket(reg)
    setIsQRModalOpen(true)
  }

  const handlePayResume = async (reg: Registration) => {
    try {
      const res = await api.post<any>(`/api/v1/payments/${reg.id}`)
      
      if (res.success && res.data) {
        setPaymentInfo({
          amount: res.data.payment.amount,
          url: res.data.checkout_url,
          title: reg.workshopTitle
        })
        setShowPaymentDialog(true)
      }
    } catch (error: any) {
      const errorMsg = error.message || ""
      
      if (errorMsg.includes("bảo trì") || errorMsg.includes("maintenance") || errorMsg.includes("outage") || errorMsg.includes("circuit breaker")) {
        toast.error('Cổng thanh toán đang bảo trì', {
          description: 'Hệ thống thanh toán hiện đang bảo trì để nâng cấp. Vui lòng quay lại sau ít phút.'
        })
      } else {
        toast.error(errorMsg || 'Thanh toán thất bại')
      }
    }
  }

  // ... (keep downloadQR and safeFormat logic)
  const downloadQR = () => {
    if (!selectedTicket) return
    const svg = document.querySelector(".qr-container svg") as SVGElement
    if (!svg) return
    const canvas = document.createElement("canvas")
    const svgData = new XMLSerializer().serializeToString(svg)
    const img = new Image()
    const svgBlob = new Blob([svgData], { type: "image/svg+xml;charset=utf-8" })
    const url = URL.createObjectURL(svgBlob)
    img.onload = () => {
      canvas.width = img.width * 2
      canvas.height = img.height * 2
      const ctx = canvas.getContext("2d")
      if (ctx) {
        ctx.fillStyle = "white"
        ctx.fillRect(0, 0, canvas.width, canvas.height)
        ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
        const pngUrl = canvas.toDataURL("image/png")
        const downloadLink = document.createElement("a")
        downloadLink.href = pngUrl
        downloadLink.download = `UniHub-QR-${selectedTicket.workshopTitle.replace(/\s+/g, '-')}.png`
        document.body.appendChild(downloadLink)
        downloadLink.click()
        document.body.removeChild(downloadLink)
      }
      URL.revokeObjectURL(url)
    }
    img.src = url
  }

  const safeFormat = (dateStr: string | undefined, formatStr: string) => {
    if (!dateStr) return "--:--"
    const date = new Date(dateStr)
    if (isNaN(date.getTime()) || date.getFullYear() <= 1) return "--:--"
    return format(date, formatStr, { locale: vi })
  }

  const totalUnread = unreadCount

  return (
    <>
      <Popover onOpenChange={(open) => open && fetchRegistrations()}>
        <PopoverTrigger asChild>
          <Button variant="ghost" size="icon" className="relative hover:bg-slate-100 transition-colors">
            <Bell className="h-5 w-5 text-slate-600" />
            {totalUnread > 0 && (
              <span className="absolute right-2 top-2 h-2.5 w-2.5 rounded-full bg-destructive border-2 border-white animate-pulse" />
            )}
            <span className="sr-only">Thông báo</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" className="w-96 p-0 shadow-2xl border-slate-200 overflow-hidden rounded-xl">
          <Tabs defaultValue="notifs" className="w-full">
            <div className="bg-slate-50/50 p-2 border-b">
              <TabsList className="grid w-full grid-cols-2 bg-slate-100/50">
                <TabsTrigger value="notifs" className="text-xs font-bold gap-2">
                  <Bell className="h-3.5 w-3.5" />
                  Thông báo
                </TabsTrigger>
                <TabsTrigger value="tickets" className="text-xs font-bold gap-2">
                  <Ticket className="h-3.5 w-3.5" />
                  Vé của tôi
                </TabsTrigger>
              </TabsList>
            </div>
            
            <ScrollArea className="h-[400px]">
              <TabsContent value="notifs" className="m-0">
                {loadingNotifs && (
                  <div className="p-8 text-center text-sm text-slate-400">Đang tải thông báo...</div>
                )}
                {!loadingNotifs && notifications.length === 0 && (
                  <div className="p-12 text-center">
                    <Bell className="h-8 w-8 text-slate-200 mx-auto mb-3" />
                    <p className="text-sm text-slate-500">Bạn chưa có thông báo nào.</p>
                  </div>
                )}
                <div className="flex flex-col">
                  {notifications.map((n) => (
                    <div key={n.id} className="p-4 hover:bg-slate-50 transition-colors border-b last:border-0 group">
                      <div className="flex items-start gap-3">
                        <div className="mt-1 h-2 w-2 rounded-full bg-primary shrink-0" />
                        <div className="space-y-1">
                          <p className="text-sm font-bold text-slate-900 leading-tight">{n.title}</p>
                          <p className="text-xs text-slate-500 line-clamp-2">
                            {n.title.includes("xác nhận") || n.title.includes("thành công") 
                              ? "Chúc mừng! Bạn đã đăng ký thành công workshop này. Nhấn để xem chi tiết vé." 
                              : "Bạn có một thông báo mới về workshop."}
                          </p>
                          <p className="text-[10px] text-slate-400 font-medium uppercase tracking-wider mt-2 flex items-center gap-1">
                            <Clock className="h-3 w-3" />
                            {safeFormat(n.createdAt, "HH:mm, dd/MM")}
                          </p>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </TabsContent>

              <TabsContent value="tickets" className="m-0">
                {loadingRegs && (
                  <div className="p-8 text-center text-sm text-slate-400">Đang tải vé...</div>
                )}
                {!loadingRegs && registrations.length === 0 && (
                  <div className="p-12 text-center text-sm text-slate-500">Bạn chưa có vé hoặc đăng ký nào.</div>
                )}
                <div className="flex flex-col">
                  {registrations.map((reg) => (
                    <div key={reg.id} className="p-4 hover:bg-slate-50 transition-colors group border-b last:border-0 relative overflow-hidden">
                      {reg.status === 'PENDING_PAYMENT' && (
                        <div className="absolute top-0 right-0">
                          <Badge className="rounded-none rounded-bl-lg bg-amber-500 text-[8px] font-black uppercase tracking-widest px-2 py-0.5">
                            Chờ thanh toán
                          </Badge>
                        </div>
                      )}
                      
                      <p className="font-bold text-sm text-slate-900 group-hover:text-primary transition-colors pr-10">
                        {reg.workshopTitle}
                      </p>
                      <div className="mt-2 space-y-1 text-xs text-slate-500">
                        <div className="flex items-center gap-2">
                          <Clock className="h-3 w-3 text-primary" />
                          {safeFormat(reg.startTime, "HH:mm, dd/MM")}
                        </div>
                        <div className="flex items-center gap-2">
                          <MapPin className="h-3 w-3 text-primary" />
                          {reg.workshopRoom}
                        </div>
                      </div>

                      {reg.status === 'PENDING_PAYMENT' ? (
                        <Button 
                          variant="default" 
                          size="sm" 
                          className="w-full mt-4 h-8 text-[10px] font-black bg-primary hover:bg-primary/90 text-white transition-all gap-2 tracking-widest shadow-md shadow-primary/20"
                          onClick={() => handlePayResume(reg)}
                        >
                          <Zap className="h-3.5 w-3.5" />
                          THANH TOÁN NGAY
                        </Button>
                      ) : (
                        <Button 
                          variant="outline" 
                          size="sm" 
                          className="w-full mt-4 h-8 text-[10px] font-black border-primary/20 hover:bg-primary hover:text-white transition-all gap-2 tracking-widest"
                          onClick={() => openQR(reg)}
                        >
                          <QrCode className="h-3.5 w-3.5" />
                          XEM MÃ QR
                        </Button>
                      )}
                    </div>
                  ))}
                </div>
              </TabsContent>
            </ScrollArea>
          </Tabs>
          <div className="p-3 bg-slate-50 text-center border-t">
            <button className="text-[10px] font-bold text-slate-400 hover:text-primary uppercase tracking-widest transition-colors">
              Đánh dấu đã đọc tất cả
            </button>
          </div>
        </PopoverContent>
      </Popover>

      {/* QR Code Modal */}
      <Dialog open={isQRModalOpen} onOpenChange={setIsQRModalOpen}>
        <DialogContent className="sm:max-w-[380px] p-0 overflow-hidden border-none shadow-2xl rounded-3xl">
          <div className="bg-primary p-6 text-center relative">
            <div className="mx-auto w-12 h-12 bg-white/10 rounded-xl flex items-center justify-center mb-3 backdrop-blur-sm">
              <QrCode className="h-6 w-6 text-white" />
            </div>
            <DialogTitle className="text-lg font-bold text-white mb-1 leading-tight">
              Mã Check-in Workshop
            </DialogTitle>
            <DialogDescription className="text-white/70 text-xs">
              Đưa mã này cho nhân viên tại cửa.
            </DialogDescription>
          </div>

          <div className="p-6 bg-white text-center">
            <div className="bg-slate-50 p-4 rounded-2xl inline-block shadow-inner border border-slate-100 qr-container">
              {selectedTicket?.ticketSignature && (
                <QRCodeSVG 
                  value={selectedTicket.ticketSignature}
                  size={180}
                  level="H"
                  includeMargin={false}
                />
              )}
            </div>

            <div className="mt-4">
              <Button 
                variant="default" 
                size="default" 
                className="w-full font-bold gap-2 rounded-xl shadow-lg shadow-primary/20 hover:shadow-primary/40 transition-all h-11"
                onClick={downloadQR}
              >
                <QrCode className="h-4 w-4" />
                TẢI VỀ MÁY (PNG)
              </Button>
            </div>

            <div className="mt-6 space-y-1">
              <p className="font-bold text-base text-slate-900 tracking-tight line-clamp-1">
                {selectedTicket?.workshopTitle}
              </p>
              <div className="flex items-center justify-center gap-3 text-xs font-medium text-slate-400">
                <span className="flex items-center gap-1">
                  <Clock className="h-3 w-3 text-primary" />
                  {safeFormat(selectedTicket?.startTime, "HH:mm")}
                </span>
                <span className="w-1 h-1 rounded-full bg-slate-200" />
                <span className="flex items-center gap-1">
                  <MapPin className="h-3 w-3 text-primary" />
                  {selectedTicket?.workshopRoom}
                </span>
              </div>
            </div>

            <p className="mt-6 text-[9px] text-slate-400 uppercase tracking-widest font-medium">
              UniHub Ticket System • ID: {selectedTicket?.id.slice(0, 8)}
            </p>
          </div>
        </DialogContent>
      </Dialog>
      {/* Payment Dialog */}
      <PaymentDialog 
        open={showPaymentDialog}
        onOpenChange={setShowPaymentDialog}
        amount={paymentInfo?.amount || 0}
        paymentUrl={paymentInfo?.url || ""}
        workshopTitle={paymentInfo?.title || ""}
      />
    </>
  )
}
