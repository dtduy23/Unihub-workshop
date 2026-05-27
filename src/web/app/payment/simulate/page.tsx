"use client"

import { useState, useEffect } from "react"
import { 
  CheckCircle2, 
  XCircle, 
  AlertTriangle, 
  Server, 
  Activity, 
  RefreshCcw,
  ShieldCheck,
  Zap,
  Clock,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"
import { api } from "@/lib/api-client"

export default function PaymentSimulationPage() {
  const [gatewayStatus, setGatewayStatus] = useState<"up" | "down">("up")
  const [loading, setLoading] = useState(false)
  const [stats, setStats] = useState({ success: 0, failed: 0, pending: 0 })
  const [pendingPayments, setPendingPayments] = useState<any[]>([])
  const [selectedTx, setSelectedTx] = useState<string | null>(null)

  // Polling data
  useEffect(() => {
    const fetchData = async () => {
      try {
        // 1. Fetch Gateway Status
        const statusRes = await api.get<any>("/api/v1/admin/payment/gateway-status")
        if (statusRes.data?.status) {
          setGatewayStatus(statusRes.data.status)
        }

        // 2. Fetch Pending Payments
        const pendingRes = await api.get<any[]>("/api/v1/admin/payments/pending")
        if (pendingRes.success && pendingRes.data) {
          setPendingPayments(pendingRes.data)
          setStats(prev => ({ ...prev, pending: pendingRes.data?.length || 0 }))
        }
      } catch (err) {
        console.error("Failed to fetch data", err)
      }
    }

    // Initial tx from URL
    const params = new URLSearchParams(window.location.search)
    const txFromUrl = params.get("tx")
    if (txFromUrl) setSelectedTx(txFromUrl)

    fetchData()
    const interval = setInterval(fetchData, 3000) // Poll every 3s
    return () => clearInterval(interval)
  }, [])

  const toggleGateway = async (newStatus: "up" | "down") => {
    setLoading(true)
    try {
      await api.post("/api/v1/mock/payment/toggle", { status: newStatus })
      setGatewayStatus(newStatus)
      toast.success(`Hệ thống thanh toán đã chuyển sang: ${newStatus.toUpperCase()}`)
    } catch (err) {
      toast.error("Không thể cập nhật trạng thái hệ thống")
    } finally {
      setLoading(false)
    }
  }

  const simulateSuccess = async (txID: string) => {
    setLoading(true)
    setSelectedTx(txID)
    try {
      await api.post("/api/v1/payment/webhook", {
        transaction_id: txID,
        status: "SUCCESS",
        signature: "MOCK_SIGNATURE"
      })
      toast.success("Thanh toán thành công! Webhook đã được gửi.")
      setStats(prev => ({ ...prev, success: prev.success + 1 }))
      setSelectedTx(null)
    } catch (err) {
      toast.error("Lỗi khi gửi xác nhận thanh toán")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-[#0b0f1a] text-slate-100 p-6 md:p-12 font-sans selection:bg-indigo-500/30">
      {/* Header Area */}
      <div className="max-w-7xl mx-auto mb-10 flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-4">
          <div className="flex items-center gap-3">
            <div className="bg-indigo-500 p-2.5 rounded-2xl shadow-lg shadow-indigo-500/20">
              <ShieldCheck className="h-6 w-6 text-white" />
            </div>
            <Badge variant="secondary" className="bg-indigo-500/10 text-indigo-400 border-none px-3 py-1 text-[10px] font-black uppercase tracking-[0.2em]">
              Administrative Sandbox
            </Badge>
          </div>
          <h1 className="text-5xl font-black tracking-tighter text-white">
            Payment Control <span className="text-indigo-500">Center</span>
          </h1>
          <p className="text-slate-400 text-lg max-w-2xl font-medium leading-relaxed">
            Mô phỏng phản hồi từ ngân hàng và điều khiển trạng thái Gateway để kiểm thử khả năng chịu lỗi của hệ thống UniHub.
          </p>
        </div>

        <div className="flex gap-4 p-2 bg-slate-900/50 rounded-[2rem] border border-slate-800 backdrop-blur-md">
          <div className={`flex items-center gap-2 px-6 py-3 rounded-2xl transition-all duration-500 ${gatewayStatus === 'up' ? 'bg-emerald-500/10 text-emerald-400 shadow-[0_0_20px_rgba(16,185,129,0.1)]' : 'bg-red-500/10 text-red-400'}`}>
            <Activity className={`h-5 w-5 ${gatewayStatus === 'up' ? 'animate-pulse' : ''}`} />
            <span className="text-xs font-black uppercase tracking-widest">{gatewayStatus === 'up' ? 'Gateway Active' : 'Gateway Offline'}</span>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto grid grid-cols-1 lg:grid-cols-12 gap-8">
        {/* Left Column: System Controls */}
        <div className="lg:col-span-4 space-y-8">
          <Card className="bg-slate-900/40 border-slate-800 rounded-[2.5rem] shadow-2xl backdrop-blur-xl overflow-hidden">
            <CardHeader className="p-8 border-b border-slate-800/50">
              <CardTitle className="text-xl font-bold flex items-center gap-3 text-white">
                <Zap className="h-5 w-5 text-amber-400" />
                Gateway Control
              </CardTitle>
            </CardHeader>
            <CardContent className="p-8 space-y-6">
              <Button 
                className={`w-full h-20 rounded-2xl flex flex-col items-center gap-1 transition-all duration-300 group ${gatewayStatus === 'up' ? 'bg-emerald-600 hover:bg-emerald-700 shadow-xl shadow-emerald-600/20' : 'bg-slate-800/50 text-slate-500 hover:bg-slate-800 border-2 border-dashed border-slate-700'}`}
                onClick={() => toggleGateway('up')}
                disabled={loading}
              >
                <CheckCircle2 className={`h-6 w-6 ${gatewayStatus === 'up' ? 'text-white' : 'text-slate-600'}`} />
                <span className="font-black text-sm uppercase tracking-widest">Kích hoạt Gateway</span>
              </Button>

              <Button 
                className={`w-full h-20 rounded-2xl flex flex-col items-center gap-1 transition-all duration-300 group ${gatewayStatus === 'down' ? 'bg-red-600 hover:bg-red-700 shadow-xl shadow-red-600/20' : 'bg-slate-800/50 text-slate-500 hover:bg-slate-800 border-2 border-dashed border-slate-700'}`}
                onClick={() => toggleGateway('down')}
                disabled={loading}
              >
                <XCircle className={`h-6 w-6 ${gatewayStatus === 'down' ? 'text-white' : 'text-slate-600'}`} />
                <span className="font-black text-sm uppercase tracking-widest">Đánh sập Gateway</span>
              </Button>
            </CardContent>
          </Card>

          <Card className="bg-indigo-600/10 border-indigo-500/20 rounded-[2.5rem] shadow-xl">
            <CardContent className="p-8 space-y-4">
              <div className="flex items-center gap-2 text-indigo-400">
                <RefreshCcw className="h-5 w-5 animate-spin-slow" />
                <h4 className="font-bold uppercase tracking-widest text-xs">Live Stats</h4>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="p-4 rounded-2xl bg-slate-900/50 border border-slate-800">
                  <p className="text-[10px] font-bold text-slate-500 uppercase mb-1">Success</p>
                  <p className="text-3xl font-black text-emerald-400 tracking-tighter">{stats.success}</p>
                </div>
                <div className="p-4 rounded-2xl bg-slate-900/50 border border-slate-800">
                  <p className="text-[10px] font-bold text-slate-500 uppercase mb-1">Pending</p>
                  <p className="text-3xl font-black text-amber-400 tracking-tighter">{stats.pending}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          <div className="p-8 rounded-[2rem] bg-slate-900/40 border border-slate-800 space-y-4">
            <div className="flex items-center gap-2 text-indigo-400 mb-2">
              <AlertTriangle className="h-5 w-5" />
              <h4 className="font-bold uppercase tracking-widest text-xs">Sandbox Guide</h4>
            </div>
            <ul className="space-y-3 text-[11px] text-slate-500 leading-relaxed font-medium">
              <li>1. Chọn một giao dịch từ danh sách Transaction Monitor.</li>
              <li>2. Nhấn "Confirm Payment" để mô phỏng Webhook thành công từ Ngân hàng.</li>
              <li>3. Theo dõi Live Stats để thấy thay đổi thời gian thực.</li>
            </ul>
          </div>
        </div>

        {/* Right Column: Transaction Monitor */}
        <div className="lg:col-span-8">
          <Card className="bg-slate-900/40 border-slate-800 rounded-[2.5rem] shadow-2xl backdrop-blur-xl h-full flex flex-col">
            <CardHeader className="p-8 border-b border-slate-800/50 flex flex-row items-center justify-between">
              <div className="space-y-1">
                <CardTitle className="text-2xl font-bold text-white flex items-center gap-3">
                  <Activity className="h-6 w-6 text-indigo-400" />
                  Transaction Monitor
                </CardTitle>
                <CardDescription className="text-slate-500">Danh sách các yêu cầu thanh toán đang chờ xử lý</CardDescription>
              </div>
            </CardHeader>
            <CardContent className="p-0 flex-1 flex flex-col">
              {pendingPayments.length === 0 ? (
                <div className="flex-1 flex flex-col items-center justify-center p-20 text-center space-y-4 opacity-40">
                  <div className="p-6 bg-slate-800 rounded-full">
                    <Clock className="h-12 w-12 text-slate-600" />
                  </div>
                  <p className="text-lg font-medium text-slate-400">Không có giao dịch nào đang chờ</p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse">
                    <thead>
                      <tr className="border-b border-slate-800/50">
                        <th className="px-8 py-5 text-[10px] font-bold text-slate-500 uppercase tracking-widest">Transaction ID</th>
                        <th className="px-8 py-5 text-[10px] font-bold text-slate-500 uppercase tracking-widest">Amount</th>
                        <th className="px-8 py-5 text-[10px] font-bold text-slate-500 uppercase tracking-widest text-right">Action</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-800/30">
                      {pendingPayments.map((p) => (
                        <tr key={p.id} className={`group transition-colors hover:bg-indigo-500/5 ${selectedTx === p.transactionId ? 'bg-indigo-500/10' : ''}`}>
                          <td className="px-8 py-6">
                            <div className="flex flex-col">
                              <span className="font-mono text-indigo-400 font-bold tracking-tight">{p.transactionId}</span>
                              <span className="text-[10px] text-slate-500 mt-1 font-medium">Created at: {new Date(p.createdAt).toLocaleTimeString()}</span>
                            </div>
                          </td>
                          <td className="px-8 py-6">
                            <span className="text-lg font-black text-white">{(p.amount || 0).toLocaleString()} <span className="text-[10px] text-slate-500">VNĐ</span></span>
                          </td>
                          <td className="px-8 py-6 text-right">
                            <Button 
                              size="sm"
                              className="rounded-xl bg-indigo-600 hover:bg-indigo-700 text-[10px] font-black uppercase tracking-widest px-6 h-10 shadow-lg shadow-indigo-600/20 disabled:opacity-50 disabled:grayscale transition-all"
                              onClick={() => simulateSuccess(p.transactionId)}
                              disabled={loading || gatewayStatus === 'down'}
                            >
                              {loading && selectedTx === p.transactionId ? "Confirming..." : "Confirm Payment"}
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
