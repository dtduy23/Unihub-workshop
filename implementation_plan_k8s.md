# Giai đoạn 3: Container hoá + Deploy lên GKE + Go Migration

> **Phạm vi:** Sửa code backend cho horizontal scaling, tối ưu Dockerfile (BE + FE), loại bỏ Supabase, viết Go-based database migration, viết toàn bộ Kubernetes manifests, cấu hình RabbitMQ Helm.

---

## Tổng quan các thay đổi

```mermaid
graph TD
    subgraph "Code Changes"
        A1["config.go + main.go<br/>APP_MODE (api/worker/all)"]
        A2["[NEW] migrator.go<br/>Go auto-migration"]
        A3["[DELETE] Supabase code<br/>Chuyển sang Go Backend API"]
        A4["next.config.mjs<br/>+ output: standalone"]
    end

    subgraph "Dockerfiles"
        B1["src/backend/Dockerfile<br/>Multi-stage + non-root"]
        B2["[NEW] src/web/Dockerfile<br/>Next.js standalone"]
    end

    subgraph "Kubernetes (deploy/k8s/)"
        C1["namespace.yaml"]
        C2["configmap.yaml"]
        C3["secret.yaml"]
        C4["serviceaccount.yaml"]
        C5["deployment-api.yaml"]
        C6["deployment-worker.yaml"]
        C7["deployment-web.yaml"]
        C8["service.yaml"]
        C9["ingress.yaml"]
        C10["hpa.yaml"]
        C11["rabbitmq/values.yaml"]
    end

    A1 & A2 --> B1
    A3 & A4 --> B2
    B1 --> C5 & C6
    B2 --> C7
```

---

## Proposed Changes

### Component 1: Backend Code — APP_MODE (Horizontal Scaling)

#### [MODIFY] [config.go](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/backend/internal/config/config.go)

Thêm field `AppMode` vào struct `Config`:

```diff
 type Config struct {
+    // App Mode: "api" | "worker" | "all" (default cho local dev)
+    AppMode string
+
     // Database
     DBHost     string
```

Thêm dòng load trong `Load()`:

```diff
 return &Config{
+    AppMode:    getEnv("APP_MODE", "all"),
     DBHost:     getEnv("DB_HOST", "localhost"),
```

#### [MODIFY] [main.go](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/backend/cmd/server/main.go)

Refactor để điều kiện hoá theo `cfg.AppMode`:

```diff
-    // Start background workers
-    ctx, cancel := context.WithCancel(context.Background())
-    defer cancel()
-    go startRegistrationWorker(ctx, consumer, regService)
-    go startNotificationWorker(ctx, cfg.RabbitMQURL, notifService)
-    go startPaymentCleanupWorker(ctx, paymentService)
-    go startBatchImportScheduler(ctx, batchService)
-
-    // Start server
-    srv := &http.Server{...}

+    ctx, cancel := context.WithCancel(context.Background())
+    defer cancel()
+
+    mode := cfg.AppMode
+    log.Printf("[SERVER] Running in mode: %s", mode)
+
+    // Start background workers (chỉ khi mode = "worker" hoặc "all")
+    if mode == "worker" || mode == "all" {
+        go startRegistrationWorker(ctx, consumer, regService)
+        go startNotificationWorker(ctx, cfg.RabbitMQURL, notifService)
+        go startPaymentCleanupWorker(ctx, paymentService)
+        go startBatchImportScheduler(ctx, batchService)
+    }
+
+    // Start HTTP server (chỉ khi mode = "api" hoặc "all")
+    if mode == "api" || mode == "all" {
+        srv := &http.Server{...}
+        // ... giữ nguyên logic listen/serve/graceful shutdown
+    } else {
+        // Worker-only mode: block cho đến khi nhận tín hiệu tắt
+        sigCh := make(chan os.Signal, 1)
+        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
+        <-sigCh
+        cancel()
+    }
```

> [!NOTE]
> Khi chạy local (`APP_MODE=all` — mặc định), mọi thứ hoạt động y hệt hiện tại.

---

### Component 2: Go Database Migration (Thay thế SQL thủ công)

#### [NEW] [migrator.go](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/backend/internal/database/migrator.go)

Migration runner sử dụng bảng `schema_migrations` để tracking:

- Tạo bảng `schema_migrations` (version + applied_at) nếu chưa tồn tại
- Duyệt qua từng migration file (embedded trong binary bằng `go:embed`), chạy theo thứ tự version
- Mỗi migration chạy trong 1 transaction — rollback nếu lỗi
- Idempotent: migration đã chạy rồi thì skip

#### [NEW] [migrations/](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/backend/internal/database/migrations/) (thư mục chứa SQL embed)

