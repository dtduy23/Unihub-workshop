# 🎓 ĐỒ ÁN MÔN HỌC – UniHub Workshop

Hệ thống quản lý, đăng ký và check-in sự kiện (Tuần lễ kỹ năng và nghề nghiệp) tải trọng cao dành cho sinh viên và ban tổ chức.

---

## 🌟 Tính năng nghiệp vụ đầy đủ
- **Sinh viên:** Xem danh sách workshop, sơ đồ phòng, trạng thái ghế trống (thời gian thực). Đăng ký (miễn phí/trả phí) và nhận mã QR qua email.
- **Ban tổ chức (Admin):** Quản lý sự kiện, tải lên PDF để AI tóm tắt nội dung tự động, tải lên file CSV để Import danh sách 12,000 sinh viên. Xem thống kê hệ thống.
- **Nhân sự Check-in (Staff):** Quét mã QR tại cửa hội trường bằng ứng dụng di động. Hỗ trợ ghi nhận **Offline (Mất mạng hoàn toàn)** và tự động đồng bộ khi có Internet trở lại.

## 🚀 Các cơ chế kỹ thuật nổi bật (Đã triển khai trong Code)
1. **Chống lố ghế (Tranh chấp chỗ ngồi):** Sử dụng `SELECT FOR UPDATE` (Pessimistic Locking) trong PostgreSQL đảm bảo giao dịch tuyệt đối an toàn kể cả khi hàng trăm thread gọi cùng lúc.
2. **Kiểm soát tải đột biến (Rate Limiting & Waiting Room):** Thuật toán **Token Bucket** trên Redis chống Spam API. Cùng với **Virtual Waiting Room** (Redis ZSET) đưa sinh viên vào phòng chờ khi quá tải để bảo vệ hệ thống.
3. **Event-Driven Architecture (RabbitMQ):** Bóc tách luồng Đăng ký và Thông báo (Web/Email) thành các Worker hoạt động độc lập nhằm phản hồi API đăng ký dưới 10ms.
4. **Xử lý thanh toán lỗi (Circuit Breaker & Idempotency):** Áp dụng khóa Luỹ đẳng chặn trừ tiền 2 lần. Bọc Circuit Breaker để "Ngắt mạch" (Fail Fast) ngay lập tức khi hệ thống thanh toán sập, giúp server Go không bị treo.
5. **Bảo mật mã QR bằng Chữ ký số RSA-2048:** App Mobile nhận Public Key để tự phân biệt vé thật/giả ngay cả khi không có kết nối Internet mạng.
6. **AI Summary Pipe-and-Filter:** Đẩy luồng xử lý PDF qua các filter trích xuất, làm sạch và gọi Google Gemini API để tóm tắt học thuật.
7. **Batch Import CSV hiệu năng cao:** Phân tách file 12,000 sinh viên thành các lô (Chunks) và ghi đè bằng `INSERT ... ON CONFLICT DO UPDATE` chỉ trong 1 truy vấn.

---


## ⚙️ Hướng dẫn cài đặt và khởi chạy

> **Yêu cầu môi trường:** Đã cài đặt `Docker & Docker Compose`, `Node.js (v18+)`, `Golang (v1.22+)`.

### Bước 1: Khởi động Hạ tầng (Cơ sở dữ liệu & Message Broker)
Mở Terminal, di chuyển vào thư mục backend và chạy Docker Compose:
```bash
cd src/backend
docker-compose up -d
```
Lệnh này sẽ khởi động:
- **PostgreSQL** (port `5433`)
- **Redis** (port `6379`)
- **RabbitMQ** (port `5672` và `15672`)
*(Schema Database sẽ tự động được chạy qua file `src/data/init_schema.sql`)*

### Bước 2: Cấu hình và Chạy Backend
Tạo file `.env` từ file mẫu:
```bash
cp .env.example .env
```
*(Bạn có thể điền thông tin SMTP Email hoặc Google Gemini API Key vào file `.env` nếu muốn test tính năng thực).*

Khởi chạy Backend (Port `8080`):
```bash
go mod tidy
go run cmd/server/main.go
```

### Bước 3: Cấu hình và Chạy Web Frontend
Mở một Terminal mới:
```bash
cd src/web
npm install
npm run dev
```
Trang Web Sinh Viên và Admin sẽ khởi chạy tại: `http://localhost:3000`

### Bước 4: Khởi chạy Mobile App (Nhân sự Check-in)
Mở một Terminal mới:
```bash
cd src/mobile
npm install
npx expo start --clear
```
Sử dụng ứng dụng **Expo Go** trên điện thoại (hoặc máy ảo Simulator) để quét mã QR hiện ra trên màn hình Terminal.

*(Lưu ý: Nếu test bằng điện thoại vật lý, hãy đảm bảo Backend đổi biến `.env` hoặc IP trong file `api.ts` để gọi API qua Local Area Network).*

---

## 🔑 Dữ liệu mẫu (Seed Data)

Database đã tự động được Seed các Admin ban đầu. Để đăng nhập hệ thống, vui lòng sử dụng:

**Tài khoản Admin:**
- Email / Username: `admin` (hoặc `admin@unihub.edu.vn`)
- Mật khẩu: `admin123`

**Test Tính Năng Sinh Viên:**
1. Truy cập trang Web Admin.
2. Vào mục **Sinh viên**, upload file `src/backend/data/sample_students_v2.csv`.
3. Hệ thống sẽ Import thành công. 
4. Đăng xuất, dùng Email sinh viên trong file (Mật khẩu mặc định: `123456`) để đăng nhập và đăng ký Workshop!