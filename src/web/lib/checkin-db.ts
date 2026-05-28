'use client'

/**
 * IndexedDB Storage cho Check-in Offline
 * 
 * Thay thế SQLite của mobile app.
 * Data lưu trên ổ đĩa thiết bị qua browser IndexedDB engine.
 */

import { openDB, type DBSchema, type IDBPDatabase } from 'idb'

// ==========================================
// Schema Definition
// ==========================================

interface CheckinDB extends DBSchema {
  checkins: {
    key: string
    value: CheckinRecord
    indexes: {
      'by-sync': string
      'by-workshop': string
      'by-student-workshop': [string, string]
    }
  }
  publicKey: {
    key: string
    value: { id: string; pem: string; fetchedAt: number }
  }
}

export interface CheckinRecord {
  id: string
  studentId: string
  userUuid: string
  workshopId: string
  workshopTitle: string
  scannedAt: number
  syncStatus: 'PENDING' | 'SYNCED' | 'FAILED'
}

// ==========================================
// Database Initialization
// ==========================================

let dbPromise: Promise<IDBPDatabase<CheckinDB>> | null = null

function getDB(): Promise<IDBPDatabase<CheckinDB>> {
  if (!dbPromise) {
    dbPromise = openDB<CheckinDB>('unihub-checkin', 1, {
      upgrade(db) {
        // Bảng lưu lịch sử check-in
        const checkinStore = db.createObjectStore('checkins', { keyPath: 'id' })
        checkinStore.createIndex('by-sync', 'syncStatus')
        checkinStore.createIndex('by-workshop', 'workshopId')
        checkinStore.createIndex('by-student-workshop', ['studentId', 'workshopId'])

        // Bảng cache RSA public key
        db.createObjectStore('publicKey', { keyPath: 'id' })
      },
    })
  }
  return dbPromise
}

// ==========================================
// Check-in CRUD Operations
// ==========================================

/** Lưu 1 bản ghi check-in (offline) */
export async function saveCheckin(record: CheckinRecord): Promise<void> {
  const db = await getDB()
  await db.put('checkins', record)
}

/** Kiểm tra sinh viên đã check-in workshop này chưa (local) */
export async function isAlreadyCheckedIn(studentId: string, workshopId: string): Promise<boolean> {
  const db = await getDB()
  const record = await db.getFromIndex('checkins', 'by-student-workshop', [studentId, workshopId])
  return !!record
}

/** Lấy tất cả bản ghi PENDING (chưa sync) */
export async function getPendingCheckins(): Promise<CheckinRecord[]> {
  const db = await getDB()
  return db.getAllFromIndex('checkins', 'by-sync', 'PENDING')
}

/** Đánh dấu bản ghi đã sync thành công */
export async function markAsSynced(id: string): Promise<void> {
  const db = await getDB()
  const record = await db.get('checkins', id)
  if (record) {
    record.syncStatus = 'SYNCED'
    await db.put('checkins', record)
  }
}

/** Đánh dấu bản ghi sync thất bại */
export async function markAsFailed(id: string): Promise<void> {
  const db = await getDB()
  const record = await db.get('checkins', id)
  if (record) {
    record.syncStatus = 'FAILED'
    await db.put('checkins', record)
  }
}

/** Lấy toàn bộ lịch sử check-in (cho hiển thị) */
export async function getAllCheckins(): Promise<CheckinRecord[]> {
  const db = await getDB()
  const all = await db.getAll('checkins')
  // Sắp xếp mới nhất trước
  return all.sort((a, b) => b.scannedAt - a.scannedAt)
}

/** Lấy số lượt đã quét cho 1 workshop */
export async function getCheckinCount(workshopId: string): Promise<number> {
  const db = await getDB()
  const records = await db.getAllFromIndex('checkins', 'by-workshop', workshopId)
  return records.length
}

// ==========================================
// Public Key Cache
// ==========================================

/** Lưu RSA public key */
export async function cachePublicKey(pem: string): Promise<void> {
  const db = await getDB()
  await db.put('publicKey', { id: 'rsa-public-key', pem, fetchedAt: Date.now() })
}

/** Lấy RSA public key từ cache */
export async function getCachedPublicKey(): Promise<string | null> {
  const db = await getDB()
  const record = await db.get('publicKey', 'rsa-public-key')
  if (!record) return null
  
  // Key hết hạn sau 24 giờ
  const ONE_DAY = 24 * 60 * 60 * 1000
  if (Date.now() - record.fetchedAt > ONE_DAY) return null
  
  return record.pem
}
