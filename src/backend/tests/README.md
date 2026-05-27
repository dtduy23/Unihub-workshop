# 🧪 UniHub Workshop — Load Testing Guide

Bộ công cụ mô phỏng **12,000 sinh viên** đăng ký workshop đồng thời, dùng để kiểm tra hiệu năng và độ tin cậy của hệ thống.

---

## 📁 Cấu trúc thư mục

```
tests/
├── README.md                 # File này
├── setup.sh                  # One-click: Docker + DB + Seed + Import 12k
├── load_test.sh              # Load test: Login + Registration storm
├── generate_12k_students.py  # Tạo 12,000 hồ sơ sinh viên CSV
└── direct_import_12k.py      # Import trực tiếp vào PostgreSQL (fast)
```

---

## 🚀 Quickstart (3 lệnh)

```bash
# 1. Khởi tạo toàn bộ (Docker + DB + 12k students)
./tests/setup.sh

# 2. Chạy server (mở terminal riêng)
go run cmd/server/main.go

# 3. Chạy load test
./tests/load_test.sh loadtest
```

---

## 📋 Chi tiết từng bước

### Step 1: Setup môi trường

```bash
./tests/setup.sh
```

Script này sẽ tự động:
1. **Start Docker Compose** (PostgreSQL, Redis, RabbitMQ, MailHog)
2. **Chờ PostgreSQL** sẵn sàng
3. **Kiểm tra schema** (tự thêm cột thiếu nếu cần)
4. **Seed dữ liệu** (admin, staff, 5 students, 4 workshops)
5. **Import 12,000 students** trực tiếp vào DB (~0.4s, ~27k records/s)

**Kết quả:**
```
📊 Database Summary:
   Users:     12,007 (12,005 students)
   Workshops: 4
   Password:  123456 (bcrypt hashed)

🔑 Accounts:
   Admin:   admin / 123456
   Staff:   staff01 / 123456
   Student: 21127001-21139099 / 123456
```

### Step 2: Chạy server

```bash
go run cmd/server/main.go
```

### Step 3: Chạy load test

```bash
# Chạy chỉ load test (login + registration)
./tests/load_test.sh loadtest

# Hoặc chạy tất cả phases (generate + import + loadtest)
./tests/load_test.sh all
```

**Các phases:**

| Phase | Lệnh | Mô tả |
|-------|-------|--------|
| `generate` | `./tests/load_test.sh generate` | Tạo CSV 12,000 sinh viên |
| `import` | `./tests/load_test.sh import` | Import CSV qua API |
| `loadtest` | `./tests/load_test.sh loadtest` | Login + đăng ký workshop |
| `report` | `./tests/load_test.sh report` | Xem lại báo cáo lần test gần nhất |
| `all` | `./tests/load_test.sh all` | Chạy tất cả |

---

## ⚙️ Tuỳ chỉnh

### Biến môi trường

```bash
# Giới hạn concurrent requests (default: 200)
CONCURRENCY=100 ./tests/load_test.sh loadtest

# Đổi URL server
BASE_URL=http://192.168.1.100:8080 ./tests/load_test.sh loadtest

# Đổi số students
TOTAL_STUDENTS=5000 ./tests/load_test.sh loadtest
```

### Mô phỏng server yếu (VPS 1 core)

```bash
# Giới hạn Go chỉ dùng 1 CPU core
GOMAXPROCS=1 go run cmd/server/main.go

# Giới hạn 2 cores
GOMAXPROCS=2 go run cmd/server/main.go

# Pin vào CPU core cụ thể + giới hạn goroutines
GOMAXPROCS=1 taskset -c 0 go run cmd/server/main.go
```

---

## 📊 Đọc kết quả

Sau khi load test chạy xong, bạn sẽ thấy báo cáo:

```
┌─────────────────────────────────────────────────────┐
│         📊 LOAD TEST RESULTS SUMMARY                │
├─────────────────────────────────────────────────────┤
│  Total Requests:          12000                     │
│  HTTP 202 (Queued):         100                     │
│  HTTP 429 (Rate):         11900                     │
├─────────────────────────────────────────────────────┤
│  Latency (avg):            76ms                     │
│  Latency (p50):            50ms                     │
│  Latency (p95):           247ms                     │
│  Latency (p99):           281ms                     │
│  Throughput:            36697 req/s                  │
└─────────────────────────────────────────────────────┘
```

### Ý nghĩa HTTP Status:

| Status | Ý nghĩa | Mong đợi |
|--------|----------|----------|
| **202** | Đăng ký thành công, queued vào RabbitMQ | ✅ Cao = tốt |
| **200** | Request xử lý đồng bộ thành công | ✅ OK |
| **400** | Workshop hết chỗ / đã đăng ký rồi | ⚠️ Bình thường khi hết seats |
| **429** | Rate limiter chặn (Token Bucket) | ⚠️ Đúng hành vi, tăng `RATE_LIMIT_CAPACITY` nếu cần |
| **500** | Server error | ❌ Cần debug |
| **000** | Timeout / Connection refused | ❌ Server quá tải |

### Tuning Rate Limiter

Nếu thấy **99% requests bị 429**, tăng capacity trong `.env`:

```env
RATE_LIMIT_CAPACITY=5000
RATE_LIMIT_REFILL_RATE=1000
```

---

## 🏗️ Kiến trúc hệ thống được test

```
Client (12,000 students)
   │
   ▼
┌──────────────┐
│ Rate Limiter │ ← Token Bucket (Redis)
│ (chặn spam)  │
└──────┬───────┘
       │ (chỉ N req/s đi qua)
       ▼
┌──────────────┐
│ Waiting Room │ ← Virtual Queue (Redis)
│ (xếp hàng)   │
└──────┬───────┘
       │ (FIFO ordering)
       ▼
┌──────────────┐
│ Seat Limiter │ ← Redis pre-check
│ (còn chỗ?)   │
└──────┬───────┘
       │ (nếu còn seats)
       ▼
┌──────────────┐
│  RabbitMQ    │ ← Async processing
│  (enqueue)   │
└──────┬───────┘
       │ (background worker)
       ▼
┌──────────────┐
│  PostgreSQL  │ ← SELECT ... FOR UPDATE
│ (final lock) │   Pessimistic Locking
└──────────────┘
```

---

## 🔄 Reset và chạy lại

```bash
# Xóa sạch Docker + data, rebuild từ đầu
docker compose down -v && ./tests/setup.sh

# Chỉ xóa kết quả test
rm -rf data/loadtest_results/
```
