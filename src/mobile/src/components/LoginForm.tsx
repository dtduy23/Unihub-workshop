import React, { useState, useEffect } from 'react';
import { View, Text, TextInput, TouchableOpacity, ActivityIndicator, Alert, StyleSheet } from 'react-native';
import { Mail, Lock, Eye, EyeOff, CheckSquare, Square } from 'lucide-react-native';
import { useAuth, type UserSession } from '../hooks/useAuth';
import { useRememberMe } from '../hooks/useRememberMe';

export function LoginForm({ onLoginSuccess }: { onLoginSuccess: (s: UserSession) => void }) {
  const { login, loading } = useAuth();
  const { rememberedEmail, rememberedPassword, isRememberActive, setIsRememberActive, saveOrClearData } = useRememberMe();
  
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);

  // Khi có dữ liệu đã lưu, tự động điền vào
  useEffect(() => {
    if (rememberedEmail) {
      setEmail(rememberedEmail);
    }
    if (rememberedPassword) {
      setPassword(rememberedPassword);
    }
  }, [rememberedEmail, rememberedPassword]);

  const handleLogin = async () => {
    const session = await login(email, password);
    if (session) {
      await saveOrClearData(email, password, isRememberActive);
      onLoginSuccess(session);
    }
  };

  return (
    <View style={styles.form}>
      {/* Email Input */}
      <View style={styles.inputGroup}>
        <Text style={styles.label}>Email nhân sự</Text>
        <View style={styles.inputContainer}>
          <Mail size={20} color="#64748B" />
          <TextInput
            style={styles.input}
            placeholder="nhanvien@unihub.vn"
            placeholderTextColor="#94A3B8"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            keyboardType="email-address"
          />
        </View>
      </View>

      {/* Password Input */}
      <View style={styles.inputGroup}>
        <Text style={styles.label}>Mật khẩu</Text>
        <View style={styles.inputContainer}>
          <Lock size={20} color="#64748B" />
          <TextInput
            style={styles.input}
            placeholder="••••••••"
            placeholderTextColor="#94A3B8"
            value={password}
            onChangeText={setPassword}
            secureTextEntry={!showPassword}
          />
          <TouchableOpacity onPress={() => setShowPassword(!showPassword)}>
            {showPassword ? <EyeOff size={20} color="#64748B" /> : <Eye size={20} color="#64748B" />}
          </TouchableOpacity>
        </View>
      </View>

      {/* Options Row: Remember Me & Forgot Password */}
      <View style={styles.optionsRow}>
        <TouchableOpacity 
          style={styles.rememberMe} 
          onPress={() => setIsRememberActive(!isRememberActive)}
        >
          {isRememberActive ? (
            <CheckSquare size={20} color="#312E81" />
          ) : (
            <Square size={20} color="#94A3B8" />
          )}
          <Text style={[styles.rememberText, isRememberActive && styles.activeText]}>Ghi nhớ tôi</Text>
        </TouchableOpacity>

        <TouchableOpacity>
          <Text style={styles.forgotPassText}>Quên mật khẩu?</Text>
        </TouchableOpacity>
      </View>

      {/* Login Button */}
      <TouchableOpacity
        onPress={handleLogin}
        disabled={loading}
        style={[styles.button, loading && styles.buttonDisabled]}
      >
        {loading ? (
          <ActivityIndicator color="white" />
        ) : (
          <Text style={styles.buttonText}>Đăng nhập ngay</Text>
        )}
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  form: {
    width: '100%',
    padding: 16,
  },
  inputGroup: {
    marginBottom: 20,
  },
  label: {
    fontSize: 14,
    fontWeight: '500',
    color: '#334155',
    marginBottom: 8,
    marginLeft: 4,
  },
  inputContainer: {
    height: 56,
    backgroundColor: 'white',
    borderWidth: 1,
    borderColor: '#E2E8F0',
    borderRadius: 12,
    paddingHorizontal: 16,
    flexDirection: 'row',
    alignItems: 'center',
  },
  input: {
    flex: 1,
    marginLeft: 12,
    fontSize: 16,
    color: '#0F172A',
  },
  optionsRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: 4,
  },
  rememberMe: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  rememberText: {
    marginLeft: 8,
    fontSize: 14,
    color: '#64748B',
  },
  activeText: {
    color: '#312E81',
    fontWeight: '500',
  },
  forgotPassText: {
    color: '#312E81',
    fontWeight: '500',
    fontSize: 14,
  },
  button: {
    height: 56,
    backgroundColor: '#312E81',
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 32,
    elevation: 4,
    shadowColor: '#312E81',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.2,
    shadowRadius: 4,
  },
  buttonDisabled: {
    backgroundColor: '#C7D2FE',
  },
  buttonText: {
    color: 'white',
    fontSize: 18,
    fontWeight: 'bold',
  },
});
