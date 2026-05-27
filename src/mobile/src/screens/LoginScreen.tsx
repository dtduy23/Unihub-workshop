import React from 'react';
import { View, Text, SafeAreaView, KeyboardAvoidingView, Platform, ScrollView, StyleSheet } from 'react-native';
import { LoginForm } from '../components/LoginForm';
import { type UserSession } from '../hooks/useAuth';

export default function LoginScreen({ onLoginSuccess }: { onLoginSuccess: (s: UserSession) => void }) {
  return (
    <SafeAreaView style={styles.container}>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        style={styles.flex}
      >
        <ScrollView contentContainerStyle={styles.scrollContent}>
          <View style={styles.innerContainer}>

            {/* Branding Section */}
            <View style={styles.branding}>
              <View style={styles.logoBox}>
                <Text style={styles.logoText}>U</Text>
              </View>
              <Text style={styles.title}>UniHub Staff</Text>
              <Text style={styles.subtitle}>
                Đăng nhập để bắt đầu quét mã QR và hỗ trợ Workshop
              </Text>
            </View>

            {/* Form Section */}
            <View style={styles.formContainer}>
              <LoginForm onLoginSuccess={onLoginSuccess} />
            </View>

            {/* Support Footer */}
            <View style={styles.footer}>
              <Text style={styles.footerText}>Phiên bản 1.0.0 • © 2026 UniHub</Text>
            </View>

          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F8FAFC',
  },
  flex: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
  },
  innerContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 24,
    paddingVertical: 48,
  },
  branding: {
    alignItems: 'center',
    marginBottom: 48,
  },
  logoBox: {
    height: 80,
    width: 80,
    backgroundColor: '#312E81', // Midnight Indigo (Web Primary)
    borderRadius: 24,
    alignItems: 'center',
    justifyContent: 'center',
    elevation: 8,
    shadowColor: '#312E81',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.3,
    shadowRadius: 10,
  },
  logoText: {
    color: 'white',
    fontSize: 40,
    fontWeight: '800',
  },
  title: {
    fontSize: 30,
    fontWeight: 'bold',
    color: '#0F172A', // Slate 900
    marginTop: 24,
  },
  subtitle: {
    fontSize: 16,
    color: '#64748B', // Slate 500
    textAlign: 'center',
    marginTop: 8,
    paddingHorizontal: 16,
  },
  formContainer: {
    width: '100%',
    backgroundColor: 'rgba(255, 255, 255, 0.5)',
    borderRadius: 24,
    padding: 8,
  },
  footer: {
    marginTop: 48,
    alignItems: 'center',
  },
  footerText: {
    color: '#94A3B8', // Slate 400
    fontSize: 12,
  },
});
