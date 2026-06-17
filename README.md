<p align="center">
  <img src="docs/images/hero_banner.png" alt="UniHub Workshop Banner" width="800"/>
</p>

<h1 align="center">🎓 UniHub Workshop</h1>
<p align="center">
  <strong>High-Concurrency Event Management Platform for Universities</strong>
</p>
<p align="center">
  Hệ thống quản lý, đăng ký và check-in sự kiện (Tuần lễ kỹ năng và nghề nghiệp) tải trọng cao dành cho sinh viên và ban tổ chức.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go"/>
  <img src="https://img.shields.io/badge/PostgreSQL-16-336791?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL"/>
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis"/>
  <img src="https://img.shields.io/badge/RabbitMQ-3-FF6600?style=for-the-badge&logo=rabbitmq&logoColor=white" alt="RabbitMQ"/>
  <img src="https://img.shields.io/badge/Next.js-15-000000?style=for-the-badge&logo=nextdotjs&logoColor=white" alt="Next.js"/>
  <img src="https://img.shields.io/badge/Expo-React_Native-000020?style=for-the-badge&logo=expo&logoColor=white" alt="Expo"/>
  <img src="https://img.shields.io/badge/GCP-Kubernetes-4285F4?style=for-the-badge&logo=googlecloud&logoColor=white" alt="GCP"/>
</p>

---

## 🌟 Tính năng nghiệp vụ

| Vai trò | Tính năng |
|---------|-----------|
| **Sinh viên** | Xem danh sách workshop, sơ đồ phòng, trạng thái ghế trống (thời gian thực). Đăng ký (miễn phí/trả phí) và nhận mã QR qua email. |
| **Ban tổ chức (Admin)** | Quản lý sự kiện, tải lên PDF để AI tóm tắt nội dung tự động, Import CSV 12,000 sinh viên. Xem thống kê hệ thống. |
| **Nhân sự Check-in (Staff)** | Quét mã QR tại cửa hội trường bằng ứng dụng di động. Hỗ trợ ghi nhận **Offline (Mất mạng hoàn toàn)** và tự động đồng bộ khi có Internet. |

---

## 📋 Yêu cầu Chức năng (Functional Requirements)

### FR-1: Xác thực và Phân quyền (Authentication & Authorization)

| ID | Mô tả | Actor |
|----|-------|-------|
| FR-1.1 | Hệ thống cho phép người dùng đăng nhập bằng **Mã số Sinh viên** (hoặc Email) và mật khẩu. Mật khẩu được mã hoá bằng **bcrypt**. | Tất cả |
| FR-1.2 | Sau khi đăng nhập, hệ thống trả về **JWT Token** chứa thông tin user ID và role. Token được sử dụng cho tất cả các API yêu cầu xác thực. | Tất cả |
| FR-1.3 | Hệ thống phân quyền 3 vai trò: **STUDENT**, **STAFF**, **ADMIN**. Middleware kiểm tra role trên từng nhóm endpoint. | Hệ thống |
| FR-1.4 | Sinh viên có thể **Quên mật khẩu**: hệ thống tự sinh mật khẩu ngẫu nhiên, cập nhật vào DB và gửi email thông báo mật khẩu mới. | Sinh viên |
| FR-1.5 | Người dùng đã đăng nhập có thể **Đổi mật khẩu** bằng cách cung cấp mật khẩu cũ và mật khẩu mới. | Tất cả |
| FR-1.6 | API **Get Public Key** cho phép ứng dụng Mobile tải về khoá công khai RSA-2048 để xác thực chữ ký vé offline. | Mobile App |

---

### FR-2: Quản lý Workshop (Workshop Management)

| ID | Mô tả | Actor |
|----|-------|-------|
| FR-2.1 | Sinh viên có thể **xem danh sách** tất cả Workshop đang `PUBLISHED`, hỗ trợ **tìm kiếm theo tiêu đề**. | Sinh viên |
| FR-2.2 | Sinh viên có thể **xem chi tiết** Workshop: tiêu đề, diễn giả, phòng, sơ đồ phòng (`room_layout_url`), thời gian, sức chứa, số ghế trống (real-time), giá vé, và tóm tắt nội dung (AI-generated). | Sinh viên |
| FR-2.3 | Admin có thể **tạo mới** Workshop: nhập tiêu đề, diễn giả, phòng, thời gian bắt đầu/kết thúc, thời gian mở/đóng đăng ký, sức chứa, giá vé (miễn phí hoặc trả phí), sơ đồ phòng. | Admin |
| FR-2.4 | Admin có thể **cập nhật** thông tin Workshop (partial update): thay đổi bất kỳ trường nào mà không ảnh hưởng các trường khác. | Admin |
| FR-2.5 | Admin có thể **huỷ** Workshop: trạng thái chuyển sang `DELETED`, Workshop không hiển thị cho sinh viên. | Admin |

