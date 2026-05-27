import React from 'react';
import { View, Text, SafeAreaView, ScrollView, TouchableOpacity, StyleSheet, RefreshControl, ActivityIndicator, StatusBar, Platform } from 'react-native';
import { PlayCircle, Clock } from 'lucide-react-native';
import { useWorkshops } from '../hooks/useWorkshops';
import { theme } from '../constants/theme';

export default function HomeScreen({ onStartScan }: { onStartScan: (id: string, title: string, room: string) => void }) {
  const { workshops, loading, refreshing, onRefresh } = useWorkshops();

  const formatTimeRange = (startStr: string, endStr: string) => {
    const start = new Date(startStr);
    const end = new Date(endStr);
    const d = start.getDate().toString().padStart(2, '0');
    const m = (start.getMonth() + 1).toString().padStart(2, '0');
    const y = start.getFullYear();
    const timeOptions: Intl.DateTimeFormatOptions = { hour: '2-digit', minute: '2-digit', hour12: false };
    const startTime = start.toLocaleTimeString([], timeOptions);
    const endTime = end.toLocaleTimeString([], timeOptions);
    return `${d}/${m}/${y}  •  ${startTime} - ${endTime}`;
  };

  if (loading && !refreshing) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color={theme.colors.primary} />
        <Text style={styles.loadingText}>Đang tải danh sách sự kiện...</Text>
      </View>
    );
  }

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Sự kiện hôm nay</Text>
        <Text style={styles.headerSubtitle}>Tự động cập nhật trạng thái</Text>
      </View>

      <ScrollView
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} colors={[theme.colors.primary]} />
        }
      >
        {workshops.length === 0 ? (
          <View style={styles.emptyState}>
            <Text style={styles.emptyText}>Không có sự kiện nào được tìm thấy</Text>
          </View>
        ) : (
          workshops.map((item) => (
            <View
              key={item.id}
              style={[
                styles.card,
                item.status === 'In Progress' && styles.cardActive,
                item.status === 'Completed' && styles.cardCompleted
              ]}
            >
              <View style={styles.cardHeader}>
                <View style={styles.tag}>
                  <Text style={styles.tagText}>{item.room}</Text>
                </View>
                {item.status === 'In Progress' && (
                  <View style={styles.liveTag}>
                    <View style={styles.dot} />
                    <Text style={styles.liveText}>Đang diễn ra</Text>
                  </View>
                )}
                {item.status === 'Upcoming' && (
                  <View style={styles.upcomingTag}>
                    <Text style={styles.upcomingText}>Sắp diễn ra</Text>
                  </View>
                )}
                {item.status === 'Completed' && (
                  <View style={styles.completedTag}>
                    <Text style={styles.completedText}>Đã kết thúc</Text>
                  </View>
                )}
              </View>

              <Text style={styles.cardTitle}>{item.title}</Text>

              <View style={styles.infoRow}>
                <Clock size={16} color={item.status === 'In Progress' ? theme.colors.primary : theme.colors.textMuted} />
                <Text style={[
                  styles.infoText,
                  item.status === 'In Progress' && styles.infoTextActive,
                  item.status === 'Completed' && styles.infoTextCompleted
                ]}>
                  {formatTimeRange(item.start_time, item.end_time)}
                </Text>
              </View>

              <TouchableOpacity
                style={[
                  styles.button,
                  item.status !== 'In Progress' && styles.buttonDisabled
                ]}
                onPress={() => onStartScan(item.id, item.title, item.room)}
                disabled={item.status !== 'In Progress'}
                activeOpacity={0.7}
              >
                <PlayCircle size={20} color="white" />
                <Text style={styles.buttonText}>
                  {item.status === 'In Progress' ? 'Bắt đầu Quét mã' : 'Chưa đến giờ'}
                </Text>
              </TouchableOpacity>
            </View>
          ))
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: theme.colors.background },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center', backgroundColor: theme.colors.background },
  loadingText: { marginTop: 12, color: theme.colors.textMuted, fontSize: 14 },
  header: {
    paddingHorizontal: 24,
    paddingTop: Platform.OS === 'android' ? (StatusBar.currentHeight || 0) + 16 : 16,
    paddingBottom: 16,
    backgroundColor: 'white',
    borderBottomWidth: 1,
    borderBottomColor: '#F1F5F9'
  },
  headerTitle: { fontSize: 24, fontWeight: 'bold', color: theme.colors.text },
  headerSubtitle: { fontSize: 14, color: theme.colors.textMuted, marginTop: 2 },
  scrollContent: { padding: 20 },
  card: {
    backgroundColor: theme.colors.card,
    borderRadius: theme.radius.xl,
    padding: 20,
    marginBottom: 16,
    borderWidth: 1,
    borderColor: theme.colors.border,
    elevation: 2,
    shadowColor: theme.colors.text,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.05,
    shadowRadius: 12,
  },
  cardActive: { backgroundColor: '#EEF2FF', borderColor: '#C7D2FE', borderWidth: 1.5 },
  cardCompleted: { backgroundColor: '#FFF1F2', borderColor: '#FECDD3', opacity: 0.9 },
  cardHeader: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 12 },
  tag: { backgroundColor: '#EEF2FF', paddingHorizontal: 12, paddingVertical: 6, borderRadius: 10 },
  tagText: { color: '#312E81', fontSize: 12, fontWeight: '700' },
  liveTag: { flexDirection: 'row', alignItems: 'center', backgroundColor: '#DCFCE7', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 8 },
  dot: { width: 6, height: 6, borderRadius: 3, backgroundColor: '#22C55E', marginRight: 6 },
  liveText: { color: '#15803D', fontSize: 12, fontWeight: '600' },
  upcomingTag: { backgroundColor: '#FEF9C3', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 8 },
  upcomingText: { color: '#854D0E', fontSize: 12, fontWeight: '600' },
  completedTag: { backgroundColor: '#FFE4E6', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 8 },
  completedText: { color: '#E11D48', fontSize: 12, fontWeight: '600' },
  cardTitle: { fontSize: 19, fontWeight: 'bold', color: theme.colors.text, marginBottom: 12 },
  infoRow: { flexDirection: 'row', alignItems: 'center', marginBottom: 20 },
  infoText: { color: theme.colors.textMuted, fontSize: 14, marginLeft: 8, fontWeight: '500' },
  infoTextActive: { color: '#312E81' },
  infoTextCompleted: { color: '#F43F5E' },
  button: {
    backgroundColor: '#312E81',
    flexDirection: 'row',
    height: 52,
    borderRadius: 14,
    alignItems: 'center',
    justifyContent: 'center',
    elevation: 3,
  },
  buttonDisabled: { backgroundColor: '#CBD5E1', elevation: 0 },
  buttonText: { color: 'white', fontWeight: 'bold', marginLeft: 8, fontSize: 16 },
  emptyState: { alignItems: 'center', marginTop: 40 },
  emptyText: { color: theme.colors.textMuted, fontSize: 16 },
});
