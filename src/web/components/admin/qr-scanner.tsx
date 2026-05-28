'use client'

/**
 * QR Scanner Component — Wrapper cho html5-qrcode
 * 
 * Full-screen camera viewfinder cho mobile browser.
 * Quét QR liên tục, debounce 1.5s giữa các lần quét.
 */

import { useEffect, useRef, useCallback, useState } from 'react'
import { Html5Qrcode } from 'html5-qrcode'

interface QRScannerProps {
  onScan: (data: string) => void
  isActive: boolean
}

export function QRScanner({ onScan, isActive }: QRScannerProps) {
  const scannerRef = useRef<Html5Qrcode | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const isLockedRef = useRef(false)
  const [hasPermission, setHasPermission] = useState<boolean | null>(null)
  const [error, setError] = useState<string | null>(null)

  const handleScan = useCallback(
    (decodedText: string) => {
      if (isLockedRef.current) return
      isLockedRef.current = true

      onScan(decodedText)

      // Reset lock sau 1.5s
      setTimeout(() => {
        isLockedRef.current = false
      }, 1500)
    },
    [onScan]
  )

  useEffect(() => {
    if (!isActive) return

    const scannerId = 'qr-scanner-region'
    let scanner: Html5Qrcode | null = null

    const startScanner = async () => {
      try {
        scanner = new Html5Qrcode(scannerId)
        scannerRef.current = scanner

        await scanner.start(
          { facingMode: 'environment' }, // Camera sau trên mobile
          {
            fps: 10,
            qrbox: { width: 250, height: 250 },
            aspectRatio: 1,
          },
          handleScan,
          () => {} // Ignore scan failures (no QR in frame)
        )
        setHasPermission(true)
        setError(null)
      } catch (err: any) {
        console.error('[Scanner] Failed to start:', err)
        setHasPermission(false)
        setError(err?.message || 'Không thể truy cập camera')
      }
    }

    startScanner()

    return () => {
      if (scanner && scanner.isScanning) {
        scanner.stop().catch(() => {})
      }
      scannerRef.current = null
    }
  }, [isActive, handleScan])

  if (!isActive) return null

  if (hasPermission === false) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-4 p-8 text-center">
        <div className="w-16 h-16 rounded-2xl bg-red-50 flex items-center justify-center">
          <svg className="w-8 h-8 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
        </div>
        <p className="text-slate-600 font-medium">Cần quyền truy cập Camera để quét mã QR</p>
        <p className="text-sm text-slate-400">{error}</p>
        <button
          onClick={() => window.location.reload()}
          className="px-6 py-3 bg-indigo-600 text-white rounded-xl font-semibold hover:bg-indigo-700 transition-colors"
        >
          Thử lại
        </button>
      </div>
    )
  }

  return (
    <div className="relative w-full h-full bg-black flex items-center justify-center">
      <div
        id="qr-scanner-region"
        ref={containerRef}
        className="w-full max-w-md"
      />
      
      {/* Scanning indicator */}
      {hasPermission === null && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/80">
          <div className="flex flex-col items-center gap-3">
            <div className="w-10 h-10 border-4 border-white/30 border-t-white rounded-full animate-spin" />
            <p className="text-white/80 text-sm font-medium">Đang mở camera...</p>
          </div>
        </div>
      )}
    </div>
  )
}
