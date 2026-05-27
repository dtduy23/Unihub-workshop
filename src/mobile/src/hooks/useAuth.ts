import { useState } from 'react';
import { Alert } from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { apiRequest, saveToken, clearToken } from '../services/api';

const SESSION_KEY = '@unihub_session';

/** Thông tin phiên đăng nhập lưu trên máy */
export type UserSession = {
  id: string;
  user_id: string;
  full_name: string;
  email: string;
  role: string;
};

/**
 * Hook xử lý xác thực người dùng.
 * Gọi Go Backend API thay vì Supabase trực tiếp.
 */
export function useAuth() {
  const [loading, setLoading] = useState(false);

  /** Đăng nhập bằng MSSV/email + password qua Go Backend */
  const login = async (email: string, password: string): Promise<UserSession | null> => {
    if (!email || !password) {
      Alert.alert('Thiếu thông tin', 'Vui lòng nhập Email/MSSV và Mật khẩu');
      return null;
    }

    setLoading(true);
    try {
      // Gọi Go Backend — POST /api/v1/auth/login
      const { data, error } = await apiRequest<{
        token: string;
        user: {
          id: string;
          user_id: string;
          full_name: string;
          email: string;
          role: string;
        };
      }>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({
          student_id: email,
          password: password,
        }),
      });

      if (error || !data) {
        throw new Error(error || 'Đăng nhập thất bại');
      }

      // Kiểm tra quyền — Chỉ Staff/Admin được vào Mobile
      if (data.user.role !== 'STAFF' && data.user.role !== 'ADMIN') {
        throw new Error(`Tài khoản sinh viên (${data.user.user_id}) không có quyền vào ứng dụng Staff`);
      }

      // Lưu JWT token
      await saveToken(data.token);

      // Tạo session và lưu vào AsyncStorage
      const session: UserSession = {
        id: data.user.id,
        user_id: data.user.user_id,
        full_name: data.user.full_name,
        email: data.user.email,
        role: data.user.role,
      };
      await AsyncStorage.setItem(SESSION_KEY, JSON.stringify(session));

      return session;
    } catch (error: any) {
      Alert.alert('Đăng nhập thất bại', error.message);
      return null;
    } finally {
      setLoading(false);
    }
  };

  /** Đăng xuất — xóa session và token khỏi bộ nhớ máy */
  const logout = async () => {
    await AsyncStorage.removeItem(SESSION_KEY);
    await clearToken();
  };

  /** Đọc session đã lưu (dùng khi mở lại app) */
  const getSession = async (): Promise<UserSession | null> => {
    const raw = await AsyncStorage.getItem(SESSION_KEY);
    return raw ? JSON.parse(raw) : null;
  };

  return { login, logout, getSession, loading };
}
