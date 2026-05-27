"use client"

import { useState, useEffect } from "react"
import { CheckCircle2, Clock, AlertCircle, Copy, ExternalLink, ShieldCheck } from "lucide-react"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Badge } from "@/components/ui/badge"
import { api } from "@/lib/api-client"
import { toast } from "sonner"

interface PaymentDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  amount: number
  paymentUrl: string
  workshopTitle: string
}

export function PaymentDialog({ open, onOpenChange, amount, paymentUrl, workshopTitle }: PaymentDialogProps) {
  const [timeLeft, setTimeLeft] = useState(900) // 15 minutes in seconds
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!open) return
    
    const timer = setInterval(() => {
      setTimeLeft((prev) => (prev > 0 ? prev - 1 : 0))
    }, 1000)

    return () => clearInterval(timer)
  }, [open])

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, "0")}`
  }

  const handleCopy = () => {
    navigator.clipboard.writeText(amount.toString())
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const [isVerifying, setIsVerifying] = useState(false)

  const handleTransferred = () => {
    setIsVerifying(true)
    // Hiển thị thông báo ghi nhận ngay lập tức
    toast.success("Đã ghi nhận thông báo", {
      id: "payment-recorded",
      description: "Hệ thống đang kiểm tra giao dịch với ngân hàng. Vé sẽ được cập nhật tự động sau ít phút."
    })
    
    // Đóng dialog sau 1.5 giây để tạo cảm giác hệ thống đang phản hồi
    setTimeout(() => {
      setIsVerifying(false)
      onOpenChange(false)
    }, 1500)
  }

  const getTxId = () => {
    try {
      const url = new URL(paymentUrl, window.location.origin)
      return url.searchParams.get("tx")?.slice(0, 8).toUpperCase() || "UNIPAY"
    } catch {
      return "UNIPAY"
    }
  }

  const progress = (timeLeft / 900) * 100

  return (
    <Dialog open={open} onOpenChange={(val) => !val && !isVerifying && onOpenChange(false)}>
      <DialogContent className="sm:max-w-[800px] overflow-hidden p-0 rounded-[2rem] border-none shadow-2xl bg-white">
        <div className="bg-primary p-4 text-white text-center shrink-0">
          <DialogTitle className="text-lg font-bold flex items-center justify-center gap-2">
            <ShieldCheck className="h-5 w-5" />
            Xác nhận thanh toán an toàn
          </DialogTitle>
        </div>

        <div className="flex flex-col sm:flex-row">
          {/* Left: QR Code */}
          <div className="sm:w-[55%] p-8 flex flex-col items-center justify-center bg-slate-50/50 border-r border-slate-100">
            <div className="relative group">
              <div className="absolute -inset-6 bg-primary/10 rounded-[2.5rem] opacity-50 blur-2xl"></div>
              <div className="relative bg-white p-4 rounded-3xl shadow-sm border border-slate-200">
                <img 
                  src={`https://img.vietqr.io/image/970422-0000000000-compact.png?amount=${amount}&addInfo=UNIPAY%20${getTxId()}`} 
                  alt="Payment QR" 
                  className="w-72 h-72 object-contain rounded-lg"
                />
              </div>
            </div>
            <p className="mt-8 text-[11px] text-slate-400 uppercase tracking-[0.3em] font-black text-center">
              Quét mã bằng ứng dụng ngân hàng
            </p>
          </div>

          {/* Right: Info & Actions */}
          <div className="sm:w-[45%] p-8 flex flex-col justify-between">
            <div className="space-y-6">
              <div>
                <Badge variant="outline" className="text-primary border-primary/20 bg-primary/5 mb-2 px-3 py-1 text-[10px] font-bold">
                  Nội dung chuyển khoản
                </Badge>
                <div className="flex items-center justify-between p-4 rounded-2xl bg-primary text-white shadow-xl shadow-primary/20 mb-2 group border border-white/10">
                  <span className="text-xl font-mono font-black tracking-widest text-white group-hover:text-indigo-100 transition-colors">
                    {getTxId()}
                  </span>
                  <button 
                    onClick={() => {
                      navigator.clipboard.writeText(getTxId())
                      toast.success("Đã sao chép mã nội dung")
                    }}
                    className="p-2 hover:bg-white/20 rounded-xl transition-colors text-white"
                  >
                    <Copy className="h-4 w-4" />
                  </button>
                </div>
                <h3 className="text-sm font-bold text-slate-500 leading-tight line-clamp-1">
                  {workshopTitle}
                </h3>
              </div>

              <div className="space-y-1">
                <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Số tiền cần chuyển</p>
                <div className="flex items-center gap-2">
                  <span className="text-4xl font-black text-primary tracking-tighter">
                    {(amount || 0).toLocaleString("vi-VN")}
                  </span>
                  <span className="text-xl font-bold text-slate-400 self-end mb-1.5">đ</span>
                  <button 
                    onClick={handleCopy}
                    className="p-2 hover:bg-slate-100 rounded-full transition-colors text-slate-400 mb-1"
                  >
                    {copied ? <CheckCircle2 className="h-5 w-5 text-emerald-500" /> : <Copy className="h-5 w-5" />}
                  </button>
                </div>
              </div>

              <div className="space-y-4">
                <div className="flex items-center justify-between p-4 rounded-2xl bg-emerald-50 border border-emerald-100">
                  <div className="flex items-center gap-2">
                    <Clock className="h-4 w-4 text-emerald-600" />
                    <span className="text-xs font-bold text-emerald-700">Hiệu lực còn lại</span>
                  </div>
                  <span className="text-sm font-mono font-black text-emerald-600">
                    {formatTime(timeLeft)}
                  </span>
                </div>
                
                <div className="flex items-start gap-3 p-4 rounded-2xl bg-blue-50/50 border border-blue-100">
                  <AlertCircle className="h-4 w-4 text-blue-500 shrink-0 mt-0.5" />
                  <p className="text-[10px] text-blue-700 leading-relaxed font-bold italic">
                    Hệ thống sẽ tự động xác nhận vé ngay sau khi nhận được tiền. Vui lòng không đóng cửa sổ này cho đến khi hoàn tất.
                  </p>
                </div>
              </div>
            </div>

            <div className="space-y-3 mt-8">
              <Button 
                className="w-full h-14 rounded-2xl font-black text-lg bg-primary hover:bg-primary/90 text-white shadow-xl shadow-primary/20 transition-all hover:-translate-y-1 active:translate-y-0 disabled:opacity-50 disabled:cursor-not-allowed"
                onClick={handleTransferred}
                disabled={isVerifying}
              >
                {isVerifying ? (
                  <div className="flex items-center gap-2">
                    <div className="h-4 w-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    Đang kiểm tra...
                  </div>
                ) : "Tôi đã chuyển khoản"}
              </Button>
              
            </div>
          </div>
        </div>

        <div className="bg-slate-50 py-2 border-t border-slate-100 text-center">
          <p className="text-[8px] text-slate-400 uppercase tracking-widest font-bold">
            Secure Payment Gateway powered by UniHub
          </p>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function Info({ className }: { className?: string }) {
  return <AlertCircle className={className} />
}
