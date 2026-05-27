import { useState, useEffect, useCallback } from 'react';
import { apiRequest } from '../services/api';
import { getWorkshopsOffline, saveWorkshopsOffline, Workshop } from '../services/storage';

export function useWorkshops() {
  const [workshops, setWorkshops] = useState<Workshop[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  /**
   * Hàm bổ trợ để tính toán trạng thái dựa trên thời gian thực
   */
  const processWorkshops = (data: any[]) => {
    const now = new Date();

    const processed = data.map((ws) => {
      const start = new Date(ws.start_time);
      const end = new Date(ws.end_time);

      let status: 'In Progress' | 'Upcoming' | 'Completed';
      if (now >= start && now <= end) {
        status = 'In Progress';
      } else if (now < start) {
        status = 'Upcoming';
      } else {
        status = 'Completed';
      }

      return { ...ws, status };
    });

    // Sắp xếp theo thứ tự ưu tiên hiển thị
    return processed.sort((a, b) => {
      const priority = { 'In Progress': 1, 'Upcoming': 2, 'Completed': 3 };
      return priority[a.status] - priority[b.status];
    });
  };

  const fetchWorkshops = useCallback(async (isBackground = false) => {
    if (!isBackground) setLoading(true);

    try {
      // Gọi Go Backend — GET /api/v1/workshops
      const { data, error } = await apiRequest<any[]>('/api/v1/workshops');

      if (error || !data) throw new Error(error || 'Fetch failed');

      const finalData = processWorkshops(data);
      setWorkshops(finalData);
      await saveWorkshopsOffline(finalData); // Lưu vào máy để dùng lần sau

    } catch (error) {
      console.warn('[Offline Mode] Đang nạp dữ liệu từ bộ nhớ máy...');
      const cached = await getWorkshopsOffline();
      if (cached.length > 0) {
        // TÍNH TOÁN LẠI TRẠNG THÁI CHO DỮ LIỆU OFFLINE
        const finalCached = processWorkshops(cached);
        setWorkshops(finalCached);
      }
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    const init = async () => {
      const cached = await getWorkshopsOffline();
      if (cached.length > 0) {
        // Hiển thị dữ liệu Offline ngay lập tức với trạng thái được tính toán lại
        setWorkshops(processWorkshops(cached));
        setLoading(false);
        fetchWorkshops(true); // Cập nhật ngầm
      } else {
        fetchWorkshops();
      }
    };
    init();
  }, [fetchWorkshops]);

  const onRefresh = () => {
    setRefreshing(true);
    fetchWorkshops();
  };

  return { workshops, loading, refreshing, onRefresh };
}
