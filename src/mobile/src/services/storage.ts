import * as SQLite from 'expo-sqlite';

const db = SQLite.openDatabaseSync('unihub_checkin.db');

export interface CheckinRecord {
  id: string;
  student_id: string;
  user_uuid: string;
  workshop_id: string;
  workshop_title: string;
  scanned_at: number;
  sync_status: 'PENDING' | 'SYNCED';
}

export interface Workshop {
  id: string;
  title: string;
  description: string;
  room: string;
  speaker: string;
  start_time: string;
  end_time: string;
  status?: 'In Progress' | 'Upcoming' | 'Completed';
}

/**
 * Khởi tạo Database và Tự động dọn rác
 */
export const initDatabase = async () => {
  try {
    await db.execAsync(`
      PRAGMA journal_mode = WAL;

      -- Bảng lưu lịch sử quét tại máy này
      CREATE TABLE IF NOT EXISTS local_checkins (
        id TEXT PRIMARY KEY NOT NULL,
        student_id TEXT NOT NULL,
        workshop_id TEXT NOT NULL,
        data TEXT NOT NULL,
        sync_status TEXT NOT NULL DEFAULT 'PENDING',
        created_at INTEGER NOT NULL,
        UNIQUE(student_id, workshop_id)
      );

      -- Bảng lưu danh sách đen (Tất cả những người đã điểm danh trên hệ thống)
      CREATE TABLE IF NOT EXISTS already_checked_in (
        student_id TEXT NOT NULL,
        workshop_id TEXT NOT NULL,
        PRIMARY KEY (student_id, workshop_id)
      );

      -- Bảng lưu thông tin Workshop
      CREATE TABLE IF NOT EXISTS cached_workshops (
        id TEXT PRIMARY KEY NOT NULL,
        title TEXT NOT NULL,
        description TEXT,
        room TEXT NOT NULL,
        speaker TEXT,
        start_time TEXT,
        end_time TEXT,
        updated_at INTEGER NOT NULL
      );
    `);

    // TỰ ĐỘNG DỌN RÁC: Xóa dữ liệu của các workshop đã kết thúc hơn 24h
    await cleanupOldData();

    console.log('[Storage] Database ready and cleaned.');
  } catch (error) {
    console.error('[Storage] Init error:', error);
  }
};

/**
 * Logic dọn rác: Xóa bản ghi của các Workshop đã cũ
 */
const cleanupOldData = async () => {
  const oneDayAgo = Date.now() - 24 * 60 * 60 * 1000;
  // Xóa lịch sử quét và danh sách đen của các workshop đã kết thúc quá 24h
  // Dựa trên end_time trong cached_workshops
  await db.runAsync(`
    DELETE FROM local_checkins 
    WHERE workshop_id IN (
      SELECT id FROM cached_workshops WHERE datetime(end_time) < datetime(?, 'unixepoch')
    )
  `, [oneDayAgo / 1000]);
};

/**
 * Kiểm tra xem MSSV đã có trong danh sách điểm danh của Workshop này chưa
 * Kiểm tra cả Local (vừa quét xong) và Global (từ Server gửi về)
 */
export const isStudentCheckedIn = async (studentId: string, workshopId: string): Promise<boolean> => {
  try {
    // Check bảng local
    const local = await db.getFirstAsync<{count: number}>(
      'SELECT COUNT(*) as count FROM local_checkins WHERE student_id = ? AND workshop_id = ?',
      [studentId, workshopId]
    );
    if (local && local.count > 0) return true;

    // Check bảng global (danh sách đen từ server)
    const global = await db.getFirstAsync<{count: number}>(
      'SELECT COUNT(*) as count FROM already_checked_in WHERE student_id = ? AND workshop_id = ?',
      [studentId, workshopId]
    );
    return (global && global.count > 0) || false;
  } catch (e) {
    return false;
  }
};

/**
 * Lưu danh sách đen từ Server gửi về (để chặn quét trùng giữa các thiết bị)
 */
export const saveCheckedInStudents = async (data: {student_id: string, workshop_id: string}[]) => {
  for (const item of data) {
    await db.runAsync(
      'INSERT OR IGNORE INTO already_checked_in (student_id, workshop_id) VALUES (?, ?)',
      [item.student_id, item.workshop_id]
    );
  }
};

export const saveCheckinLocal = async (record: Omit<CheckinRecord, 'sync_status'>) => {
  const jsonData = JSON.stringify({ ...record, sync_status: 'PENDING' });
  await db.runAsync(
    'INSERT OR IGNORE INTO local_checkins (id, student_id, workshop_id, data, sync_status, created_at) VALUES (?, ?, ?, ?, ?, ?)',
    [record.id, record.student_id, record.workshop_id, jsonData, 'PENDING', record.scanned_at]
  );
};

export const getAllHistory = async (): Promise<CheckinRecord[]> => {
  const rows = await db.getAllAsync<{data: string, sync_status: string}>(
    "SELECT data, sync_status FROM local_checkins ORDER BY created_at DESC"
  );
  return rows.map(row => {
    try {
      const item = JSON.parse(row.data);
      return { ...item, sync_status: row.sync_status };
    } catch (e) { return null; }
  }).filter((i): i is CheckinRecord => i !== null);
};

export const getPendingCheckins = async (): Promise<CheckinRecord[]> => {
  const rows = await db.getAllAsync<{data: string}>("SELECT data FROM local_checkins WHERE sync_status = 'PENDING'");
  return rows.map(row => JSON.parse(row.data)).filter(Boolean);
};

export const markAsSynced = async (id: string) => {
  await db.runAsync("UPDATE local_checkins SET sync_status = 'SYNCED' WHERE id = ?", [id]);
};

export const saveWorkshopsOffline = async (workshops: Workshop[]) => {
  await db.runAsync('DELETE FROM cached_workshops');
  for (const ws of workshops) {
    await db.runAsync('INSERT INTO cached_workshops (id, title, description, room, speaker, start_time, end_time, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)', [ws.id, ws.title, ws.description || '', ws.room, ws.speaker || '', ws.start_time, ws.end_time, Date.now()]);
  }
};

export const getWorkshopsOffline = async (): Promise<Workshop[]> => {
  return await db.getAllAsync<Workshop>('SELECT * FROM cached_workshops');
};
