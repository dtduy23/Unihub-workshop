"use client"

import { useState } from "react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"
import { api } from "@/lib/api-client"

interface ForgotPasswordDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ForgotPasswordDialog({ open, onOpenChange }: ForgotPasswordDialogProps) {
  const [identifier, setIdentifier] = useState("")
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!identifier.trim()) {
      toast.error("Vui lòng nhập MSSV hoặc Email")
      return
    }

    setIsLoading(true)
    try {
      const res = await api.post<any>("/api/v1/auth/forgot-password", {
        identifier: identifier.trim()
      })
      
      if (res.success) {
        toast.success(res.message || "Yêu cầu đã được xử lý")
        onOpenChange(false)
        setIdentifier("") // Reset
      } else {
        toast.error(res.message || "Đã xảy ra lỗi")
      }
    } catch (error: any) {
      toast.error(error.message || "Không thể xử lý yêu cầu")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Quên mật khẩu?</DialogTitle>
          <DialogDescription>
            Nhập MSSV hoặc Email đã đăng ký. Hệ thống sẽ tạo một mật khẩu mới ngẫu nhiên và gửi vào email của bạn.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 pt-4">
          <div className="space-y-2">
            <Label htmlFor="identifier">MSSV hoặc Email</Label>
            <Input
              id="identifier"
              type="text"
              placeholder="VD: 21127001 hoặc nva@student.hcmus.edu.vn"
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              disabled={isLoading}
              required
            />
          </div>
          <Button type="submit" className="w-full" disabled={isLoading}>
            {isLoading ? "Đang xử lý..." : "Cấp lại mật khẩu mới"}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}
