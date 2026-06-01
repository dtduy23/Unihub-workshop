"use client"

import { useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import * as z from "zod"
import { CalendarIcon, Loader2, Sparkles, Upload } from "lucide-react"
import { format } from "date-fns"
import { vi } from "date-fns/locale"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "@/components/ui/select"
import { toast } from "sonner"
import { api } from "@/lib/api-client"

const workshopSchema = z.object({
  title: z.string().min(2, "Tiêu đề phải ít nhất 2 ký tự"),
  speaker: z.string().min(2, "Vui lòng nhập tên diễn giả"),
  room: z.string().min(1, "Vui lòng chọn phòng học"),
  startTime: z.date({
    required_error: "Vui lòng chọn thời gian bắt đầu",
  }),
  endTime: z.date({
    required_error: "Vui lòng chọn thời gian kết thúc",
  }),
  registrationStartTime: z.date({
    required_error: "Vui lòng chọn thời gian bắt đầu đăng ký",
  }),
  registrationEndTime: z.date({
    required_error: "Vui lòng chọn thời gian kết thúc đăng ký",
  }),
  capacity: z.coerce.number().min(1, "Sức chứa phải lớn hơn 0"),
  price: z.coerce.number().min(0, "Giá vé không được âm"),
  summary: z.string().min(5, "Bản tóm tắt phải ít nhất 5 ký tự"),
  status: z.enum(["PUBLISHED", "CLOSED", "DELETED"]).default("PUBLISHED"),
  roomLayoutUrl: z.string().optional(),
}).refine((data) => data.startTime < data.endTime, {
  message: "Thời gian kết thúc workshop phải sau thời gian bắt đầu",
  path: ["endTime"],
}).refine((data) => (data.endTime.getTime() - data.startTime.getTime()) <= 24 * 60 * 60 * 1000, {
  message: "Thời lượng workshop không được vượt quá 24 giờ",
  path: ["endTime"],
}).refine((data) => data.registrationStartTime < data.registrationEndTime, {
  message: "Thời gian đóng đăng ký phải sau thời gian mở",
  path: ["registrationEndTime"],
}).refine((data) => data.registrationEndTime <= data.startTime, {
  message: "Phải đóng đăng ký trước khi workshop bắt đầu",
  path: ["registrationEndTime"],
})

type WorkshopFormValues = z.infer<typeof workshopSchema>

interface WorkshopFormProps {
  initialData?: any
  onSubmit: (data: WorkshopFormValues) => Promise<void>
  isEditing?: boolean
}

export function WorkshopForm({ initialData, onSubmit, isEditing = false }: WorkshopFormProps) {
  const [isAiProcessing, setIsAiProcessing] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const form = useForm<WorkshopFormValues>({
    resolver: zodResolver(workshopSchema),
    defaultValues: initialData ? {
      ...initialData,
      registrationStartTime: initialData.registrationStartTime ? new Date(initialData.registrationStartTime) : new Date(),
      registrationEndTime: initialData.registrationEndTime ? new Date(initialData.registrationEndTime) : new Date(),
      startTime: initialData.startTime ? new Date(initialData.startTime) : new Date(),
      endTime: initialData.endTime ? new Date(initialData.endTime) : new Date(),
    } : {
      title: "",
      speaker: "",
      room: "Hội trường 1",
      capacity: 50,
      price: 0,
      summary: "",
      status: "PUBLISHED",
      roomLayoutUrl: "/maps/grand-hall.jpg",
      registrationStartTime: new Date(),
      registrationEndTime: new Date(new Date().getTime() + 7 * 24 * 60 * 60 * 1000), // Default 1 week
    },
  })

  // Ánh xạ phòng và sơ đồ
  const ROOMS = [
    { name: "Hội trường 1", layout: "/maps/grand-hall.jpg" },
    { name: "Hội trường 2", layout: "/maps/grand-hall_2.jpg" },
  ]

  const handleSubmit = async (values: WorkshopFormValues) => {
    console.log("Submitting form with values:", values)
    setIsSubmitting(true)
    try {
      await onSubmit(values)
    } catch (error) {
      console.error("Form submit error:", error)
    } finally {
      setIsSubmitting(false)
    }
  }

  const onInvalid = (errors: any) => {
    console.error("Form validation errors:", errors)
    toast.error("Vui lòng kiểm tra lại các trường thông tin màu đỏ.")
  }

  const handleAiAnalysis = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    if (file.type !== "application/pdf") {
      toast.error("Vui lòng tải lên file định dạng PDF")
      return
    }

    setIsAiProcessing(true)
    const formData = new FormData()
    formData.append("file", file)

    try {
      // Sử dụng endpoint mới hỗ trợ cả tạo mới và chỉnh sửa
      const response = await api.upload<any>(`/api/v1/ai/summarize`, formData)
      
      if (response.success && response.data?.summary) {
        form.setValue("summary", response.data.summary)
        toast.success("AI đã phân tích và tóm tắt nội dung thành công!")
      } else {
        toast.error(response.message || "Không thể lấy tóm tắt từ AI")
      }
    } catch (error) {
      console.error("AI Analysis error:", error)
      toast.error("Không thể phân tích PDF. Vui lòng kiểm tra lại file hoặc thử lại sau.")
    } finally {
      setIsAiProcessing(false)
      // Reset input file để có thể chọn lại cùng 1 file nếu cần
      event.target.value = ""
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleSubmit, onInvalid)} className="space-y-8">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Cột trái: Nội dung */}
          <div className="space-y-6 bg-white p-6 rounded-2xl border border-slate-200 shadow-sm">
            <h3 className="text-lg font-bold text-slate-900 border-b pb-3">Thông tin nội dung</h3>
            
            <FormField
              control={form.control}
              name="title"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Tiêu đề Workshop</FormLabel>
                  <FormControl>
                    <Input placeholder="Nhập tên workshop..." {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="speaker"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Diễn giả / Diễn giả chính</FormLabel>
                  <FormControl>
                    <Input placeholder="Tên diễn giả..." {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="summary"
              render={({ field }) => (
                <FormItem>
                  <div className="flex items-center justify-between mb-2">
                    <FormLabel className="mb-0">Nội dung tóm tắt & Chương trình</FormLabel>
                    <div className="relative">
                      <Input
                        type="file"
                        id="pdf-upload"
                        className="hidden"
                        accept=".pdf"
                        onChange={handleAiAnalysis}
                        disabled={isAiProcessing}
                      />
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        className="h-8 gap-2 text-indigo-600 border-indigo-200 bg-indigo-50 hover:bg-indigo-100 font-bold"
                        onClick={() => document.getElementById('pdf-upload')?.click()}
                        disabled={isAiProcessing}
                      >
                        {isAiProcessing ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                          <Sparkles className="h-3.5 w-3.5" />
                        )}
                        AI Assistant
                      </Button>
                    </div>
                  </div>
                  <FormControl>
                    <Textarea 
                      placeholder="Nhập nội dung tóm tắt buổi workshop hoặc sử dụng AI để tạo từ file PDF..." 
                      className="min-h-[250px] bg-slate-50/30 text-base leading-relaxed" 
                      {...field} 
                    />
                  </FormControl>
                  <FormDescription>
                    Bạn có thể tự nhập nội dung hoặc dùng AI Assistant để tóm tắt từ đề án PDF.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          {/* Cột phải: Kỹ thuật & Thời gian */}
          <div className="space-y-6">
            <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-6">
              <h3 className="text-lg font-bold text-slate-900 border-b pb-3">Cấu hình kỹ thuật</h3>
              
              <div className="grid grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="room"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Phòng học / Địa điểm</FormLabel>
                      <Select 
                        onValueChange={(val) => {
                          field.onChange(val)
                          const layout = ROOMS.find(r => r.name === val)?.layout
                          if (layout) form.setValue("roomLayoutUrl", layout)
                        }} 
                        defaultValue={field.value}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Chọn phòng" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {ROOMS.map(r => (
                            <SelectItem key={r.name} value={r.name}>{r.name}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="status"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Trạng thái</FormLabel>
                      <Select onValueChange={field.onChange} defaultValue={field.value}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Chọn trạng thái" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value="PUBLISHED">Đang mở (Published)</SelectItem>
                          <SelectItem value="CLOSED">Đã đóng (Closed)</SelectItem>
                          <SelectItem value="DELETED">Đã xóa (Deleted)</SelectItem>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="capacity"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Sức chứa (Seats)</FormLabel>
                      <FormControl>
                        <Input type="number" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="price"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Giá vé (VNĐ)</FormLabel>
                      <FormControl>
                        <Input type="number" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="registrationStartTime"
                  render={({ field }) => (
                    <FormItem className="flex flex-col">
                      <FormLabel>Thời gian mở đăng ký</FormLabel>
                      <div className="flex gap-2">
                        <Popover>
                          <PopoverTrigger asChild>
                            <FormControl>
                              <Button
                                variant={"outline"}
                                className={cn(
                                  "flex-1 pl-3 text-left font-normal h-12",
                                  !field.value && "text-muted-foreground"
                                )}
                              >
                                {field.value ? (
                                  format(field.value, "dd/MM/yyyy", { locale: vi })
                                ) : (
                                  <span>Chọn ngày</span>
                                )}
                                <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                              </Button>
                            </FormControl>
                          </PopoverTrigger>
                          <PopoverContent className="w-auto p-0" align="start">
                            <Calendar
                              mode="single"
                              selected={field.value}
                              onSelect={(date) => {
                                if (date) {
                                  const current = field.value || new Date()
                                  date.setHours(current.getHours())
                                  date.setMinutes(current.getMinutes())
                                  field.onChange(date)
                                }
                              }}
                              initialFocus
                            />
                          </PopoverContent>
                        </Popover>
                        <Input
                          type="time"
                          className="w-32 h-12 text-center font-bold"
                          value={field.value ? format(field.value, "HH:mm") : "08:00"}
                          onChange={(e) => {
                            if (!e.target.value) return
                            const [hours, minutes] = e.target.value.split(':').map(Number)
                            if (isNaN(hours) || isNaN(minutes)) return
                            const current = field.value || new Date()
                            const newDate = new Date(current)
                            newDate.setHours(hours)
                            newDate.setMinutes(minutes)
                            field.onChange(newDate)
                          }}
                        />
                      </div>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="registrationEndTime"
                  render={({ field }) => (
                    <FormItem className="flex flex-col">
                      <FormLabel>Thời gian đóng đăng ký</FormLabel>
                      <div className="flex gap-2">
                        <Popover>
                          <PopoverTrigger asChild>
                            <FormControl>
                              <Button
                                variant={"outline"}
                                className={cn(
                                  "flex-1 pl-3 text-left font-normal h-12",
                                  !field.value && "text-muted-foreground"
                                )}
                              >
                                {field.value ? (
                                  format(field.value, "dd/MM/yyyy", { locale: vi })
                                ) : (
                                  <span>Chọn ngày</span>
                                )}
                                <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                              </Button>
                            </FormControl>
                          </PopoverTrigger>
                          <PopoverContent className="w-auto p-0" align="start">
                            <Calendar
                              mode="single"
                              selected={field.value}
                              onSelect={(date) => {
                                if (date) {
                                  const current = field.value || new Date()
                                  date.setHours(current.getHours())
                                  date.setMinutes(current.getMinutes())
                                  field.onChange(date)
                                }
                              }}
                              initialFocus
                            />
                          </PopoverContent>
                        </Popover>
                        <Input
                          type="time"
                          className="w-32 h-12 text-center font-bold"
                          value={field.value ? format(field.value, "HH:mm") : "10:00"}
                          onChange={(e) => {
                            if (!e.target.value) return
                            const [hours, minutes] = e.target.value.split(':').map(Number)
                            if (isNaN(hours) || isNaN(minutes)) return
                            const current = field.value || new Date()
                            const newDate = new Date(current)
                            newDate.setHours(hours)
                            newDate.setMinutes(minutes)
                            field.onChange(newDate)
                          }}
                        />
                      </div>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="startTime"
                  render={({ field }) => (
                    <FormItem className="flex flex-col">
                      <FormLabel>Thời gian bắt đầu</FormLabel>
                      <div className="flex gap-2">
                        <Popover>
                          <PopoverTrigger asChild>
                            <FormControl>
                              <Button
                                variant={"outline"}
                                className={cn(
                                  "flex-1 pl-3 text-left font-normal h-12",
                                  !field.value && "text-muted-foreground"
                                )}
                              >
                                {field.value ? (
                                  format(field.value, "dd/MM/yyyy", { locale: vi })
                                ) : (
                                  <span>Chọn ngày</span>
                                )}
                                <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                              </Button>
                            </FormControl>
                          </PopoverTrigger>
                          <PopoverContent className="w-auto p-0" align="start">
                            <Calendar
                              mode="single"
                              selected={field.value}
                              onSelect={(date) => {
                                if (date) {
                                  const current = field.value || new Date()
                                  date.setHours(current.getHours())
                                  date.setMinutes(current.getMinutes())
                                  field.onChange(date)
                                }
                              }}
                              disabled={(date) =>
                                date < new Date(new Date().setHours(0, 0, 0, 0))
                              }
                              initialFocus
                            />
                          </PopoverContent>
                        </Popover>
                        <Input
                          type="time"
                          className="w-32 h-12 text-center font-bold"
                          value={field.value ? format(field.value, "HH:mm") : "08:00"}
                          onChange={(e) => {
                            if (!e.target.value) return
                            const [hours, minutes] = e.target.value.split(':').map(Number)
                            if (isNaN(hours) || isNaN(minutes)) return
                            const current = field.value || new Date()
                            const newDate = new Date(current)
                            newDate.setHours(hours)
                            newDate.setMinutes(minutes)
                            field.onChange(newDate)
                          }}
                        />
                      </div>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="endTime"
                  render={({ field }) => (
                    <FormItem className="flex flex-col">
                      <FormLabel>Thời gian kết thúc</FormLabel>
                      <div className="flex gap-2">
                        <Popover>
                          <PopoverTrigger asChild>
                            <FormControl>
                              <Button
                                variant={"outline"}
                                className={cn(
                                  "flex-1 pl-3 text-left font-normal h-12",
                                  !field.value && "text-muted-foreground"
                                )}
                              >
                                {field.value ? (
                                  format(field.value, "dd/MM/yyyy", { locale: vi })
                                ) : (
                                  <span>Chọn ngày</span>
                                )}
                                <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                              </Button>
                            </FormControl>
                          </PopoverTrigger>
                          <PopoverContent className="w-auto p-0" align="start">
                            <Calendar
                              mode="single"
                              selected={field.value}
                              onSelect={(date) => {
                                if (date) {
                                  const current = field.value || new Date()
                                  date.setHours(current.getHours())
                                  date.setMinutes(current.getMinutes())
                                  field.onChange(date)
                                }
                              }}
                              disabled={(date) =>
                                date < (form.getValues("startTime") || new Date())
                              }
                              initialFocus
                            />
                          </PopoverContent>
                        </Popover>
                        <Input
                          type="time"
                          className="w-32 h-12 text-center font-bold"
                          value={field.value ? format(field.value, "HH:mm") : "10:00"}
                          onChange={(e) => {
                            if (!e.target.value) return
                            const [hours, minutes] = e.target.value.split(':').map(Number)
                            if (isNaN(hours) || isNaN(minutes)) return
                            const current = field.value || new Date()
                            const newDate = new Date(current)
                            newDate.setHours(hours)
                            newDate.setMinutes(minutes)
                            field.onChange(newDate)
                          }}
                        />
                      </div>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>

            <div className="bg-indigo-50 p-6 rounded-2xl border border-indigo-100 flex items-start gap-4">
              <div className="bg-white p-2 rounded-lg shadow-sm">
                <Sparkles className="h-5 w-5 text-indigo-600" />
              </div>
              <div>
                <h4 className="font-bold text-indigo-900 text-sm">Mẹo nhỏ từ AI</h4>
                <p className="text-xs text-indigo-700 mt-1 leading-relaxed">
                  Hãy thử tải lên file PDF kế hoạch chi tiết của buổi workshop. AI sẽ giúp bạn tóm tắt nội dung chính xác và chuyên nghiệp hơn, giúp sinh viên dễ dàng nắm bắt mục tiêu.
                </p>
              </div>
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-4">
          <Button 
            type="button" 
            variant="outline" 
            size="lg" 
            className="px-8 rounded-xl font-bold"
            onClick={() => window.history.back()}
          >
            Hủy bỏ
          </Button>
          <Button 
            type="submit" 
            size="lg" 
            className="px-12 rounded-xl font-bold shadow-lg shadow-primary/20"
            disabled={isSubmitting}
          >
            {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isEditing ? "Cập nhật Workshop" : "Tạo Workshop mới"}
          </Button>
        </div>
      </form>
    </Form>
  )
}