| File | Nội dung |
|------|----------|
| `001_init_schema.sql` | Copy từ [init_schema.sql](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/data/init_schema.sql) (7 bảng + indexes) |
| `002_seed_data.sql` | Copy từ [002_seed_data.sql](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/backend/migrations/002_seed_data.sql) (users + workshops mẫu). Chỉ chạy khi biến `RUN_SEED=true` |

#### [MODIFY] [main.go](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/backend/cmd/server/main.go)

Gọi migration sau khi kết nối DB:

```diff
     pgPool := database.NewPostgresPool(cfg)
     defer pgPool.Close()
+
+    // Auto-run database migrations
+    if err := database.RunMigrations(pgPool); err != nil {
+        log.Fatalf("[MIGRATION] Failed: %v", err)
+    }
```

---

### Component 3: Loại bỏ Supabase — Chuyển sang Go Backend API

> [!IMPORTANT]
> Frontend hiện dùng Supabase ở 2 chỗ. Cả hai đều cần chuyển sang gọi Go Backend API.

#### [DELETE] [utils/supabase/middleware.ts](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/web/utils/supabase/middleware.ts)

File middleware Supabase cũ. Đã bị thay thế bởi [middleware.ts](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/web/middleware.ts) mới (dùng cookie `unihub_token` từ Go Backend). **Xoá hoàn toàn.**

#### [DELETE] [lib/supabase.ts](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/web/lib/supabase.ts)

Supabase client init. **Xoá hoàn toàn.**

#### [MODIFY] [app/api/checkin/sync/route.ts](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/web/app/api/checkin/sync/route.ts)

Hiện tại file này gọi **trực tiếp vào Supabase DB** để lookup user UUID và update checkin. Cần chuyển sang gọi **Go Backend API** (`POST /api/v1/checkin/sync` đã có sẵn):

```diff
-import { supabase } from '@/lib/supabase';
+const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

 export async function POST(request: Request) {
-  // ... gọi supabase trực tiếp
+  const records = await request.json();
+  const token = request.headers.get('cookie')?.match(/unihub_token=([^;]*)/)?.[1];
+
+  // Forward tới Go Backend API
+  const res = await fetch(`${API_URL}/api/v1/checkin/sync`, {
+    method: 'POST',
+    headers: {
+      'Content-Type': 'application/json',
+      'Authorization': `Bearer ${token}`,
+    },
+    body: JSON.stringify(records),
+  });
+  return new Response(res.body, { status: res.status, headers: res.headers });
```

#### [MODIFY] [.env.local](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/web/.env.local)

Xoá biến Supabase:

```diff
-NEXT_PUBLIC_SUPABASE_URL=https://hmgrbkzdstqcfhznvrqd.supabase.co
-NEXT_PUBLIC_SUPABASE_ANON_KEY=eyJ...
-
 # Go Backend API
 NEXT_PUBLIC_API_URL=http://localhost:8080
```

#### [MODIFY] [package.json](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/web/package.json)

Gỡ dependencies Supabase:

```diff
-    "@supabase/ssr": "^0.10.3",
-    "@supabase/supabase-js": "^2.105.4",
```

---

### Component 4: Tối ưu Dockerfile Backend

#### [MODIFY] [src/backend/Dockerfile](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/backend/Dockerfile)

| Cải tiến | Mô tả |
|----------|-------|
| `-ldflags="-w -s"` | Strip debug symbols, giảm ~30% kích thước binary |
| Non-root user (`appuser`) | Chạy ứng dụng không dùng quyền root |
| Bỏ `COPY .env.example .env` | Trên K8s, biến môi trường truyền qua ConfigMap/Secret |
| `ENTRYPOINT` thay `CMD` | Nhận tín hiệu SIGTERM từ K8s chính xác hơn |

#### [NEW] [src/backend/.dockerignore](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/backend/.dockerignore)

Loại trừ `.git`, `.env`, `tests/`, `docker-compose.yml` khỏi build context.

---

### Component 5: Dockerfile Frontend (Next.js Standalone)

#### [MODIFY] [next.config.mjs](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/web/next.config.mjs)

```diff
 const nextConfig = {
+  output: "standalone",
   typescript: {
     ignoreBuildErrors: true,
   },
```

> `output: "standalone"` khiến `next build` tạo ra thư mục `.next/standalone/` chứa server Node.js tối giản, không cần `node_modules` → Docker image nhẹ hơn 10 lần.

#### [NEW] [src/web/Dockerfile](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/web/Dockerfile)

Multi-stage build cho Next.js:

