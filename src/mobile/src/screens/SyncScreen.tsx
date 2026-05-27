import React, { useState, useCallback, useEffect, useRef } from 'react';
import { View, Text, StyleSheet, SafeAreaView, FlatList, ActivityIndicator, StatusBar, Platform } from 'react-native';
import { Clock, CheckCircle2, Database, User, MapPin, Calendar } from 'lucide-react-native';
import { getAllHistory, CheckinRecord } from '../services/storage';

export default function SyncScreen() {
  const [historyList, setHistoryList] = useState<CheckinRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const isFetching = useRef(false); // Cờ ngăn chặn truy vấn chồng chéo

  const fetchHistory = useCallback(async () => {
    if (isFetching.current) return; // Nếu đang tải thì bỏ qua lượt này

    isFetching.current = true;
    try {
      const data = await getAllHistory();
      setHistoryList(data);
    } catch (error) {
      console.error('Lỗi tải lịch sử:', error);
    } finally {
      setLoading(false);
      isFetching.current = false;
    }
  }, []);

  useEffect(() => {
    fetchHistory();
    // Tăng thời gian giãn cách lên 3 giây để Database được nghỉ ngơi
    const interval = setInterval(fetchHistory, 3000);
    return () => clearInterval(interval);
  }, [fetchHistory]);

  const formatDate = (timestamp: number) => {
    const d = new Date(timestamp);
    return `${d.getHours()}:${d.getMinutes().toString().padStart(2, '0')} - ${d.getDate()}/${d.getMonth() + 1}/${d.getFullYear()}`;
  };

  const pendingCount = historyList.filter(item => item.sync_status === 'PENDING').length;

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.header}>
        <View>
          <Text style={styles.title}>Lịch sử quét</Text>
          <Text style={styles.subtitle}>Tất cả dữ liệu đã quét trên máy</Text>
        </View>
        <View style={styles.statsContainer}>
          <View style={[styles.statBadge, { backgroundColor: '#EFF6FF' }]}>
            <Text style={[styles.statText, { color: '#1E40AF' }]}>Tổng: {historyList.length}</Text>
          </View>
          {pendingCount > 0 && (
            <View style={[styles.statBadge, { backgroundColor: '#FFFBEB' }]}>
              <Text style={[styles.statText, { color: '#B45309' }]}>Chờ: {pendingCount}</Text>
            </View>
          )}
        </View>
      </View>

      {loading && historyList.length === 0 ? (
        <View style={styles.center}><ActivityIndicator color="#1E40AF" /></View>
      ) : (
        <FlatList
          data={historyList}
          keyExtractor={(item, index) => item.id + index}
          contentContainerStyle={styles.list}
          ListEmptyComponent={
            <View style={styles.empty}>
              <Database size={48} color="#CBD5E1" />
              <Text style={styles.emptyText}>Chưa có dữ liệu quét nào.</Text>
            </View>
          }
          renderItem={({ item }) => (
            <View style={[styles.item, item.sync_status === 'SYNCED' ? styles.syncedBorder : styles.pendingBorder]}>
              <View style={styles.itemHeader}>
                <View style={[styles.workshopIcon, { backgroundColor: item.sync_status === 'SYNCED' ? '#ECFDF5' : '#FFFBEB' }]}>
                  <Calendar size={18} color={item.sync_status === 'SYNCED' ? '#10B981' : '#F59E0B'} />
                </View>
                <Text style={styles.workshopTitle} numberOfLines={1}>{item.workshop_title || 'Workshop'}</Text>
                {item.sync_status === 'SYNCED' ? (
                  <CheckCircle2 size={18} color="#10B981" />
                ) : (
                  <Clock size={18} color="#F59E0B" />
                )}
              </View>

              <View style={styles.details}>
                <View style={styles.row}>
                  <User size={14} color="#64748B" />
                  <Text style={styles.detailText}>MSSV: <Text style={styles.bold}>{item.student_id}</Text></Text>
                  {item.sync_status === 'SYNCED' && (
                    <Text style={styles.syncedLabel}>Đã đồng bộ</Text>
                  )}
                </View>

                <View style={styles.row}>
                  <MapPin size={14} color="#64748B" />
                  <Text style={styles.detailText}>ID Workshop: {item.workshop_id.substring(0, 8)}...</Text>
                </View>

                <View style={styles.row}>
                  <Clock size={14} color="#64748B" />
                  <Text style={styles.timeText}>{formatDate(item.scanned_at)}</Text>
                </View>
              </View>
            </View>
          )}
        />
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#F8FAFC' },
  header: {
    flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center',
    paddingHorizontal: 24,
    paddingTop: Platform.OS === 'android' ? (StatusBar.currentHeight || 0) + 16 : 16,
    paddingBottom: 16,
    backgroundColor: 'white',
    borderBottomWidth: 1,
    borderBottomColor: '#F1F5F9'
  },
  title: { fontSize: 24, fontWeight: 'bold', color: '#0F172A' },
  subtitle: { fontSize: 14, color: '#64748B', marginTop: 2 },
  statsContainer: { alignItems: 'flex-end', gap: 4 },
  statBadge: { paddingHorizontal: 10, paddingVertical: 4, borderRadius: 12 },
  statText: { fontSize: 12, fontWeight: 'bold' },
  list: { padding: 16 },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  empty: { flex: 1, alignItems: 'center', marginTop: 100 },
  emptyText: { color: '#94A3B8', marginTop: 16, fontSize: 14 },
  item: {
    backgroundColor: 'white', padding: 16, borderRadius: 20, marginBottom: 12,
    borderWidth: 1, elevation: 2, shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 }, shadowOpacity: 0.05, shadowRadius: 8
  },
  syncedBorder: { borderColor: '#ECFDF5' },
  pendingBorder: { borderColor: '#F1F5F9' },
  itemHeader: { flexDirection: 'row', alignItems: 'center', marginBottom: 12, borderBottomWidth: 1, borderBottomColor: '#F1F5F9', paddingBottom: 12 },
  workshopIcon: { padding: 8, borderRadius: 10, marginRight: 10 },
  workshopTitle: { fontSize: 16, fontWeight: 'bold', color: '#1E293B', flex: 1 },
  details: { gap: 8 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  detailText: { fontSize: 14, color: '#475569' },
  bold: { fontWeight: 'bold', color: '#0F172A' },
  timeText: { fontSize: 13, color: '#64748B' },
  syncedLabel: { fontSize: 10, backgroundColor: '#ECFDF5', color: '#10B981', paddingHorizontal: 6, paddingVertical: 2, borderRadius: 10, fontWeight: 'bold', marginLeft: 'auto' },
});
