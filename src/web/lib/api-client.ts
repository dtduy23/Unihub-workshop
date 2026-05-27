/**
 * API Client — Lớp giao tiếp giữa Frontend và Go Backend.
 *
 * Chức năng chính:
 * - Tự động gắn JWT Bearer token từ cookie vào mọi request
 * - Chuyển đổi snake_case (Go) → camelCase (TS) ở response
 * - Chuyển đổi camelCase (TS) → snake_case (Go) ở request body (tuỳ chọn)
 * - Xử lý chuẩn APIResponse wrapper từ Go
 *
 * Tuân thủ: agent.md mục 10.2 và 10.3
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

// ==========================================
// Types — khớp với Go model.APIResponse
// ==========================================

export type APIResponse<T = unknown> = {
  success: boolean
  data?: T
  message?: string
  error?: string
}

// ==========================================
// snake_case ↔ camelCase converters
// ==========================================

function snakeToCamel(str: string): string {
  return str.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase())
}

function camelToSnake(str: string): string {
  return str.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`)
}

/** Chuyển đổi key của object từ snake_case sang camelCase (recursive) */
function toCamelCase<T>(obj: unknown): T {
  if (Array.isArray(obj)) {
    return obj.map((item) => toCamelCase(item)) as T
  }
  if (obj !== null && typeof obj === 'object' && !(obj instanceof Date)) {
    const converted: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
      converted[snakeToCamel(key)] = toCamelCase(value)
    }
    return converted as T
  }
  return obj as T
}

/** Chuyển đổi key của object từ camelCase sang snake_case (recursive) */
function toSnakeCase(obj: unknown): unknown {
  if (Array.isArray(obj)) {
    return obj.map((item) => toSnakeCase(item))
  }
  if (obj !== null && typeof obj === 'object' && !(obj instanceof Date)) {
    const converted: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
      converted[camelToSnake(key)] = toSnakeCase(value)
    }
    return converted
  }
  return obj
}

// ==========================================
// Cookie helper (đọc JWT token)
// ==========================================

function getAuthToken(): string | null {
  if (typeof document === 'undefined') return null
  const match = document.cookie.match(/(?:^|;\s*)unihub_token=([^;]*)/)
  return match ? decodeURIComponent(match[1]) : null
}

// ==========================================
// Core fetch wrapper
// ==========================================

type FetchOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
  body?: unknown
  headers?: Record<string, string>
  /** Không chuyển đổi snake → camel (dùng cho upload file, v.v.) */
  rawResponse?: boolean
}

/**
 * Gọi API Go Backend.
 * Tự động gắn JWT, chuyển đổi case, và unwrap APIResponse.
 *
 * @throws Error khi success=false hoặc HTTP lỗi
 */
async function fetchAPI<T = unknown>(
  endpoint: string,
  options: FetchOptions = {}
): Promise<APIResponse<T>> {
  const { method = 'GET', body, headers = {}, rawResponse = false } = options

  const url = `${API_BASE_URL}${endpoint}`

  const requestHeaders: Record<string, string> = {
    ...headers,
  }

  // Gắn JWT token
  const token = getAuthToken()
  if (token) {
    requestHeaders['Authorization'] = `Bearer ${token}`
  }

  // Chuẩn bị request
  const fetchInit: RequestInit = {
    method,
    headers: requestHeaders,
  }

  if (body !== undefined && body !== null) {
    if (body instanceof FormData) {
      fetchInit.body = body
    } else {
      requestHeaders['Content-Type'] = 'application/json'
      // Chuyển đổi camelCase (TS) → snake_case (Go)
      fetchInit.body = JSON.stringify(toSnakeCase(body))
    }
  }

  try {
    const response = await fetch(url, fetchInit)

    // Xử lý trường hợp không có body (204 No Content)
    if (response.status === 204) {
      return { success: true } as APIResponse<T>
    }

    const json = await response.json()

    // Chuyển đổi snake_case → camelCase
    const converted = rawResponse ? json : toCamelCase<APIResponse<T>>(json)

    // Nếu HTTP lỗi hoặc success=false, throw để caller xử lý
    if (!response.ok || !converted.success) {
      const errorMessage = converted.error || converted.message || `HTTP ${response.status}`
      
      // Tự động clear session nếu token hết hạn (401)
      if (response.status === 401) {
        auth.clearSession()
        if (typeof window !== 'undefined') {
          window.location.href = '/login' // Chuyển hướng về trang login
        }
      }
      
      throw new APIError(errorMessage, response.status, response.headers)
    }

    return converted
  } catch (error) {
    // Log lỗi chi tiết để debug (đã ẩn theo yêu cầu người dùng)
    
    if (error instanceof APIError) throw error
    
    // Lỗi network (Fail to fetch)
    throw new Error('Không thể kết nối đến server. Vui lòng kiểm tra backend đang chạy tại port 8080.')
  }
}

