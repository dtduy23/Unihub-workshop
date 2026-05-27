import { useEffect } from 'react';
import { getPendingCheckins, markAsSynced } from '../services/storage';
import { API_BASE_URL } from '../services/crypto';

const SYNC_INTERVAL = 30000; // 30 giây theo specs
const API_SYNC_URL = `${API_BASE_URL}/api/checkin/sync`;

export function useSync() {
  useEffect(() => {
    const syncData = async () => {
      try {
        const pending = await getPendingCheckins();
        if (pending.length === 0) return;

        console.log(`[Sync] Đang gọi API: ${API_SYNC_URL}`);

        const response = await fetch(API_SYNC_URL, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(pending),
        });

        const contentType = response.headers.get('content-type');

        if (response.ok && contentType?.includes('application/json')) {
          const result = await response.json();
          const successfullySynced = result.synced_ids || [];

          // CHỈ XÓA các bản ghi mà Server xác nhận đã lưu thành công
          for (const id of successfullySynced) {
            await markAsSynced(id);
          }

          if (successfullySynced.length > 0) {
            console.log(`[Sync] Thành công! Đã đồng bộ ${successfullySynced.length} bản ghi.`);
          }
        } else {
          const text = await response.text();
          console.warn(`[Sync] Server trả về không phải JSON (${response.status}):`, text.substring(0, 200));
        }
      } catch (error: any) {
        console.warn('[Sync] Lỗi kết nối mạng:', error.message);
        console.warn('[Sync] Hãy kiểm tra: 1. Server đang chạy? 2. Cùng Wi-Fi? 3. Tường lửa máy tính?');
      }
    };

    const timer = setInterval(syncData, SYNC_INTERVAL);
    // Chạy lần đầu ngay khi mount
    syncData();

    return () => clearInterval(timer);
  }, []);
}
