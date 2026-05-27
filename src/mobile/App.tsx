import React, { useState, useEffect } from 'react';
import { View, Text, TouchableOpacity, StyleSheet, SafeAreaView } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { Home, History, LogOut } from 'lucide-react-native';
import LoginScreen from './src/screens/LoginScreen';
import HomeScreen from './src/screens/HomeScreen';
import ScannerScreen from './src/screens/ScannerScreen';
import SyncScreen from './src/screens/SyncScreen';
import { useAuth, type UserSession } from './src/hooks/useAuth';
import { initDatabase } from './src/services/storage';
import { useSync } from './src/hooks/useSync';

export default function App() {
  const { logout, getSession } = useAuth();
  useSync(); // Kích hoạt tiến trình đồng bộ ngầm
  const [session, setSession] = useState<UserSession | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [currentTab, setCurrentTab] = useState<'home' | 'sync'>('home');
  const [isScanning, setIsScanning] = useState(false);
  const [selectedWorkshop, setSelectedWorkshop] = useState<{id: string, title: string, room: string} | null>(null);

  // Khởi tạo DB và kiểm tra session khi mở app
  useEffect(() => {
    const init = async () => {
      await initDatabase();
      const savedSession = await getSession();
      setSession(savedSession);
      setIsLoading(false);
    };
    init();
  }, []);

  const handleLoginSuccess = (userSession: UserSession) => {
    setSession(userSession);
  };

  const handleStartScan = (id: string, title: string, room: string) => {
    setSelectedWorkshop({ id, title, room });
    setIsScanning(true);
  };

  const handleLogout = async () => {
    await logout();
    setSession(null);
  };

  // Đang kiểm tra session
  if (isLoading) return null;

  // Chưa đăng nhập -> Hiện Login
  if (!session) {
    return <LoginScreen onLoginSuccess={handleLoginSuccess} />;
  }

  // Đang mở Camera quét QR
  if (isScanning && selectedWorkshop) {
    return (
      <ScannerScreen 
        currentWorkshop={selectedWorkshop}
        onClose={() => setIsScanning(false)}
        onScanSuccess={(data) => {
          console.log('Scanned:', data);
          // TODO(team): Lưu vào SQLite khi triển khai offline — Deadline: trước demo
        }}
      />
    );
  }

  return (
    <View style={styles.container}>
      <StatusBar style="dark" />
      
      {/* Main Content Area */}
      <View style={styles.content}>
        {currentTab === 'home' ? (
          <HomeScreen onStartScan={handleStartScan} />
        ) : (
          <SyncScreen />
        )}
      </View>

      {/* Bottom Tab Bar */}
      <SafeAreaView style={styles.tabBar}>
        <TouchableOpacity 
          style={styles.tabItem} 
          onPress={() => setCurrentTab('home')}
        >
          <Home size={24} color={currentTab === 'home' ? '#1E40AF' : '#94A3B8'} />
          <Text style={[styles.tabLabel, { color: currentTab === 'home' ? '#1E40AF' : '#94A3B8' }]}>Trang chủ</Text>
        </TouchableOpacity>

        <TouchableOpacity 
          style={styles.tabItem} 
          onPress={() => setCurrentTab('sync')}
        >
          <History size={24} color={currentTab === 'sync' ? '#1E40AF' : '#94A3B8'} />
          <Text style={[styles.tabLabel, { color: currentTab === 'sync' ? '#1E40AF' : '#94A3B8' }]}>Lịch sử</Text>
        </TouchableOpacity>

        <TouchableOpacity style={styles.tabItem} onPress={handleLogout}>
          <LogOut size={24} color="#EF4444" />
          <Text style={[styles.tabLabel, { color: '#EF4444' }]}>Thoát</Text>
        </TouchableOpacity>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: 'white' },
  content: { flex: 1 },
  tabBar: { 
    flexDirection: 'row', 
    height: 90, 
    borderTopWidth: 1, 
    borderTopColor: '#F1F5F9',
    backgroundColor: 'white',
    paddingBottom: 25, // Đẩy cụm nút lên cao hơn để tránh đè thanh điều hướng
  },
  tabItem: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  tabLabel: { fontSize: 10, fontWeight: '600', marginTop: 4 },
});