// ==========================================
// Custom Error class — giữ HTTP status và headers
// ==========================================

export class APIError extends Error {
  status: number
  headers: Headers

  constructor(message: string, status: number, headers: Headers) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.headers = headers
  }

  /** Lấy giá trị Retry-After header (dùng cho HTTP 429) */
  get retryAfter(): number | null {
    const value = this.headers.get('Retry-After')
    return value ? parseInt(value, 10) : null
  }
}

// ==========================================
// Public API methods
// ==========================================

export const api = {
  get: <T = unknown>(endpoint: string, options?: Omit<FetchOptions, 'method' | 'body'>) =>
    fetchAPI<T>(endpoint, { ...options, method: 'GET' }),

  post: <T = unknown>(endpoint: string, body?: unknown, options?: Omit<FetchOptions, 'method' | 'body'>) =>
    fetchAPI<T>(endpoint, { ...options, method: 'POST', body }),

  put: <T = unknown>(endpoint: string, body?: unknown, options?: Omit<FetchOptions, 'method' | 'body'>) =>
    fetchAPI<T>(endpoint, { ...options, method: 'PUT', body }),

  delete: <T = unknown>(endpoint: string, options?: Omit<FetchOptions, 'method' | 'body'>) =>
    fetchAPI<T>(endpoint, { ...options, method: 'DELETE' }),

  /**
   * Upload file (FormData) — Dùng cho CSV import và PDF upload.
   * Không set Content-Type header (browser tự thêm multipart boundary).
   */
  upload: <T = unknown>(endpoint: string, formData: FormData) =>
    fetchAPI<T>(endpoint, { method: 'POST', body: formData }),
}

// ==========================================
// Auth helpers — Quản lý JWT token và session
// ==========================================

const SESSION_COOKIE = 'unihub_session'
const TOKEN_COOKIE = 'unihub_token'

export const auth = {
  /** Lưu JWT token và thông tin user vào cookie sau khi login thành công */
  setSession(token: string, user: Record<string, unknown>) {
    const maxAge = 60 * 60 * 24 // 24 giờ
    document.cookie = `${TOKEN_COOKIE}=${encodeURIComponent(token)}; path=/; max-age=${maxAge}; SameSite=Lax`
    document.cookie = `${SESSION_COOKIE}=${encodeURIComponent(JSON.stringify(user))}; path=/; max-age=${maxAge}; SameSite=Lax`
  },

  /** Xóa session (logout) */
  clearSession() {
    document.cookie = `${TOKEN_COOKIE}=; path=/; max-age=0`
    document.cookie = `${SESSION_COOKIE}=; path=/; max-age=0`
  },

  /** Đọc thông tin user từ session cookie */
  getUser(): Record<string, unknown> | null {
    if (typeof document === 'undefined') return null
    const match = document.cookie.match(/(?:^|;\s*)unihub_session=([^;]*)/)
    if (!match) return null
    try {
      return JSON.parse(decodeURIComponent(match[1]))
    } catch {
      return null
    }
  },

  /** Kiểm tra có đang đăng nhập không */
  isAuthenticated(): boolean {
    return getAuthToken() !== null
  },
}
