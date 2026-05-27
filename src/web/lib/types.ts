/**
 * TypeScript Types — khớp 1:1 với Go `internal/model/models.go`
 *
 * Quy tắc (agent.md mục 10.3):
 * - Backend Go dùng snake_case trong JSON
 * - Frontend TS dùng camelCase
 * - Chuyển đổi thực hiện tại api-client.ts, types ở đây dùng camelCase
 *
 * Quy tắc (agent.md mục 3.3):
 * - Type dùng PascalCase, không prefix I/T
 * - Dùng `as const` thay cho enum
 */

// ==========================================
// User
// ==========================================

export const Role = {
  STUDENT: 'STUDENT',
  STAFF: 'STAFF',
  ORGANIZER: 'ORGANIZER',
} as const
export type Role = (typeof Role)[keyof typeof Role]

export type User = {
  id: string
  studentId: string
  fullName: string
  email: string
  phone?: string
  role: Role
  createdAt: string
  updatedAt: string
}

// ==========================================
// Workshop
// ==========================================

export const WorkshopStatus = {
  PUBLISHED: 'PUBLISHED',
  CANCELLED: 'CANCELLED',
  DRAFT: 'DRAFT',
} as const
export type WorkshopStatus = (typeof WorkshopStatus)[keyof typeof WorkshopStatus]

export type Workshop = {
  id: string
  title: string
  description?: string
  speaker: string
  room: string
  startTime: string
  endTime: string
  capacity: number
  availableSeats: number
  price: number
  summary?: string
  status: WorkshopStatus
  createdAt: string
}

// ==========================================
// Registration
// ==========================================

export const RegistrationStatus = {
  PENDING_PAYMENT: 'PENDING_PAYMENT',
  SUCCESS: 'SUCCESS',
  FAILED: 'FAILED',
  CANCELLED: 'CANCELLED',
  REJECTED: 'REJECTED',
} as const
export type RegistrationStatus = (typeof RegistrationStatus)[keyof typeof RegistrationStatus]

export type Registration = {
  id: string
  userId: string
  workshopId: string
  status: RegistrationStatus
  qrCode?: string
  isCheckedIn: boolean
  scannedAt?: string
  createdAt: string
}

export type RegistrationStatusResponse = {
  correlationId: string
  status: RegistrationStatus | 'PROCESSING'
  registration?: Registration
  message?: string
}

// ==========================================
// Payment
// ==========================================

export const PaymentStatus = {
  PENDING: 'PENDING',
  SUCCESS: 'SUCCESS',
  FAILED: 'FAILED',
  CANCELLED: 'CANCELLED',
} as const
export type PaymentStatus = (typeof PaymentStatus)[keyof typeof PaymentStatus]

export type Payment = {
  id: string
  registrationId: string
  transactionId: string
  amount: number
  provider: string
  status: PaymentStatus
  createdAt: string
}

// ==========================================
// Notification
// ==========================================

export const NotificationChannel = {
  EMAIL: 'EMAIL',
  WEB: 'WEB',
} as const
export type NotificationChannel = (typeof NotificationChannel)[keyof typeof NotificationChannel]

export const NotificationStatus = {
  PENDING: 'PENDING',
  SENT: 'SENT',
  FAILED: 'FAILED',
} as const
export type NotificationStatus = (typeof NotificationStatus)[keyof typeof NotificationStatus]

export type Notification = {
  id: string
  userId: string
  registrationId?: string
  channel: NotificationChannel
  title: string
  content: string
  status: NotificationStatus
  eventId: string
  errorMessage?: string
  createdAt: string
  sentAt?: string
}

// ==========================================
// Import Job (Admin)
// ==========================================

export const ImportJobStatus = {
  PROCESSING: 'PROCESSING',
  COMPLETED: 'COMPLETED',
  FAILED: 'FAILED',
} as const
export type ImportJobStatus = (typeof ImportJobStatus)[keyof typeof ImportJobStatus]

export type ImportJob = {
  id: string
  fileName: string
  status: ImportJobStatus
  totalRecords: number
  successCount: number
  errorCount: number
  startedAt: string
  completedAt?: string
}

// ==========================================
// Request DTOs (gửi lên Go dùng snake_case)
// ==========================================

export type LoginRequest = {
  student_id: string
  password: string
}

export type LoginResponse = {
  token: string
  user: User
}

export type RegisterWorkshopRequest = {
  workshop_id: string
}

export type CreateWorkshopRequest = {
  title: string
  description: string
  speaker: string
  room: string
  start_time: string
  end_time: string
  capacity: number
  price: number
}

export type UpdateWorkshopRequest = {
  title?: string
  description?: string
  speaker?: string
  room?: string
  start_time?: string
  end_time?: string
  capacity?: number
  price?: number
  status?: string
}

export type CheckinRequest = {
  registration_id: string
  workshop_id: string
  student_id: string
}

export type BulkCheckinRequest = {
  records: OfflineCheckinRecord[]
}

export type OfflineCheckinRecord = {
  id: string
  student_id: string
  workshop_id: string
  scanned_at: number
}
