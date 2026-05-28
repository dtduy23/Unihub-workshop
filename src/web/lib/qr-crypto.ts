'use client'

/**
 * RSA Signature Verification cho QR Check-in (Web Crypto API)
 * 
 * Thay thế jsrsasign của mobile app.
 * Sử dụng Web Crypto API (SubtleCrypto) — native browser, không cần thư viện.
 * 
 * Thuật toán: RSASSA-PKCS1-v1_5 với SHA-256 (khớp với Go backend)
 */

import { api } from '@/lib/api-client'
import { cachePublicKey, getCachedPublicKey } from '@/lib/checkin-db'

// ==========================================
// QR Payload Type
// ==========================================

export interface QRPayload {
  sid: string  // Student ID (MSSV)
  uid: string  // User UUID
  wid: string  // Workshop UUID
  sig: string  // RSA Signature (Base64)
}

// ==========================================
// Public Key Management
// ==========================================

let cachedCryptoKey: CryptoKey | null = null

/** Lấy RSA Public Key từ backend, cache vào IndexedDB */
async function getPublicKey(): Promise<string> {
  // 1. Kiểm tra cache IndexedDB trước
  const cached = await getCachedPublicKey()
  if (cached) return cached

  // 2. Gọi API lấy key mới
  const response = await api.get<{ public_key: string }>('/api/v1/auth/public-key')
  const pem = (response.data as any)?.publicKey || (response.data as any)?.public_key
  
  if (!pem) throw new Error('Backend không trả về public key')

  // 3. Cache lại
  await cachePublicKey(pem)
  cachedCryptoKey = null // Reset CryptoKey cache vì key mới

  return pem
}

/** Chuyển PEM string → CryptoKey object */
async function importPublicKey(pem: string): Promise<CryptoKey> {
  if (cachedCryptoKey) return cachedCryptoKey

  // Xóa header/footer PEM và khoảng trắng
  const pemBody = pem
    .replace(/-----BEGIN PUBLIC KEY-----/, '')
    .replace(/-----END PUBLIC KEY-----/, '')
    .replace(/\s/g, '')

  // Base64 → ArrayBuffer
  const binaryString = atob(pemBody)
  const bytes = new Uint8Array(binaryString.length)
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i)
  }

  // Import key với thuật toán RSASSA-PKCS1-v1_5
  cachedCryptoKey = await crypto.subtle.importKey(
    'spki',
    bytes.buffer,
    { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-256' },
    false,
    ['verify']
  )

  return cachedCryptoKey
}

// ==========================================
// QR Parsing & Verification
// ==========================================

/** Parse raw QR string → QRPayload (xử lý cả double-encoded JSON) */
export function parseQRData(raw: string): QRPayload | null {
  try {
    const payload = JSON.parse(raw)

    let sid = String(payload.sid || '').trim()
    let uid = String(payload.uid || '').trim()
    let wid = String(payload.wid || '').trim()
    let sig = String(payload.sig || '').trim()

    // Handle double-encoded: sig có thể là nested JSON string
    if (sig.startsWith('{')) {
      try {
        const inner = JSON.parse(sig)
        sid = sid || String(inner.sid || '').trim()
        uid = uid || String(inner.uid || '').trim()
        wid = wid || String(inner.wid || '').trim()
        sig = String(inner.sig || '').trim()
      } catch {
        // Ignore parse error
      }
    }

    if (!sid || !uid || !wid || !sig) return null

    return { sid, uid, wid, sig }
  } catch {
    return null
  }
}

/** Verify RSA signature (offline — chỉ dùng Web Crypto API) */
export async function verifySignature(payload: QRPayload): Promise<boolean> {
  try {
    const pem = await getPublicKey()
    const key = await importPublicKey(pem)

    // Tạo chuỗi dữ liệu gốc (phải khớp 100% với Go backend)
    const rawData = `${payload.sid}|${payload.uid}|${payload.wid}`
    const dataBytes = new TextEncoder().encode(rawData)

    // Decode base64 signature → ArrayBuffer
    const sigBinary = atob(payload.sig)
    const sigBytes = new Uint8Array(sigBinary.length)
    for (let i = 0; i < sigBinary.length; i++) {
      sigBytes[i] = sigBinary.charCodeAt(i)
    }

    // Verify!
    const isValid = await crypto.subtle.verify(
      'RSASSA-PKCS1-v1_5',
      key,
      sigBytes.buffer,
      dataBytes.buffer
    )

    return isValid
  } catch (error) {
    console.error('[QR Crypto] Verification error:', error)
    return false
  }
}

/** Prefetch public key (gọi khi mở trang check-in) */
export async function prefetchPublicKey(): Promise<boolean> {
  try {
    await getPublicKey()
    return true
  } catch {
    return false
  }
}
