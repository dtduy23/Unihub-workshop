import { useEffect } from 'react';
import { getPendingCheckins, markAsSynced } from '../services/storage';
import { apiRequest, getToken } from '../services/api';

export function useSync() {
  useEffect(() => {
    let syncing = false;
    let stopped = false;
    const syncData = async () => {
      if (syncing || stopped || !(await getToken())) return;
      syncing = true;
      try {
        const pending = await getPendingCheckins();
        for (let offset = 0; offset < pending.length && !stopped; offset += 500) {
          const chunk = pending.slice(offset, offset + 500);
          const records = chunk.map(record => ({
            id: record.id, student_id: record.student_id, workshop_id: record.workshop_id,
            scanned_at: Math.floor(record.scanned_at > 1e12 ? record.scanned_at / 1000 : record.scanned_at),
          }));
          const { data, error } = await apiRequest<{ synced: string[]; failed: unknown[] }>('/api/v1/checkin/sync', {
            method: 'POST', body: JSON.stringify({ records }),
          });
          if (error || !data) { console.warn('[Sync]', error || 'Chưa nhận xác nhận đồng bộ'); break; }
          const ids = new Set(chunk.map(record => record.id));
          for (const id of data.synced || []) if (ids.has(id)) await markAsSynced(id);
          if (data.failed?.length) console.warn(`[Sync] ${data.failed.length} bản ghi cần kiểm tra lại.`);
        }
      } catch (error) { console.warn('[Sync]', error instanceof Error ? error.message : 'Không thể đồng bộ'); }
      finally { syncing = false; }
    };
    const timer = setInterval(() => { void syncData(); }, 30_000);
    void syncData();
    return () => { stopped = true; clearInterval(timer); };
  }, []);
}
