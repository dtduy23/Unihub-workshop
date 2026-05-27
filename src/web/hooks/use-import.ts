import { useState, useCallback, useEffect } from 'react'
import { api } from '@/lib/api-client'
import { toast } from 'sonner'

export interface ImportError {
  id: string
  jobId: string
  rowNumber: number
  rawData: string
  errorReason: string
}

export interface ImportJob {
  id: string
  fileName: string
  status: 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED'
  totalRecords: number
  successCount: number
  errorCount: number
  startedAt: string
  completedAt: string | null
}

export function useImport() {
  const [jobs, setJobs] = useState<ImportJob[]>([])
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)

  const fetchJobs = useCallback(async () => {
    setLoading(true)
    try {
      const response = await api.get<ImportJob[]>('/api/v1/admin/import/jobs')
      if (response.data) {
        setJobs(response.data)
      }
    } catch (error) {
      console.error('Failed to fetch jobs:', error)
      toast.error('Không thể lấy lịch sử đồng bộ')
    } finally {
      setLoading(false)
    }
  }, [])

  const uploadCSV = async (file: File) => {
    setUploading(true)
    const formData = new FormData()
    formData.append('file', file)

    try {
      const response = await api.post<any>('/api/v1/admin/import/csv', formData)
      toast.success('Đã tải lên và lập lịch xử lý', {
        description: 'Dữ liệu sẽ được đồng bộ vào lúc 02:00 sáng hoặc bạn có thể chạy thủ công.'
      })
      fetchJobs()
      return response.data
    } catch (error: any) {
      toast.error(error.message || 'Tải lên thất bại')
      throw error
    } finally {
      setUploading(false)
    }
  }

  const runJob = async (id: string) => {
    try {
      await api.post(`/api/v1/admin/import/jobs/${id}/run`, {})
      toast.success('Đang bắt đầu xử lý ngầm')
      fetchJobs()
    } catch (error: any) {
      toast.error(error.message || 'Không thể chạy job này')
    }
  }

  useEffect(() => {
    fetchJobs()
  }, [fetchJobs])

  return {
    jobs,
    loading,
    uploading,
    uploadCSV,
    runJob,
    refresh: fetchJobs
  }
}
