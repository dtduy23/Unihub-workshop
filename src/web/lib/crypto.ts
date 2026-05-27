import nacl from 'tweetnacl';
import { decodeUTF8, encodeBase64 } from 'tweetnacl-util';

// CHÚ Ý: Trong thực tế, Private Key phải được lưu ở biến môi trường (.env)
// Khóa này đã được chuẩn hóa đúng 64 byte cho Ed25519
const MOCK_PRIVATE_KEY_BASE64 = '5V1veJbk6S1k/ifwrMQ/bpESC0u7oMgZhnX0K4HaLvAsh6c9kJd5a4OwS8nS9xQvwjzvh2wS1GRP6eS6SJAsUg==';
const privateKey = Buffer.from(MOCK_PRIVATE_KEY_BASE64, 'base64');

/**
 * Tạo chữ ký số cho thông tin vé
 * @param studentId MSSV
 * @param workshopId ID của Workshop
 * @returns Chuỗi signature định dạng Base64
 */
export function signTicket(studentId: string, workshopId: string): string {
  const data = `${studentId}:${workshopId}`;
  const msg = decodeUTF8(data);
  const signature = nacl.sign.detached(msg, privateKey);
  return encodeBase64(signature);
}

/**
 * Lấy Public Key từ Private Key hiện tại để gửi cho Client
 */
export function getPublicKey(): string {
  const keyPair = nacl.sign.keyPair.fromSecretKey(privateKey);
  return encodeBase64(keyPair.publicKey);
}
