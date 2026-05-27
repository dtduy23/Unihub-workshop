# UniHub Workshop — Go Backend

## Kiến trúc tổng quan

Backend API cho hệ thống UniHub Workshop, xây dựng bằng Go với kiến trúc hỗn hợp:

- **Layered Architecture**: Handler → Service → Repository
- **Event-Driven**: RabbitMQ + Background Workers cho đăng ký workshop
- **Pipe-and-Filter**: AI Summary pipeline (Extract → Clean → Prompt → AI → Persist)
- **Batch Sequential**: CSV Import pipeline (Extract → Transform → Load → Cleanup)

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go 1.22+ |
| HTTP Router | chi |
| Database | PostgreSQL 16 (pgx driver) |
| Cache | Redis 7 |
| Message Broker | RabbitMQ 3 |
| Auth | JWT (HS256) |
| QR Code | go-qrcode |
| Email | SMTP (MailHog for dev) |
| Container | Docker Compose |

## Khởi chạy nhanh

### 1. Khởi động infrastructure

```bash
cd src/backend
docker compose up -d
```

Chờ tất cả services healthy (~10-15s), hệ thống tự động chạy schema migration và seed data.

### 2. Chạy backend server

```bash
go run cmd/server/main.go
```

Server khởi động tại `http://localhost:8080`

### 3. Kiểm tra health

```bash
curl http://localhost:8080/health
```

## Tài khoản mẫu

| Student ID | Password | Role | Mô tả |
|---|---|---|---|
| `admin` | `123456` | ADMIN | System Administrator |
| `staff01` | `123456` | STAFF | Trần Staff |
| `21127001` | `123456` | STUDENT | Nguyễn Văn An |
| `21127002` | `123456` | STUDENT | Trần Thị Bình |

## API Endpoints

### Auth
```
POST   /api/v1/auth/login          # Đăng nhập → JWT token
GET    /api/v1/auth/me              # Thông tin user (Auth)
```

### Workshop (Public)
```
GET    /api/v1/workshops            # Danh sách workshop
GET    /api/v1/workshops/{id}       # Chi tiết workshop
```

### Registration (Student)
```
POST   /api/v1/registrations                     # Đăng ký → HTTP 202 + Correlation ID
GET    /api/v1/registrations/status/{correlationId} # Polling trạng thái
GET    /api/v1/registrations/my                   # Danh sách đã đăng ký
```

### Payment (Student)
```
POST   /api/v1/payments/{registrationId}  # Khởi tạo thanh toán
POST   /api/v1/payment/webhook            # Webhook callback (public)
```

### Check-in (Staff)
```
POST   /api/v1/checkin/live         # Check-in online
POST   /api/v1/checkin/sync         # Đồng bộ offline (bulk)
```

### Admin (Organizer)
```
POST   /api/v1/workshops            # Tạo workshop
PUT    /api/v1/workshops/{id}       # Sửa workshop
DELETE /api/v1/workshops/{id}       # Hủy workshop
POST   /api/v1/admin/import/csv     # Upload CSV import
GET    /api/v1/admin/import/jobs    # Lịch sử import
POST   /api/v1/admin/workshops/{workshopId}/summary  # Upload PDF → AI Summary
GET    /api/v1/admin/stats          # Thống kê hệ thống
GET    /api/v1/admin/payment/circuit-breaker  # Trạng thái Circuit Breaker
```

### Notification
```
GET    /api/v1/notifications        # Lịch sử thông báo
```

## Ví dụ sử dụng

### Đăng nhập
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"student_id":"21127001","password":"password123"}'
```

### Xem danh sách workshop
```bash
curl http://localhost:8080/api/v1/workshops
```

### Đăng ký workshop (cần JWT)
```bash
TOKEN="<jwt_token_from_login>"
curl -X POST http://localhost:8080/api/v1/registrations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"workshop_id":"<workshop_id>"}'
```

### Import CSV (Organizer)
```bash
TOKEN="<admin_jwt_token>"
curl -X POST http://localhost:8080/api/v1/admin/import/csv \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@data/sample_students.csv"
```

## Cơ chế kỹ thuật đã triển khai

### 1. Rate Limiting (Token Bucket)
- Redis Lua Script đảm bảo atomicity
- Fallback sang In-memory khi Redis down
- Response 429 + Retry-After header

### 2. Circuit Breaker (Payment Gateway)
- 3 trạng thái: CLOSED → OPEN → HALF-OPEN
- Ngưỡng: 50% error rate trong 10s
- Sleep window: 30s
- Graceful Degradation khi mạch mở

### 3. Idempotency (Chống trừ tiền 2 lần)
- Redis SETNX với TTL 24h
- Key format: `payment:idempotency:{transaction_id}`

### 4. Pessimistic Locking (Tranh chấp chỗ ngồi)
- `SELECT ... FOR UPDATE` trong PostgreSQL
- Background Worker xử lý tuần tự từ Message Queue

### 5. Event-Driven Registration
- Request → RabbitMQ → Worker → DB
- HTTP 202 Accepted + Correlation ID polling

### 6. Offline Check-in Sync
- Bulk sync API cho mobile app
- Conflict resolution: earliest timestamp wins

### 7. Notification (Strategy + Observer)
- Email Strategy (SMTP)
- Web Notification Strategy (DB-backed)
- Idempotent delivery: UNIQUE(event_id, channel)

### 8. Batch Import (CSV)
- Batch Sequential: Extract → Transform → Load → Cleanup
- Chunk processing (1000 records/chunk)
- Error tracking per row

### 9. AI Summary (Pipe-and-Filter)
- PDF → Extract → Clean → Prompt → AI API → DB
- Circuit Breaker cho AI service
- Mock response khi không có API key

## Monitoring

- **MailHog UI**: http://localhost:8025 (xem email đã gửi)
- **RabbitMQ UI**: http://localhost:15672 (guest/guest)
- **Health Check**: http://localhost:8080/health

## Cấu trúc thư mục

```
backend/
├── cmd/server/main.go           # Entry point + routing + workers
├── internal/
│   ├── config/                  # Environment config
│   ├── database/                # PostgreSQL + Redis connections
│   ├── middleware/              # Auth, RBAC, Rate Limit, CORS
│   ├── model/                   # Domain models + DTOs
│   ├── handler/                 # HTTP handlers
│   ├── service/                 # Business logic
│   ├── repository/              # Data access layer
│   ├── queue/                   # RabbitMQ publisher/consumer
│   ├── circuitbreaker/          # Circuit Breaker pattern
│   └── ratelimiter/             # Token Bucket (Redis + local)
├── migrations/                  # SQL seed data
├── data/                        # Sample CSV
├── docker-compose.yml           # Infrastructure
└── .env                         # Configuration
```
