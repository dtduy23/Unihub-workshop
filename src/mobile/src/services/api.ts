/**
 * API Client cho Go Backend
 * Thay thế Supabase — tất cả data đi qua 1 nguồn duy nhất
 */
import AsyncStorage from '@react-native-async-storage/async-storage';
import { API_BASE_URL } from './crypto';

const TOKEN_KEY = '@unihub_token';

/**
 * Lưu JWT token sau khi login thành công
 */
export const saveToken = async (token: string) => {
  await AsyncStorage.setItem(TOKEN_KEY, token);
};

/**
 * Đọc JWT token đã lưu
 */
export const getToken = async (): Promise<string | null> => {
  return await AsyncStorage.getItem(TOKEN_KEY);
};

/**
 * Xóa JWT token (logout)
 */
export const clearToken = async () => {
  await AsyncStorage.removeItem(TOKEN_KEY);
};

/**
 * Gọi API tới Go Backend với JWT token tự động
 */
export async function apiRequest<T = any>(
  path: string,
  options: RequestInit = {}
): Promise<{ data: T | null; error: string | null }> {
  try {
    const token = await getToken();
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options.headers as Record<string, string>) || {}),
    };

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE_URL}${path}`, {
      ...options,
      headers,
    });

    const json = await response.json();

    if (!response.ok) {
      return { data: null, error: json.error || `HTTP ${response.status}` };
    }

    return { data: json.data || json, error: null };
  } catch (error: any) {
    return { data: null, error: error.message || 'Network error' };
  }
}