---

### FR-3: Đăng ký Workshop (Registration)

| ID | Mô tả | Actor |
|----|-------|-------|
| FR-3.1 | Sinh viên gửi yêu cầu đăng ký → hệ thống kiểm tra **Virtual Waiting Room** (Redis ZSET). Nếu quá tải, sinh viên được xếp vào phòng chờ và nhận vị trí hàng đợi. | Sinh viên |
| FR-3.2 | Khi được cấp quyền vào, hệ thống kiểm tra **Redis Seat Lock** (kiểm tra nhanh số ghế trống trên cache) → nếu còn ghế, yêu cầu được đẩy vào **RabbitMQ** và trả về `HTTP 202 Accepted` cùng `correlation_id`. | Sinh viên |
| FR-3.3 | Background Worker nhận message → thực hiện **Pessimistic Locking** (`SELECT FOR UPDATE`) trên PostgreSQL → trừ ghế → tạo bản ghi Registration. Nếu workshop miễn phí → trạng thái `SUCCESS`; nếu có phí → trạng thái `PENDING_PAYMENT`. | Hệ thống |
| FR-3.4 | Với workshop miễn phí, hệ thống tự động **ký chữ ký số RSA-2048** lên vé (chứa mã SV + Workshop ID) và lưu `ticket_signature` vào DB. | Hệ thống |
| FR-3.5 | Sinh viên có thể **polling trạng thái** đăng ký qua `correlation_id` để biết kết quả xử lý (PROCESSING → SUCCESS / PENDING_PAYMENT / REJECTED / FAILED). | Sinh viên |
| FR-3.6 | Sinh viên có thể **xem danh sách** các Workshop đã đăng ký của mình (bao gồm tên workshop, phòng, thời gian). | Sinh viên |
| FR-3.7 | Sinh viên có thể **huỷ đăng ký** (chỉ khi trạng thái là `SUCCESS` hoặc `PENDING_PAYMENT`): hệ thống hoàn trả ghế vào cả PostgreSQL và Redis Cache. | Sinh viên |
| FR-3.8 | Admin có thể **xem danh sách sinh viên** đã đăng ký theo từng Workshop (bao gồm mã SV, họ tên, email, trạng thái, điểm danh). | Admin |
| FR-3.9 | Admin có thể **xuất CSV** danh sách sinh viên theo Workshop: hỗ trợ 2 loại — `registered` (tất cả đã đăng ký) và `attended` (chỉ những người đã check-in). File CSV có BOM UTF-8 để tương thích Excel. | Admin |

---

### FR-4: Thanh toán (Payment)

| ID | Mô tả | Actor |
|----|-------|-------|
| FR-4.1 | Khi đăng ký Workshop trả phí, hệ thống **tự động tạo giao dịch** với mã `transaction_id` duy nhất và trả về `checkout_url` cho sinh viên thanh toán. | Hệ thống |
| FR-4.2 | **Idempotency**: nếu sinh viên gọi lại API thanh toán cho cùng đăng ký, hệ thống trả về giao dịch `PENDING` đã tồn tại thay vì tạo mới (tránh payment rác). | Hệ thống |
| FR-4.3 | Cổng thanh toán gọi lại **Webhook** với `transaction_id`, `status` (SUCCESS/FAILED) và `signature`. Hệ thống xác thực HMAC-SHA256 trước khi xử lý. | Cổng TT |
| FR-4.4 | **Webhook Idempotency**: sử dụng Redis `SETNX` với TTL 24h để chặn xử lý trùng lặp (chống trừ tiền 2 lần). | Hệ thống |
| FR-4.5 | Thanh toán thành công → Registration chuyển sang `SUCCESS` → ký chữ ký RSA → gửi thông báo email xác nhận. | Hệ thống |
| FR-4.6 | Thanh toán thất bại → Registration chuyển sang `FAILED` → hoàn trả ghế vào DB và Redis. | Hệ thống |
| FR-4.7 | **Circuit Breaker**: khi cổng thanh toán liên tục lỗi (≥50% failure rate), mạch ngắt mở ra (Fail Fast) → sinh viên được thông báo ngay mà không phải chờ timeout. Mạch tự phục hồi sau 30 giây. | Hệ thống |
| FR-4.8 | **Payment Cleanup**: Job nền tự động xoá các đăng ký `PENDING_PAYMENT` quá 15 phút chưa thanh toán → hoàn trả ghế cho sinh viên khác. | Hệ thống |
| FR-4.9 | Admin có thể xem **trạng thái Circuit Breaker** (Closed/Open/HalfOpen) và trạng thái cổng thanh toán (Up/Down). | Admin |

---

### FR-5: Check-in (Điểm danh)