```dockerfile
# Stage 1: Build
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Stage 2: Runtime
FROM node:20-alpine AS runner
WORKDIR /app
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy standalone output + static files
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

USER appuser
EXPOSE 3000
ENV PORT=3000 HOSTNAME="0.0.0.0"
CMD ["node", "server.js"]
```

#### [NEW] [src/web/.dockerignore](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/src/web/.dockerignore)

Loại trừ `node_modules`, `.next`, `.env.local` khỏi build context.

---

### Component 6: Kubernetes Manifests (deploy/k8s/)

#### [NEW] `namespace.yaml`

Tạo namespace `unihub` để cô lập tài nguyên.

---

#### [NEW] `serviceaccount.yaml`

Tạo 2 K8s Service Accounts liên kết Workload Identity → GCP SA (đã tạo trong [iam.tf](file:///home/tuna/learn/se/UniHub-Workshop/unihub-workshop/deploy/terraform/iam.tf)):

| K8s SA | GCP SA | Namespace |
|--------|--------|-----------|
| `unihub-api` | `unihub-workload@project.iam` | `unihub` |
| `unihub-worker` | `unihub-workload@project.iam` | `unihub` |

---

#### [NEW] `configmap.yaml`

Biến môi trường **không nhạy cảm** (giá trị lấy từ `terraform output`):

| Key | Value (template) |
|-----|---------|
| `DB_HOST` | `<cloudsql_private_ip>` |
| `DB_PORT` | `5432` |
| `DB_NAME` | `unihub_workshop` |
| `DB_USER` | `unihub` |
| `DB_SSLMODE` | `require` |
| `REDIS_ADDR` | `<redis_host>:<redis_port>` |
| `RABBITMQ_URL` | `amqp://user:pass@rabbitmq.unihub.svc.cluster.local:5672/` |
| `SERVER_PORT` | `8080` |
| `CORS_ORIGINS` | `https://yourdomain.dev` |
| `NEXT_PUBLIC_API_URL` | `https://api.yourdomain.dev` (cho FE) |
| `RUN_SEED` | `true` (lần đầu), chuyển `false` sau khi seed xong |

---

#### [NEW] `secret.yaml`

K8s Secret dạng Opaque chứa các giá trị nhạy cảm (base64-encoded). Giá trị lấy từ `terraform output` hoặc Secret Manager:

| Key | Source |
|-----|--------|
| `DB_PASSWORD` | `terraform output -raw` hoặc Secret Manager |
| `AUTH_SECRET` | JWT secret |
| `RSA_PRIVATE_KEY` | RSA key cho vé QR |
| `PAYMENT_WEBHOOK_SECRET` | Payment webhook |
| `SMTP_USER`, `SMTP_PASS` | Email credentials |
| `GEMINI_API_KEY` | Google AI key |

---

#### [NEW] `deployment-api.yaml` — Backend API Server

| Config | Giá trị |
|--------|---------|
| `APP_MODE` | `api` |
| `replicas` | `2` |
| Image | `asia-southeast1-docker.pkg.dev/<project>/unihub-repo/backend:latest` |
| Readiness Probe | `GET /health`, port 8080, initialDelay 5s |
| Liveness Probe | `GET /health`, port 8080, initialDelay 15s |
| Resources | requests: 200m CPU, 256Mi RAM / limits: 500m CPU, 512Mi RAM |
| Service Account | `unihub-api` |

---

#### [NEW] `deployment-worker.yaml` — Background Worker

| Config | Giá trị |
|--------|---------|
| `APP_MODE` | `worker` |
| `replicas` | `1` (cố định) |
| Resources | requests: 200m CPU, 256Mi RAM / limits: 500m CPU, 512Mi RAM |
| Service Account | `unihub-worker` |

---

#### [NEW] `deployment-web.yaml` — Frontend Next.js

| Config | Giá trị |
|--------|---------|
| `replicas` | `2` |
| Image | `asia-southeast1-docker.pkg.dev/<project>/unihub-repo/web:latest` |
| Port | `3000` |
| Env | `NEXT_PUBLIC_API_URL` (từ ConfigMap) |
| Readiness Probe | `GET /`, port 3000, initialDelay 5s |
| Resources | requests: 100m CPU, 128Mi RAM / limits: 300m CPU, 256Mi RAM |

---

#### [NEW] `service.yaml`

2 ClusterIP Services:

| Service | Selector | Port → TargetPort |
|---------|----------|-------------------|
| `unihub-api-service` | `app: unihub-api` | 80 → 8080 |
| `unihub-web-service` | `app: unihub-web` | 80 → 3000 |

---

#### [NEW] `ingress.yaml`

GKE Ingress routing (Google Cloud Load Balancer):

| Rule | Host | Path | Backend Service |
|------|------|------|----------------|
| API | `api.yourdomain.dev` | `/*` | `unihub-api-service:80` |
| Web | `yourdomain.dev` | `/*` | `unihub-web-service:80` |

> [!WARNING]
> Nếu chưa mua domain, Ingress sẽ dùng path-based routing trên 1 IP: `/api/*` → Backend, `/*` → Frontend (không cần domain).

---

#### [NEW] `hpa.yaml` — Auto Scale API Pods

| Config | Giá trị |
|--------|---------|
| Target | Deployment `unihub-api` |
| Min replicas | `2` |
| Max replicas | `20` |
| Metric | CPU > 70% → scale up |

---

#### [NEW] `rabbitmq/values.yaml`

Helm values cho Bitnami RabbitMQ:
- 1 replica (tiết kiệm chi phí)
- Persistence: 8Gi PVC
- username/password cấu hình qua values

---

## Tổng kết File Changes

### Files cần sửa (MODIFY): 7 files

| File | Thay đổi |
|------|----------|
| `src/backend/internal/config/config.go` | + `AppMode` field |
| `src/backend/cmd/server/main.go` | + APP_MODE logic + migration call |
| `src/backend/Dockerfile` | Tối ưu multi-stage, non-root, strip binary |
| `src/web/next.config.mjs` | + `output: "standalone"` |
| `src/web/.env.local` | Xoá Supabase vars |
| `src/web/package.json` | Xoá `@supabase/*` deps |
| `src/web/app/api/checkin/sync/route.ts` | Chuyển từ Supabase → Go Backend API |

### Files cần tạo mới (NEW): 15 files

| File | Mô tả |
|------|-------|
| `src/backend/internal/database/migrator.go` | Go migration runner |
| `src/backend/internal/database/migrations/001_init_schema.sql` | Schema (embed) |
| `src/backend/internal/database/migrations/002_seed_data.sql` | Seed data (embed) |
| `src/backend/.dockerignore` | Docker build exclusions |
| `src/web/Dockerfile` | Next.js standalone build |
| `src/web/.dockerignore` | Docker build exclusions |
| `deploy/k8s/namespace.yaml` | K8s namespace |
| `deploy/k8s/serviceaccount.yaml` | Workload Identity SAs |
| `deploy/k8s/configmap.yaml` | Non-sensitive env vars |
| `deploy/k8s/secret.yaml` | Sensitive env vars |
| `deploy/k8s/deployment-api.yaml` | Backend API pods |
| `deploy/k8s/deployment-worker.yaml` | Background worker pod |
| `deploy/k8s/deployment-web.yaml` | Frontend Next.js pods |
| `deploy/k8s/service.yaml` | ClusterIP services |
| `deploy/k8s/ingress.yaml` | GKE Ingress + LB |
| `deploy/k8s/hpa.yaml` | HPA for API pods |
| `deploy/k8s/rabbitmq/values.yaml` | RabbitMQ Helm values |

### Files cần xoá (DELETE): 2 files

| File | Lý do |
|------|-------|
| `src/web/utils/supabase/middleware.ts` | Thay bằng middleware.ts mới (đã có) |
| `src/web/lib/supabase.ts` | Không dùng Supabase nữa |

---

## Open Questions

> [!IMPORTANT]
> **1. Domain name?**
> `terraform.tfvars` hiện có `domain_name = ""`. Bạn đã mua domain chưa?
> - **Có domain:** Ingress sẽ dùng host-based routing (`api.domain.dev` + `domain.dev`) + SSL tự động.
> - **Chưa có:** Ingress dùng path-based routing trên 1 IP (`/api/*` → Backend, `/*` → FE), không SSL.

> [!IMPORTANT]
> **2. Seed data trên GKE?**
> File `002_seed_data.sql` chứa user test (password 123456) và workshop mẫu. Tôi sẽ thêm biến `RUN_SEED=true/false` để kiểm soát. Bạn muốn seed khi deploy lần đầu lên GKE không?

---

## Verification Plan

### Build Tests
1. `docker build -t unihub-api:test src/backend/` — verify Backend Dockerfile
2. `docker build -t unihub-web:test src/web/` — verify Frontend Dockerfile
3. Chạy container `APP_MODE=api` → verify chỉ HTTP server start
4. Chạy container với DB mới trắng → verify auto-migration tạo schema

### K8s Validation
1. `kubectl apply --dry-run=client -f deploy/k8s/` — verify YAML syntax
2. `kubectl get pods -n unihub` → verify 2 API + 1 Worker + 2 Web pods Running
3. `curl /health` → verify `{"status":"ok"}`

### Supabase Removal
1. Grep toàn bộ `src/web/` không còn chữ "supabase"
2. FE login + checkin hoạt động bình thường qua Go Backend API
