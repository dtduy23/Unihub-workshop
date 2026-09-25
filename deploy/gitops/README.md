# UniHub GitOps manifests (ArgoCD)

Thư mục quản lý vòng đời ứng dụng theo nguyên lý GitOps bằng ArgoCD.

## Các thành phần:
* `app-of-apps.yaml`: Pattern App-of-Apps quản trị toàn bộ cụm hạ tầng và dịch vụ.
* `application-staging.yaml`: Manifest ArgoCD tự động đồng bộ (Auto-sync) trên Staging.
* `application-prod.yaml`: Manifest ArgoCD cho Production (Manual Sync + Sync Windows).
* **Sync Waves**:
  * Wave 1: Databases & Broker (PostgreSQL, Redis, RabbitMQ).
  * Wave 2: DB Migrations Job.
  * Wave 3: Backend Services (API & Worker).
  * Wave 4: Ingress & HPA.