| ID | Mô tả | Actor |
|----|-------|-------|
| FR-5.1 | Staff quét mã QR bằng ứng dụng Mobile → gửi `registration_id`, `workshop_id`, `student_id` → hệ thống xác thực và đánh dấu `is_checked_in = true` (**Live Check-in**). | Staff |
| FR-5.2 | Khi **mất kết nối Internet**, ứng dụng Mobile lưu trữ các lần quét vào bộ nhớ cục bộ (Offline Queue). Khi có mạng trở lại, app tự động gửi **Bulk Sync** (tối đa 500 bản ghi/lần) để đồng bộ lên server. | Staff |
| FR-5.3 | App Mobile sử dụng **RSA Public Key** đã tải về để **xác thực chữ ký vé offline** (verify signature) — phát hiện vé giả mà không cần kết nối server. | Mobile App |

---

### FR-6: Thông báo (Notification)

| ID | Mô tả | Actor |
|----|-------|-------|
| FR-6.1 | Khi đăng ký Workshop miễn phí thành công, hệ thống tự động gửi **email xác nhận** (HTML template) đến sinh viên qua SMTP. | Hệ thống |
| FR-6.2 | Khi thanh toán Workshop trả phí thành công, hệ thống gửi **email xác nhận thanh toán** với nội dung riêng. | Hệ thống |
| FR-6.3 | Khi quên mật khẩu, hệ thống gửi **email chứa mật khẩu mới** cho sinh viên. | Hệ thống |
| FR-6.4 | Song song với email, mọi thông báo đều được lưu vào DB dưới kênh **Web** để sinh viên xem lịch sử thông báo trên giao diện. | Hệ thống |
| FR-6.5 | **Idempotent Notification**: mỗi event có `event_id` duy nhất → nếu DB đã tồn tại notification với `event_id` đó, hệ thống bỏ qua (không gửi trùng). | Hệ thống |
| FR-6.6 | Notification Worker chạy độc lập, nhận message từ **RabbitMQ** (`notification_queue`) và xử lý gửi email bất đồng bộ — không ảnh hưởng latency của API chính. | Hệ thống |

---

### FR-7: Tóm tắt AI (AI Summary)

| ID | Mô tả | Actor |
|----|-------|-------|
| FR-7.1 | Admin upload file **PDF** (tối đa 10MB) mô tả nội dung Workshop → hệ thống trích xuất text, làm sạch, gọi **Google Gemini API** và trả về bản tóm tắt học thuật. | Admin |
| FR-7.2 | Tóm tắt AI có thể được gắn vào Workshop cụ thể (`workshop_id`) để sinh viên xem trên trang chi tiết Workshop. | Admin |
| FR-7.3 | Hệ thống kiểm tra **magic bytes** (`%PDF`) để đảm bảo file upload thực sự là PDF trước khi xử lý. | Hệ thống |

---

### FR-8: Quản trị Hệ thống (System Administration)

| ID | Mô tả | Actor |
|----|-------|-------|
| FR-8.1 | Admin upload file **CSV** chứa danh sách sinh viên (lên đến 12,000 records) → hệ thống lưu file và tạo **Import Job** với trạng thái `PENDING`. | Admin |
| FR-8.2 | Import Job có thể chạy tự động lúc 02:00 AM hoặc được **trigger thủ công** bởi Admin. Hệ thống phân tách CSV thành **Chunks** và ghi bằng `INSERT ... ON CONFLICT DO UPDATE` (upsert). | Admin |
| FR-8.3 | Hệ thống theo dõi tiến trình Import: tổng số bản ghi, số thành công, số lỗi. Các dòng lỗi được ghi chi tiết vào bảng `import_errors` (số dòng, raw data, lý do lỗi). | Admin |
| FR-8.4 | Admin có thể xem **lịch sử Import Jobs** và **chi tiết lỗi** của từng job. | Admin |
| FR-8.5 | Admin có thể xem **Dashboard thống kê hệ thống**: tổng số sinh viên, tổng workshop, tổng đăng ký, v.v. | Admin |

---

## 🏗️ Kiến trúc Backend (System Architecture)

<p align="center">
  <img src="docs/images/backend_architecture.png" alt="Backend Architecture" width="800"/>
</p>

### Các cơ chế kỹ thuật nổi bật

