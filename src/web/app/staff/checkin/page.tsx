'use client'

/**
 * Trang Check-in — Quét QR bằng Camera trên trình duyệt
 * 
 * Dành cho Staff — thay thế hoàn toàn Mobile App (Expo/React Native).
 * Hỗ trợ offline: lưu IndexedDB → auto-sync khi có mạng.
 */

import { useState, useEffect, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import {
  ScanLine,
  ArrowLeft,
  Wifi,
  WifiOff,
  Clock,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  RefreshCw,
  PlayCircle,
  History,
  Loader2,
  LogOut,
} from 'lucide-react'
import { toast } from 'sonner'

import { QRScanner } from '@/components/admin/qr-scanner'
import { parseQRData, verifySignature, prefetchPublicKey } from '@/lib/qr-crypto'
import {
  saveCheckin,
  isAlreadyCheckedIn,
  getCheckinCount,
  getAllCheckins,
  type CheckinRecord,
} from '@/lib/checkin-db'
import { useCheckinSync } from '@/hooks/use-checkin-sync'
import { api } from '@/lib/api-client'
import { auth } from '@/lib/api-client'

// ==========================================
// Types
// ==========================================

type Workshop = {
  id: string
  title: string
  room: string
  startTime: string
  endTime: string
  status: string
  capacity: number
  availableSeats: number
}

type ScanResult = {
  status: 'success' | 'error' | 'warning' | 'processing'
  message: string
}

type ViewMode = 'select' | 'scan' | 'history'

// ==========================================
// Main Component
// ==========================================

export default function StaffCheckinPage() {
  const router = useRouter()
  const { isOnline, pendingCount, isSyncing, syncNow, refreshCount } = useCheckinSync()

  // State
  const [viewMode, setViewMode] = useState<ViewMode>('select')
  const [workshops, setWorkshops] = useState<Workshop[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedWorkshop, setSelectedWorkshop] = useState<Workshop | null>(null)
  const [scanResult, setScanResult] = useState<ScanResult | null>(null)
  const [scanCount, setScanCount] = useState(0)
  const [history, setHistory] = useState<CheckinRecord[]>([])
  const [keyReady, setKeyReady] = useState(false)

  // Fetch workshops
  useEffect(() => {
    const fetchWorkshops = async () => {
      try {
        const response = await api.get<Workshop[]>('/api/v1/workshops')
        const data = (response.data || []) as Workshop[]

        // Thêm trạng thái thời gian thực
        const now = new Date()
        const enriched = data.map((w) => {
          const start = new Date(w.startTime)
          const end = new Date(w.endTime)
          let timeStatus = 'upcoming'
          if (now >= start && now <= end) timeStatus = 'in-progress'
          else if (now > end) timeStatus = 'completed'
          return { ...w, timeStatus }
        })

        // Sắp xếp: đang diễn ra trước
        enriched.sort((a, b) => {
          const order: Record<string, number> = { 'in-progress': 0, upcoming: 1, completed: 2 }
          return (order[(a as any).timeStatus] ?? 3) - (order[(b as any).timeStatus] ?? 3)
        })

        setWorkshops(enriched)
      } catch (err) {
        toast.error('Không thể tải danh sách workshop')
      } finally {
        setLoading(false)
      }
    }

    fetchWorkshops()
  }, [])

  // Prefetch RSA public key
  useEffect(() => {
    prefetchPublicKey().then(setKeyReady)
  }, [])

  // Load scan count khi chọn workshop
  useEffect(() => {
    if (selectedWorkshop) {
      getCheckinCount(selectedWorkshop.id).then(setScanCount)
    }
  }, [selectedWorkshop])

  // Handle QR scan
  const handleScan = useCallback(
    async (rawData: string) => {
      if (!selectedWorkshop) return

      setScanResult({ status: 'processing', message: 'Đang xử lý...' })

      // 1. Parse QR
      const payload = parseQRData(rawData)
      if (!payload) {
        setScanResult({ status: 'error', message: '❌ MÃ QR KHÔNG HỢP LỆ' })
        clearResult()
        return
      }

      // 2. Verify RSA (offline)
      if (keyReady) {
        const isValid = await verifySignature(payload)
        if (!isValid) {
          setScanResult({ status: 'error', message: '❌ CHỮ KÝ KHÔNG HỢP LỆ' })
          clearResult()
          return
        }
      }

      // 3. Check workshop match
      if (payload.wid !== selectedWorkshop.id) {
        setScanResult({ status: 'warning', message: `⚠️ SAI WORKSHOP! (${selectedWorkshop.room})` })
        clearResult()
        return
      }

      // 4. Check duplicate (local)
      const alreadyDone = await isAlreadyCheckedIn(payload.sid, payload.wid)
      if (alreadyDone) {
        setScanResult({ status: 'warning', message: `⚠️ ĐÃ QUÉT RỒI (${payload.sid})` })
        clearResult()
        return
      }

      // 5. Lưu check-in
      const record: CheckinRecord = {
        id: `${payload.sid}_${Date.now()}`,
        studentId: payload.sid,
        userUuid: payload.uid,
        workshopId: payload.wid,
        workshopTitle: selectedWorkshop.title,
        scannedAt: Date.now(),
        syncStatus: 'PENDING',
      }

      await saveCheckin(record)
      setScanCount((prev) => prev + 1)
      refreshCount()

      // 6. Nếu online, gọi API trực tiếp luôn
      if (navigator.onLine) {
        try {
          await api.post('/api/v1/checkin/live', {
            registration_id: '',
            workshop_id: payload.wid,
            student_id: payload.sid,
          })
          // API thành công → đánh dấu synced
          record.syncStatus = 'SYNCED'
          await saveCheckin(record)
          refreshCount()
        } catch {
          // API lỗi → giữ PENDING, sync sau
        }
      }

      setScanResult({ status: 'success', message: `✅ THÀNH CÔNG: ${payload.sid}` })
      clearResult()
    },
    [selectedWorkshop, keyReady, refreshCount]
  )

  const clearResult = () => {
    setTimeout(() => setScanResult(null), 1500)
  }

  // Start scanning
  const handleStartScan = (workshop: Workshop) => {
    setSelectedWorkshop(workshop)
    setViewMode('scan')
  }

  // Load history
  const handleShowHistory = async () => {
    const records = await getAllCheckins()
    setHistory(records)
    setViewMode('history')
  }

  // Logout
  const handleLogout = () => {
    auth.clearSession()
    toast.success('Đăng xuất thành công')
    router.push('/login')
  }

  const getTimeStatus = (w: Workshop) => {
    const now = new Date()
    const start = new Date(w.startTime)
    const end = new Date(w.endTime)
    if (now >= start && now <= end) return 'in-progress'
    if (now > end) return 'completed'
    return 'upcoming'
  }

  const formatTime = (dateStr: string) => {
    const d = new Date(dateStr)
    return d.toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit', hour12: false })
  }

  const formatDate = (timestamp: number) => {
    const d = new Date(timestamp)
    return `${d.getHours()}:${d.getMinutes().toString().padStart(2, '0')} — ${d.getDate()}/${d.getMonth() + 1}`
  }

  // ==========================================
  // RENDER: Scanner Mode (Full-screen)
  // ==========================================

  if (viewMode === 'scan' && selectedWorkshop) {
    return (
      <div className="fixed inset-0 z-50 bg-black flex flex-col">
        {/* Banner */}
        <div className="bg-indigo-900 px-4 py-3 flex items-center gap-3 z-10">
          <button
            onClick={() => setViewMode('select')}
            className="p-2 rounded-xl bg-white/10 hover:bg-white/20 transition-colors"
          >
            <ArrowLeft className="w-5 h-5 text-white" />
          </button>
          <div className="flex-1 min-w-0">
            <p className="text-indigo-200 text-xs font-medium">Đang quét tại: {selectedWorkshop.room}</p>
            <p className="text-white font-bold truncate">{selectedWorkshop.title}</p>
          </div>
          <div className="flex items-center gap-2">
            {isOnline ? (
              <Wifi className="w-4 h-4 text-green-400" />
            ) : (
              <WifiOff className="w-4 h-4 text-yellow-400" />
            )}
          </div>
        </div>

        {/* Camera */}
        <div className="flex-1 relative">
          <QRScanner onScan={handleScan} isActive={true} />

          {/* Scan Result Toast */}
          {scanResult && scanResult.status !== 'processing' && (
            <div
              className={`absolute bottom-32 left-1/2 -translate-x-1/2 px-6 py-3 rounded-full font-bold text-white text-sm shadow-2xl z-30 animate-in fade-in zoom-in-95 duration-200 ${
                scanResult.status === 'success'
                  ? 'bg-emerald-500'
                  : scanResult.status === 'error'
                  ? 'bg-red-500'
                  : 'bg-amber-500'
              }`}
            >
              {scanResult.message}
            </div>
          )}

          {scanResult?.status === 'processing' && (
            <div className="absolute bottom-32 left-1/2 -translate-x-1/2 z-30">
              <Loader2 className="w-8 h-8 text-white animate-spin" />
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="bg-black/90 px-6 py-4 flex items-center justify-between">
          <div className="bg-white rounded-2xl px-5 py-2.5 text-center">
            <p className="text-slate-400 text-[10px] font-semibold uppercase tracking-wider">Đã quét</p>
            <p className="text-slate-900 text-xl font-bold">{scanCount}</p>
          </div>
          {pendingCount > 0 && (
            <div className="flex items-center gap-2 text-yellow-400 text-sm">
              <Clock className="w-4 h-4" />
              <span className="font-medium">{pendingCount} chờ đồng bộ</span>
            </div>
          )}
        </div>
      </div>
    )
  }

  // ==========================================
  // RENDER: History Mode
  // ==========================================

  if (viewMode === 'history') {
    return (
      <div className="p-6 md:p-8 max-w-4xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex items-center gap-4">
          <button
            onClick={() => setViewMode('select')}
            className="p-2 rounded-xl bg-slate-100 hover:bg-slate-200 transition-colors"
          >
            <ArrowLeft className="w-5 h-5 text-slate-600" />
          </button>
          <div>
            <h1 className="text-2xl font-bold text-slate-900">Lịch sử quét</h1>
            <p className="text-sm text-slate-500">Dữ liệu check-in trên thiết bị này</p>
          </div>
          <div className="ml-auto flex items-center gap-2">
            <span className="bg-indigo-50 text-indigo-700 px-3 py-1 rounded-full text-sm font-bold">
              {history.length} bản ghi
            </span>
            {pendingCount > 0 && (
              <span className="bg-amber-50 text-amber-700 px-3 py-1 rounded-full text-sm font-bold">
                {pendingCount} chờ sync
              </span>
            )}
          </div>
        </div>

        {/* Records */}
        {history.length === 0 ? (
          <div className="text-center py-20 text-slate-400">
            <History className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p className="font-medium">Chưa có dữ liệu quét nào</p>
          </div>
        ) : (
          <div className="space-y-3">
            {history.map((item) => (
              <div
                key={item.id}
                className={`bg-white rounded-2xl border p-4 transition-all ${
                  item.syncStatus === 'SYNCED'
                    ? 'border-emerald-100'
                    : item.syncStatus === 'FAILED'
                    ? 'border-red-100'
                    : 'border-slate-100'
                }`}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="font-bold text-slate-900">{item.workshopTitle}</span>
                  {item.syncStatus === 'SYNCED' ? (
                    <CheckCircle2 className="w-5 h-5 text-emerald-500" />
                  ) : item.syncStatus === 'FAILED' ? (
                    <XCircle className="w-5 h-5 text-red-500" />
                  ) : (
                    <Clock className="w-5 h-5 text-amber-500" />
                  )}
                </div>
                <div className="flex items-center gap-4 text-sm text-slate-500">
                  <span>MSSV: <strong className="text-slate-700">{item.studentId}</strong></span>
                  <span>{formatDate(item.scannedAt)}</span>
                  {item.syncStatus === 'SYNCED' && (
                    <span className="text-emerald-600 text-xs font-bold bg-emerald-50 px-2 py-0.5 rounded-full">
                      Đã đồng bộ
                    </span>
                  )}
                  {item.syncStatus === 'PENDING' && (
                    <span className="text-amber-600 text-xs font-bold bg-amber-50 px-2 py-0.5 rounded-full">
                      Chờ đồng bộ
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  // ==========================================
  // RENDER: Workshop Selection (Default)
  // ==========================================

  return (
    <div className="p-6 md:p-8 max-w-4xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-3">
            <ScanLine className="w-7 h-7 text-indigo-600" />
            Check-in QR
          </h1>
          <p className="text-sm text-slate-500 mt-1">Chọn workshop để bắt đầu quét mã check-in</p>
        </div>
        <div className="flex items-center gap-3">
          {/* Online/Offline indicator */}
          <div
            className={`flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-bold ${
              isOnline
                ? 'bg-emerald-50 text-emerald-700'
                : 'bg-amber-50 text-amber-700'
            }`}
          >
            {isOnline ? <Wifi className="w-3.5 h-3.5" /> : <WifiOff className="w-3.5 h-3.5" />}
            {isOnline ? 'Online' : 'Offline'}
          </div>

          {/* Sync button */}
          {pendingCount > 0 && (
            <button
              onClick={syncNow}
              disabled={isSyncing || !isOnline}
              className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-indigo-50 text-indigo-700 text-xs font-bold hover:bg-indigo-100 transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isSyncing ? 'animate-spin' : ''}`} />
              {pendingCount} chờ sync
            </button>
          )}

          {/* History button */}
          <button
            onClick={handleShowHistory}
            className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-slate-100 text-slate-600 text-xs font-bold hover:bg-slate-200 transition-colors"
          >
            <History className="w-3.5 h-3.5" />
            Lịch sử
          </button>

          {/* Logout button */}
          <button
            onClick={handleLogout}
            className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-red-50 text-red-600 text-xs font-bold hover:bg-red-100 transition-colors"
          >
            <LogOut className="w-3.5 h-3.5" />
            Thoát
          </button>
        </div>
      </div>

      {/* Offline banner */}
      {!isOnline && (
        <div className="bg-amber-50 border border-amber-200 rounded-2xl p-4 flex items-center gap-3">
          <WifiOff className="w-5 h-5 text-amber-600 shrink-0" />
          <div>
            <p className="font-bold text-amber-800 text-sm">Đang hoạt động offline</p>
            <p className="text-amber-600 text-xs">Check-in vẫn hoạt động bình thường. Dữ liệu sẽ tự đồng bộ khi có mạng.</p>
          </div>
        </div>
      )}

      {/* RSA Key status */}
      {!keyReady && isOnline && (
        <div className="bg-blue-50 border border-blue-200 rounded-2xl p-4 flex items-center gap-3">
          <Loader2 className="w-5 h-5 text-blue-600 animate-spin shrink-0" />
          <p className="text-blue-700 text-sm font-medium">Đang tải RSA key để xác thực chữ ký...</p>
        </div>
      )}

      {/* Workshop List */}
      {loading ? (
        <div className="flex flex-col items-center justify-center py-20 gap-3">
          <Loader2 className="w-8 h-8 text-indigo-600 animate-spin" />
          <p className="text-slate-500 font-medium">Đang tải danh sách workshop...</p>
        </div>
      ) : workshops.length === 0 ? (
        <div className="text-center py-20 text-slate-400">
          <ScanLine className="w-12 h-12 mx-auto mb-3 opacity-50" />
          <p className="font-medium">Không có workshop nào</p>
        </div>
      ) : (
        <div className="space-y-4">
          {workshops.map((workshop) => {
            const timeStatus = getTimeStatus(workshop)
            const isActive = timeStatus === 'in-progress'
            const isCompleted = timeStatus === 'completed'

            return (
              <div
                key={workshop.id}
                className={`bg-white rounded-2xl border p-5 transition-all hover:shadow-md ${
                  isActive
                    ? 'border-indigo-200 bg-indigo-50/30 shadow-sm'
                    : isCompleted
                    ? 'border-red-100 bg-red-50/20 opacity-75'
                    : 'border-slate-200'
                }`}
              >
                {/* Card Header */}
                <div className="flex items-center justify-between mb-3">
                  <span className="bg-indigo-50 text-indigo-800 px-3 py-1 rounded-lg text-xs font-bold">
                    {workshop.room}
                  </span>
                  {isActive && (
                    <span className="flex items-center gap-1.5 bg-emerald-50 text-emerald-700 px-2.5 py-1 rounded-lg text-xs font-bold">
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                      Đang diễn ra
                    </span>
                  )}
                  {timeStatus === 'upcoming' && (
                    <span className="bg-amber-50 text-amber-700 px-2.5 py-1 rounded-lg text-xs font-bold">
                      Sắp diễn ra
                    </span>
                  )}
                  {isCompleted && (
                    <span className="bg-red-50 text-red-600 px-2.5 py-1 rounded-lg text-xs font-bold">
                      Đã kết thúc
                    </span>
                  )}
                </div>

                {/* Title */}
                <h3 className="text-lg font-bold text-slate-900 mb-2">{workshop.title}</h3>

                {/* Time */}
                <div className="flex items-center gap-2 text-sm text-slate-500 mb-4">
                  <Clock className="w-4 h-4" />
                  <span>
                    {formatTime(workshop.startTime)} - {formatTime(workshop.endTime)}
                  </span>
                </div>

                {/* Action Button */}
                <button
                  onClick={() => handleStartScan(workshop)}
                  disabled={isCompleted}
                  className={`w-full flex items-center justify-center gap-2 py-3.5 rounded-xl font-bold text-sm transition-all ${
                    isActive
                      ? 'bg-indigo-900 text-white hover:bg-indigo-800 shadow-lg shadow-indigo-500/20'
                      : isCompleted
                      ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                      : 'bg-slate-200 text-slate-500 hover:bg-slate-300'
                  }`}
                >
                  <PlayCircle className="w-5 h-5" />
                  {isActive ? 'Bắt đầu Quét mã' : isCompleted ? 'Đã kết thúc' : 'Chưa đến giờ'}
                </button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
