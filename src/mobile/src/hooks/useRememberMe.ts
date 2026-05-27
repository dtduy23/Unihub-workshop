import { useState, useEffect } from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';

const REMEMBER_DATA_KEY = '@unihub_remembered_data';

export function useRememberMe() {
  const [rememberedEmail, setRememberedEmail] = useState('');
  const [rememberedPassword, setRememberedPassword] = useState('');
  const [isRememberActive, setIsRememberActive] = useState(false);

  // Load dữ liệu đã lưu khi khởi động
  useEffect(() => {
    const loadData = async () => {
      const jsonValue = await AsyncStorage.getItem(REMEMBER_DATA_KEY);
      if (jsonValue) {
        const data = JSON.parse(jsonValue);
        setRememberedEmail(data.email);
        setRememberedPassword(data.password);
        setIsRememberActive(true);
      }
    };
    loadData();
  }, []);

  // Lưu hoặc Xóa dữ liệu
  const saveOrClearData = async (email: string, password: string, shouldRemember: boolean) => {
    if (shouldRemember) {
      const data = { email, password };
      await AsyncStorage.setItem(REMEMBER_DATA_KEY, JSON.stringify(data));
    } else {
      await AsyncStorage.removeItem(REMEMBER_DATA_KEY);
    }
  };

  return { 
    rememberedEmail, 
    rememberedPassword, 
    isRememberActive, 
    setIsRememberActive, 
    saveOrClearData 
  };
}