| # | Cơ chế | Mô tả |
|---|--------|-------|
| 1 | **Pessimistic Locking** | `SELECT FOR UPDATE` trong PostgreSQL đảm bảo giao dịch an toàn khi hàng trăm thread tranh chấp chỗ ngồi. |
| 2 | **Rate Limiting & Waiting Room** | Thuật toán **Token Bucket** trên Redis chống Spam API. **Virtual Waiting Room** (Redis ZSET) đưa sinh viên vào phòng chờ khi quá tải. |
| 3 | **Event-Driven Architecture** | RabbitMQ bóc tách luồng Đăng ký và Thông báo thành Worker độc lập, phản hồi API dưới 10ms. |
| 4 | **Circuit Breaker & Idempotency** | Khóa Luỹ đẳng chặn trừ tiền 2 lần. Circuit Breaker "Ngắt mạch" (Fail Fast) khi hệ thống thanh toán sập. |
| 5 | **RSA-2048 Digital Signature** | App Mobile nhận Public Key để tự verify vé thật/giả ngay cả khi Offline. |
| 6 | **AI Pipe-and-Filter** | Trích xuất PDF → Làm sạch → Gọi Google Gemini API để tóm tắt học thuật. |
| 7 | **Batch Import CSV** | Phân tách 12,000 sinh viên thành Chunks, ghi đè bằng `INSERT ... ON CONFLICT DO UPDATE`. |

---

## ☁️ Hạ tầng DevOps trên GCP (Production Deployment)

<p align="center">
  <img src="docs/images/gcp_infrastructure.png" alt="GCP Infrastructure & CI/CD" width="800"/>
</p>

| Thành phần | Công nghệ |
|------------|-----------|
| **Compute** | GKE Autopilot (Horizontal Pod Autoscaler: 2→20 pods) |
| **Database** | Cloud SQL for PostgreSQL (Private IP, Automated Backup) |
| **Cache** | Memorystore for Redis (Private IP) |
| **Message Queue** | RabbitMQ trên GKE (Helm Chart) |
| **Networking** | VPC, Cloud DNS, Global HTTP(S) Load Balancer, Cloud Armor WAF, Cloud NAT |
| **CI/CD** | Jenkins Pipeline → Build → Test → Docker Image → Artifact Registry → Rolling Deploy |
| **Monitoring** | Prometheus + Grafana, Cloud Logging, Alerting |
| **Security** | Google Secret Manager, Network Policy, Non-root containers |
| **IaC** | Terraform (Infrastructure as Code) |

---

## 📁 Cấu trúc Dự án

```
unihub-workshop/
├── src/
│   ├── backend/          # Go API Server + Background Workers
│   │   ├── cmd/server/   # Entrypoint (main.go)
│   │   ├── internal/     # Handler, Service, Repository, Middleware, Config
│   │   ├── migrations/   # Database schema & seed data
│   │   └── Dockerfile
│   ├── web/              # Next.js Frontend (Admin + Student)
│   └── mobile/           # React Native (Expo) - Staff Check-in App
├── deploy/               # DevOps & Infrastructure
│   ├── terraform/        # GCP Infrastructure as Code
│   ├── k8s/              # Kubernetes manifests
│   ├── jenkins/          # CI/CD pipeline (Jenkinsfile)
│   └── monitoring/       # Grafana + Prometheus config
├── docs/images/          # Ảnh minh hoạ cho README
└── blueprint/            # Design docs & specs
```

---

## ⚙️ Hướng dẫn cài đặt và khởi chạy (Local Development)

> **Yêu cầu môi trường:** `Docker & Docker Compose`, `Node.js (v18+)`, `Golang (v1.22+)`.

### Bước 1: Khởi động Hạ tầng (Database & Message Broker)

```bash
cd src/backend
docker-compose up -d
```

Lệnh này sẽ khởi động:
- **PostgreSQL** (port `5433`) — Schema Database tự động chạy qua `init_schema.sql`
- **Redis** (port `6379`)
- **RabbitMQ** (port `5672` / Management UI: `15672`)

### Bước 2: Chạy Backend

```bash
cp .env.example .env    # Tạo file cấu hình
go mod tidy
go run cmd/server/main.go
```

Backend sẽ chạy tại: `http://localhost:8080`

### Bước 3: Chạy Web Frontend

```bash
cd src/web
npm install
npm run dev
```

Trang Web sẽ khởi chạy tại: `http://localhost:3000`

### Bước 4: Chạy Mobile App (Staff Check-in)

```bash
cd src/mobile
npm install
npx expo start --clear
```

Sử dụng ứng dụng **Expo Go** trên điện thoại để quét mã QR trên Terminal.

> **Lưu ý:** Nếu test bằng điện thoại vật lý, hãy đổi IP trong file `api.ts` để gọi API qua Local Area Network.

---

## 🔑 Dữ liệu mẫu (Seed Data)

**Tài khoản Admin:**
| Field | Value |
|-------|-------|
| Username | `admin` (hoặc `admin@unihub.edu.vn`) |
| Password | `admin123` |

**Test Tính Năng Sinh Viên:**
1. Đăng nhập Admin → Vào mục **Sinh viên** → Upload file `src/backend/data/sample_students_v2.csv`.
2. Đăng xuất → Dùng Email sinh viên trong file (Mật khẩu: `123456`) để đăng nhập và đăng ký Workshop.