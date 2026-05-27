"use client"

import { useState, useRef } from "react"
import { 
  Database, 
  Upload, 
  Play, 
  CheckCircle2, 
  XCircle, 
  Clock, 
  Loader2, 
  FileText,
  AlertTriangle,
  RefreshCcw,
  Search,
  ChevronRight
} from "lucide-react"
import { PageHeader } from "@/components/admin/page-header"
import { Button } from "@/components/ui/button"
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { useImport, ImportJob, ImportError } from "@/hooks/use-import"
import { format } from "date-fns"
import { vi } from "date-fns/locale"
import { api } from "@/lib/api-client"
import { toast } from "sonner"

export default function DataSyncPage() {
  const { jobs, loading, uploading, uploadCSV, runJob, refresh } = useImport()
  const fileInputRef = useRef<HTMLInputElement>(null)
  
  // State for error details
  const [selectedJob, setSelectedJob] = useState<ImportJob | null>(null)
  const [errors, setErrors] = useState<ImportError[]>([])
  const [loadingErrors, setLoadingErrors] = useState(false)
  const [isSheetOpen, setIsSheetOpen] = useState(false)

  const fetchErrors = async (job: ImportJob) => {
    setSelectedJob(job)
    setLoadingErrors(true)
    setIsSheetOpen(true)
    try {
      const response = await api.get<ImportError[]>(`/api/v1/admin/import/jobs/${job.id}/errors`)
      if (response.data) {
        setErrors(response.data)
      }
    } catch (err) {
      toast.error("Không thể tải chi tiết lỗi")
    } finally {
      setLoadingErrors(false)
    }
  }

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      try {
        await uploadCSV(file)
        if (fileInputRef.current) fileInputRef.current.value = ""
      } catch (err) {
        // Error handled in hook
      }
    }
  }

  const [runningJobIds, setRunningJobIds] = useState<Set<string>>(new Set())

  const handleRunJob = async (id: string) => {
    setRunningJobIds(prev => new Set(prev).add(id))
    try {
      await runJob(id)
      // We don't remove it from runningJobIds immediately because 
      // the status will change to PROCESSING on next refresh anyway
    } catch (err) {
      setRunningJobIds(prev => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    }
  }

  const getStatusBadge = (status: ImportJob['status']) => {
    switch (status) {
      case 'COMPLETED':
        return <Badge className="bg-emerald-50 text-emerald-700 border-emerald-100 hover:bg-emerald-50 gap-1 rounded-lg">
          <CheckCircle2 className="h-3 w-3" /> Thành công
        </Badge>
      case 'PROCESSING':
        return <Badge className="bg-primary text-white border-transparent animate-pulse hover:bg-primary gap-1 rounded-lg shadow-sm">
          <Loader2 className="h-3 w-3 animate-spin" /> Đang xử lý
        </Badge>
      case 'PENDING':
        return <Badge className="bg-amber-50 text-amber-700 border-amber-100 hover:bg-amber-50 gap-1 rounded-lg">
          <Clock className="h-3 w-3" /> Chờ xử lý
        </Badge>
      case 'FAILED':
        return <Badge className="bg-rose-50 text-rose-700 border-rose-100 hover:bg-rose-50 gap-1 rounded-lg">
          <XCircle className="h-3 w-3" /> Thất bại
        </Badge>
      default:
        return <Badge variant="secondary">{status}</Badge>
    }
  }

  return (
    <div className="p-8 space-y-8 max-w-7xl mx-auto">
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4">
        <PageHeader 
          title="Trung tâm Điều phối Dữ liệu"
          description="Quản lý đồng bộ dữ liệu sinh viên hàng loạt từ file CSV. Hệ thống tự động chạy vào lúc 02:00 sáng hàng ngày."
        />
        <div className="flex items-center gap-3">
          <Button variant="outline" onClick={refresh} disabled={loading} className="rounded-xl border-slate-200">
            <RefreshCcw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} /> Làm mới
          </Button>
          <Button onClick={() => fileInputRef.current?.click()} disabled={uploading} className="rounded-xl bg-primary hover:bg-primary/90 shadow-lg shadow-indigo-100">
            {uploading ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Upload className="h-4 w-4 mr-2" />}
            Tải lên CSV
          </Button>
          <input 
            type="file" 
            ref={fileInputRef} 
            onChange={handleFileChange} 
            accept=".csv" 
            className="hidden" 
          />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Stats Cards */}
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center gap-4">
          <div className="h-12 w-12 rounded-xl bg-indigo-50 flex items-center justify-center text-primary">
            <Database className="h-6 w-6" />
          </div>
          <div>
            <p className="text-sm font-medium text-slate-500">Tổng số tệp</p>
            <p className="text-2xl font-bold text-slate-900">{jobs.length}</p>
          </div>
        </div>

        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center gap-4">
          <div className="h-12 w-12 rounded-xl bg-emerald-50 flex items-center justify-center text-emerald-600">
            <CheckCircle2 className="h-6 w-6" />
          </div>
          <div>
            <p className="text-sm font-medium text-slate-500">Xử lý thành công</p>
            <p className="text-2xl font-bold text-slate-900">
              {jobs.filter(j => j.status === 'COMPLETED').length}
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center gap-4">
          <div className="h-12 w-12 rounded-xl bg-amber-50 flex items-center justify-center text-amber-600">
            <Clock className="h-6 w-6" />
          </div>
          <div>
            <p className="text-sm font-medium text-slate-500">Đang chờ xử lý</p>
            <p className="text-2xl font-bold text-slate-900">
              {jobs.filter(j => j.status === 'PENDING').length}
            </p>
          </div>
        </div>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white overflow-hidden shadow-sm">
        <div className="p-6 border-b border-slate-100 flex items-center justify-between bg-slate-50/50">
          <h2 className="text-lg font-bold text-slate-900 flex items-center gap-2">
            <FileText className="h-5 w-5 text-primary" />
            Lịch sử điều phối dữ liệu
          </h2>
          <div className="flex items-center gap-2 text-sm text-slate-500 italic">
            <Clock className="h-4 w-4" />
            Tự động chạy tiếp theo: 02:00 AM Ngày mai
          </div>
        </div>
        
        <div className="overflow-x-auto">
          <Table>
            <TableHeader className="bg-slate-50/30">
              <TableRow>
                <TableHead className="font-bold">Tên tệp</TableHead>
                <TableHead className="font-bold">Trạng thái</TableHead>
                <TableHead className="font-bold text-center">Tổng dòng</TableHead>
                <TableHead className="font-bold text-center">Thành công</TableHead>
                <TableHead className="font-bold text-center">Lỗi</TableHead>
                <TableHead className="font-bold">Bắt đầu lúc</TableHead>
                <TableHead className="font-bold text-right">Thao tác</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {jobs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="h-32 text-center text-slate-500">
                    Chưa có lịch sử đồng bộ dữ liệu nào.
                  </TableCell>
                </TableRow>
              ) : (
                jobs.map((job) => (
                  <TableRow key={job.id} className="hover:bg-slate-50/50 transition-colors">
                    <TableCell className="font-medium text-slate-900">{job.fileName}</TableCell>
                    <TableCell>{getStatusBadge(job.status)}</TableCell>
                    <TableCell className="text-center font-bold text-slate-600">{job.totalRecords || '-'}</TableCell>
                    <TableCell className="text-center font-bold text-emerald-600">{job.successCount || '-'}</TableCell>
                    <TableCell className="text-center font-bold text-rose-600">
                      {job.errorCount > 0 ? (
                        <div className="flex items-center justify-center gap-1">
                          <AlertTriangle className="h-3 w-3" /> {job.errorCount}
                        </div>
                      ) : (
                        job.errorCount || '-'
                      )}
                    </TableCell>
                    <TableCell className="text-slate-500 text-sm">
                      {format(new Date(job.startedAt), "HH:mm, dd/MM/yyyy", { locale: vi })}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        {job.errorCount > 0 && (
                          <Button 
                            variant="outline" 
                            size="sm" 
                            onClick={() => fetchErrors(job)}
                            className="rounded-lg h-8 border-rose-200 text-rose-600 hover:bg-rose-50 hover:text-rose-700"
                          >
                            <AlertTriangle className="h-3 w-3 mr-1" /> Lỗi
                          </Button>
                        )}
                        {job.status === 'PENDING' && (
                          <Button 
                            size="sm" 
                            onClick={() => handleRunJob(job.id)} 
                            disabled={runningJobIds.has(job.id)}
                            className="rounded-lg bg-primary hover:bg-primary/90 disabled:opacity-70 shadow-md shadow-indigo-500/10 h-8 gap-1 text-white font-bold min-w-[100px]"
                          >
                            {runningJobIds.has(job.id) ? (
                              <><Loader2 className="h-3 w-3 animate-spin" /> Đang chạy</>
                            ) : (
                              <><Play className="h-3 w-3 fill-current" /> Chạy ngay</>
                            )}
                          </Button>
                        )}
                        {job.status === 'COMPLETED' && (
                          <span className="text-xs font-bold text-slate-400 px-3">Đã hoàn tất</span>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      {/* Error Details Sheet */}
      <Sheet open={isSheetOpen} onOpenChange={setIsSheetOpen}>
        <SheetContent className="sm:max-w-2xl overflow-y-auto">
          <SheetHeader className="space-y-4">
            <div className="h-12 w-12 rounded-2xl bg-rose-50 flex items-center justify-center text-rose-600">
              <AlertTriangle className="h-6 w-6" />
            </div>
            <div>
              <SheetTitle className="text-2xl font-bold text-slate-900">Chi tiết lỗi đồng bộ</SheetTitle>
              <SheetDescription className="text-slate-500">
                Tệp: <span className="font-bold text-slate-700">{selectedJob?.fileName}</span>
              </SheetDescription>
            </div>
          </SheetHeader>

          <div className="mt-8 space-y-6">
            {loadingErrors ? (
              <div className="flex flex-col items-center justify-center py-20 gap-4">
                <Loader2 className="h-8 w-8 animate-spin text-primary" />
                <p className="text-sm text-slate-500 font-medium">Đang tải danh sách lỗi...</p>
              </div>
            ) : errors.length === 0 ? (
              <div className="text-center py-20 bg-slate-50 rounded-2xl border border-dashed border-slate-200">
                <p className="text-slate-500">Không tìm thấy dữ liệu lỗi.</p>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-bold text-slate-900">Danh sách {errors.length} dòng lỗi</span>
                  <Badge variant="outline" className="text-rose-600 border-rose-100 bg-rose-50">Cần kiểm tra lại</Badge>
                </div>
                
                <div className="space-y-3">
                  {errors.map((error, idx) => (
                    <div key={error.id || idx} className="p-4 rounded-xl border border-slate-200 bg-white hover:border-rose-200 transition-colors group">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex-1 space-y-2">
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-bold bg-slate-100 text-slate-600 px-2 py-0.5 rounded">Dòng {error.rowNumber}</span>
                            <span className="text-sm font-bold text-rose-600">{error.errorReason}</span>
                          </div>
                          <div className="bg-slate-50 p-3 rounded-lg border border-slate-100 overflow-x-auto">
                            <code className="text-[11px] text-slate-600 whitespace-nowrap">{error.rawData}</code>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>

      {/* Information Box */}
      <div className="bg-indigo-50 border border-indigo-100 rounded-2xl p-6 flex gap-4">
        <div className="h-10 w-10 rounded-full bg-white flex items-center justify-center text-primary shadow-sm flex-shrink-0">
          <AlertTriangle className="h-5 w-5" />
        </div>
        <div className="space-y-1">
          <h3 className="font-bold text-indigo-900">Lưu ý quan trọng cho quản trị viên</h3>
          <p className="text-sm text-indigo-700/80 leading-relaxed">
            Hệ thống sử dụng cơ chế <span className="font-bold">Bulk Insert</span> để nạp dữ liệu tốc độ cao. 
            Nếu tệp CSV chứa MSSV đã tồn tại, thông tin của sinh viên đó sẽ được cập nhật (Upsert). 
            Để đảm bảo hiệu năng tốt nhất cho người dùng đang đăng ký workshop, chúng tôi khuyến nghị bạn 
            nên để hệ thống tự động xử lý các tệp lớn vào ban đêm.
          </p>
        </div>
      </div>
    </div>
  )
}
