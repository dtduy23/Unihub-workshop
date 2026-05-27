import * as Crypto from 'expo-crypto';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { KJUR, KEYUTIL } from 'jsrsasign';
import { Buffer } from 'buffer';

const PUBLIC_KEY_STORAGE_KEY = '@unihub_public_key';
// Thay đổi IP này cho khớp với IP máy tính chạy Go Backend
export const API_BASE_URL = 'http://192.168.7.117:8080';

/**
 * Tải và lưu trữ Public Key từ Backend.
 */
export async function syncPublicKey(): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/public-key`);
    const json = await response.json();
    
    if (json.success && json.data?.public_key) {
      await AsyncStorage.setItem(PUBLIC_KEY_STORAGE_KEY, json.data.public_key);
      console.log('✅ [Crypto] Public Key synced and stored');
      return true;
    }
    return false;
  } catch (error) {
    console.error('❌ [Crypto] Failed to sync public key:', error);
    return false;
  }
}

/**
 * Lấy Public Key từ bộ nhớ máy.
 */
export async function getStoredPublicKey(): Promise<string | null> {
  return await AsyncStorage.getItem(PUBLIC_KEY_STORAGE_KEY);
}

/**
 * Xác thực chữ ký của vé (RSA-SHA256).
 * Sử dụng thư viện jsrsasign để Verify chuẩn PKCS#1 v1.5.
 */
export async function verifyTicket(payload: {
  sid: string;
  uid: string;
  wid: string;
  sig: string;
}): Promise<boolean> {
  const { sid, uid, wid, sig } = payload;
  
  if (!sid || !uid || !wid || !sig) return false;

  let publicKeyPEM = await getStoredPublicKey();
  
  if (!publicKeyPEM) {
    console.warn('⚠️ [Crypto] No public key stored. Attempting to sync...');
    const synced = await syncPublicKey();
    if (synced) {
      publicKeyPEM = await getStoredPublicKey();
    }
  }

  if (!publicKeyPEM) return false;

  try {
    // 1. Tạo chuỗi dữ liệu gốc (phải khớp 100% với Backend)
    const rawData = `${sid}|${uid}|${wid}`;
    
    // 2. Chuyển đổi signature từ Base64 sang Hex (jsrsasign yêu cầu Hex)
    const sigHex = Buffer.from(sig, 'base64').toString('hex');

    // 3. Khởi tạo đối tượng Signature với thuật toán SHA256withRSA
    const signature = new KJUR.crypto.Signature({ alg: "SHA256withRSA" });
    
    // 4. Nạp Public Key (KEYUTIL tự động nhận diện định dạng PEM)
    const pubKeyObj = KEYUTIL.getKey(publicKeyPEM);
    signature.init(pubKeyObj);
    
    // 5. Cập nhật dữ liệu cần kiểm tra
    signature.updateString(rawData);
    
    // 6. Thực hiện Verify
    const isValid = signature.verify(sigHex);
    
    console.log(`🛡️ [Crypto] RSA Verification: ${isValid ? 'PASSED ✅' : 'FAILED ❌'}`);
    return isValid;
  } catch (error) {
    console.error('❌ [Crypto] RSA Verification Crash:', error);
    return false;
  }
}
