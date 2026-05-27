"use client"

import { useState, useEffect } from "react"
import { Calendar, MapPin, User } from "lucide-react"
import { Card, CardContent, CardFooter, CardHeader } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Button } from "@/components/ui/button"
import { WorkshopDetailDialog } from "./workshop-detail-dialog"
import { PaymentDialog } from "./payment-dialog"
import { useRegistration } from "@/hooks/use-registration"
import { cn } from "@/lib/utils"

export interface Workshop {
  id: string
  title: string
  speaker: string
  speakerTitle?: string
  date: string
  time: string
  location: string
  capacity: number
  availableSeats: number
  ticketType: "free" | "paid"
  price?: number
  category: string
  imageUrl?: string
  summary?: string // Cột summary từ Backend
  roomLayoutUrl?: string
  isRegistered?: boolean
  status: "PUBLISHED" | "CLOSED" | "DELETED"
}

interface WorkshopCardProps {
  workshop: Workshop
}

export function WorkshopCard({ workshop }: WorkshopCardProps) {
  const [isDetailOpen, setIsDetailOpen] = useState(false)
  const [hasJustRegistered, setHasJustRegistered] = useState(false)
  const { 
    isRegistering, 
    regStatus, 
    waitingPosition, 
    handleRegister,
    showPaymentDialog,
    setShowPaymentDialog,
    paymentInfo
  } = useRegistration(workshop.id)

  useEffect(() => {
    // Lắng nghe sự kiện đăng ký thành công cho RIÊNG workshop này
    const handleSuccess = () => {
      setHasJustRegistered(true)
    }
    const eventName = `workshop-reg-success-${workshop.id}`
    window.addEventListener(eventName, handleSuccess)
    return () => window.removeEventListener(eventName, handleSuccess)
  }, [workshop.id])

  const filledPercentage = Math.round(
    ((workshop.capacity - workshop.availableSeats) / workshop.capacity) * 100
  )
  const isAlmostFull = filledPercentage >= 80
  const isFull = workshop.availableSeats === 0
  const isOpen = workshop.status === "PUBLISHED"

  // Cấu hình nút dựa trên trạng thái
  const getButtonConfig = () => {
    if (workshop.status === "DELETED") {
      return {
        label: "Đã hủy",
        variant: "destructive" as const,
        disabled: true
      }
    }
    if (workshop.status === "CLOSED") {
      return {
        label: "Đã đóng đăng ký",
        variant: "outline" as const,
        className: "border-amber-500 text-amber-600 bg-amber-50 hover:bg-amber-50",
        disabled: true
      }
    }
    if (workshop.isRegistered || hasJustRegistered) {
      return {
        label: "Đã đăng ký",
        variant: "secondary" as const,
        disabled: true
      }
    }
    if (isFull) {
      return {
        label: "Đã hết chỗ",
        variant: "outline" as const,
        disabled: true
      }
    }
    if (isRegistering) {
      return {
        label: regStatus || "Đang xử lý...",
        variant: "default" as const,
        disabled: true
      }
    }
    return {
      label: "Đăng ký tham gia",
      variant: "default" as const,
      disabled: false
    }
  }

  const btnConfig = getButtonConfig()

  return (
    <>
      <Card
        className="group flex h-full flex-col overflow-hidden transition-all hover:border-primary/50 hover:shadow-md cursor-pointer"
        onClick={() => setIsDetailOpen(true)}
      >
        <CardHeader className="space-y-3 pb-3">
          <div className="flex items-center justify-between gap-2">
            <Badge variant="outline" className="font-medium text-primary border-primary/20 bg-primary/5">
              {workshop.category}
            </Badge>
            <Badge
              className={
                workshop.ticketType === "free"
                  ? "bg-emerald-500 hover:bg-emerald-600 text-white"
                  : "bg-blue-600 hover:bg-blue-700 text-white"
              }
            >
              {workshop.ticketType === "free"
                ? "Miễn phí"
                : `${workshop.price?.toLocaleString("vi-VN")}đ`}
            </Badge>
          </div>

          <h3 className="line-clamp-2 text-lg font-bold leading-tight text-foreground group-hover:text-primary transition-colors">
            {workshop.title}
          </h3>

          <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
            <div className="flex h-6 w-6 items-center justify-center rounded-full bg-muted">
              <User className="h-3.5 w-3.5" />
            </div>
            <span className="truncate">{workshop.speaker}</span>
          </div>
        </CardHeader>

        <CardContent className="flex flex-col gap-4 pb-4">
          <div className="space-y-2 text-sm text-muted-foreground">
            <div className="flex items-center gap-2">
              <Calendar className="h-4 w-4 shrink-0 text-primary/70" />
              <span>
                {workshop.date} • {workshop.time}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <MapPin className="h-4 w-4 shrink-0 text-primary/70" />
              <span className="truncate">{workshop.location}</span>
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between text-xs font-medium">
              <span className="text-muted-foreground">Sức chứa: {workshop.capacity}</span>
              <span
                className={
                  isFull
                    ? "text-destructive"
                    : isAlmostFull
                      ? "text-amber-500"
                      : "text-emerald-600"
                }
              >
                Còn {workshop.availableSeats} chỗ
              </span>
            </div>
            <Progress
              value={filledPercentage}
              className={`h-2 ${isFull
                  ? "[&>div]:bg-destructive"
                  : isAlmostFull
                    ? "[&>div]:bg-amber-500"
                    : "[&>div]:bg-emerald-500"
                }`}
            />
          </div>
        </CardContent>

        <CardFooter className="mt-auto pt-2">
          <Button
            className={cn("w-full font-semibold shadow-sm transition-all active:scale-95", btnConfig.className)}
            disabled={btnConfig.disabled}
            variant={btnConfig.variant}
            onClick={(e) => {
              if (btnConfig.disabled) return
              e.stopPropagation()
              handleRegister()
            }}
          >
            {btnConfig.label}
          </Button>
        </CardFooter>
      </Card>

      <WorkshopDetailDialog
        workshop={workshop}
        open={isDetailOpen}
        onOpenChange={setIsDetailOpen}
      />

      <PaymentDialog
        open={showPaymentDialog}
        onOpenChange={setShowPaymentDialog}
        amount={paymentInfo?.amount || 0}
        paymentUrl={paymentInfo?.url || ""}
        workshopTitle={workshop.title}
      />
    </>
  )
}
