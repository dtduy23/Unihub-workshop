# 📱 UniHub Scanner - Mobile App

Ứng dụng quét mã QR điểm danh chuyên dụng cho nhân sự (Staff) của UniHub Workshop. Ứng dụng hỗ trợ quét mã Offline và đồng bộ dữ liệu siêu tốc lên hệ thống quản lý.

## 🚀 Hướng dẫn cài đặt nhanh cho Team

Để chạy được ứng dụng này trên điện thoại cá nhân (Android/iOS), hãy làm theo các bước sau:

### 1. Chuẩn bị môi trường
*   Cài đặt **Node.js** (Phiên bản LTS mới nhất).
*   Điện thoại đã cài sẵn ứng dụng **Expo Go** (Tải trên App Store hoặc Google Play).
*   Đảm bảo **Điện thoại** và **Máy tính** của bạn đang kết nối cùng một mạng **Wi-Fi**.

### 2. Cài đặt Dependencies
Mở terminal tại thư mục `src/mobile` và chạy lệnh:
```bash
npm install
```

### 3. Cấu hình kết nối Backend (Quan trọng)
Bạn cần trỏ App về đúng địa chỉ IP máy tính đang chạy Backend của bạn:
1.  Mở file: `src/services/crypto.ts`.
2.  Tìm dòng: `export const API_BASE_URL = 'http://192.168.x.x:3000';`.
3.  Thay `192.168.x.x` bằng địa chỉ IP máy tính của bạn (Dùng lệnh `ipconfig` trên Windows hoặc `ifconfig` trên Mac để xem).

### 4. Chạy ứng dụng
Tại thư mục `src/mobile`, chạy lệnh:
```bash
npx expo start
```
*   Một mã **QR Code** sẽ hiện ra trên terminal.
*   Dùng ứng dụng **Expo Go** trên điện thoại để quét mã này.
*   Đợi một lát để App bundle và khởi chạy.

---

## 🛠 Các tính năng chính
*   **Quét mã siêu tốc:** Tự động nhận diện vé hợp lệ cho từng Workshop.
*   **Chế độ Offline:** Vẫn có thể điểm danh khi mất mạng, dữ liệu sẽ lưu vào SQLite nội bộ.
*   **Đồng bộ thông minh:** Tự động đẩy dữ liệu lên Server khi có mạng trở lại.
*   **Midnight Indigo Theme:** Giao diện tối giản, hiện đại đồng bộ với phiên bản Web.

## 📁 Cấu trúc thư mục (Dành cho Developer)
*   `src/screens/`: Giao diện các màn hình (Login, Home, Scanner, Sync).
*   `src/hooks/`: Các logic xử lý dữ liệu (useWorkshops, useSync, useAuth).
*   `src/services/`: Dịch vụ lưu trữ (SQLite) và kết nối API.
*   `src/components/`: Các thành phần giao diện dùng chung.

---
**Lưu ý:** Nếu không đăng nhập được, hãy đảm bảo Backend (Web) đang được chạy bằng lệnh `npm run dev:host` tại thư mục `src/web`.
