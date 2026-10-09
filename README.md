# UniHub Workshop

A workshop registration and community platform for students, event staff, businesses, and administrators. UniHub combines a Go API, asynchronous workers, a Next.js web application, and an Expo app for QR check-in.

The repository includes Kubernetes packaging, GitOps configuration, CI quality gates, and observability artifacts. Local validation results are distinguished from deployment work that still needs testing on a running cluster.

![UniHub Workshop](docs/images/hero_banner.png)

## Contents

- [Features](#features)
- [Architecture](#architecture)
- [Technology](#technology)
- [Local development](#local-development)
- [Configuration](#configuration)
- [Tests and validation](#tests-and-validation)
- [Kubernetes demo](#kubernetes-demo)
- [CI and GitOps](#ci-and-gitops)
- [Repository layout](#repository-layout)

## Features

| Role | Main capabilities |
| --- | --- |
| Student | Browse workshops, register and view QR tickets, publish community posts, comment, bookmark posts, and follow companies. |
| Business | Maintain a company profile, create workshop drafts, submit workshops and revisions for admin review, and publish announcements linked to approved workshops. |
| Staff | Check in attendees through web or mobile, verify signed tickets offline, and synchronize scans when connectivity returns. |
| Admin | Manage workshops, approve businesses and workshop submissions, moderate reported content, assign staff, and import student accounts. |

### Community and business

- Authenticated posts with up to four images, likes, comments with one reply level, bookmarks, company follows, and follower notifications.
- A chronological feed with cursor pagination, topic/company filters, and following, saved, and author views.
- Role-based access and ownership checks across company profiles, workshops, announcements, and media.
- Company approval/suspension, workshop reviews and revisions, content reports, and moderation audit records.
- HttpOnly JWT sessions through a same-origin web API proxy, expiring single-use password reset links, and session revocation after password changes.

### Registration and check-in

- PostgreSQL reserves a seat under a workshop row lock before enqueueing registration. Repeated pending requests reuse a correlation ID.
- Registration and notification outboxes retry unpublished events after broker/publisher failures. Registration state is persisted in PostgreSQL and scoped to its requesting user.
- Redis provides rate limiting, waiting-room coordination, caching, and approximate visitor counts through HyperLogLog.
- QR tickets carry an RSA signature over the student, user, and workshop identifiers. Staff clients cache the public key for offline verification and synchronize scans in batches of up to 500.
- CSV import processes accounts in batches; the sample-data generator supports up to 12,000 rows. An optional Gemini integration extracts workshop information from uploaded documents.

## Architecture

```mermaid
flowchart LR
    Web[Next.js web] -->|Session proxy| API[Go API]
    Mobile[Expo check-in app] -->|JWT| API
    API -->|Seat reservation and outbox| DB[(PostgreSQL)]
    API -->|Admission and cache| Redis[(Redis)]
    API -->|Registration events| MQ[RabbitMQ]
    MQ --> Worker[Go workers]
    Worker -->|Finalize tickets and status| DB
    Worker --> SMTP[SMTP]
    API -->|Metrics| Prometheus[Prometheus / Grafana]
    Worker -->|Metrics| Prometheus
```

The backend uses handler, service, and repository layers. `APP_MODE` selects API, worker, combined, or migration-only execution. Worker mode exposes health, readiness, and metrics without business API routes. Migrations use a PostgreSQL advisory lock to serialize concurrent startup.

The API commits a reservation before returning a correlation ID. Workers finalize the registration and ticket; clients poll until completion. PostgreSQL is authoritative for available seats and registration state.

## Technology

| Area | Implementation |
| --- | --- |
| Backend | Go 1.25.5, Chi, pgx, JWT, RSA signatures |
| Web | Next.js 16.2.4, React 19, TypeScript |
| Mobile | Expo SDK 54, React Native, SQLite for pending scans |
| Data and messaging | PostgreSQL 16, Redis 7, RabbitMQ |
| Containers and deployment | Docker Compose, Nginx, Kubernetes, Helm |
| Delivery | GitHub Actions, Argo Workflows, ArgoCD |
| Scaling and telemetry | HPA, KEDA, Prometheus, Grafana, Fluent Bit, OpenSearch |
| Testing | Go race/integration tests, Python regression tests, k6, Helm validation |

## Local development

Install Docker with the Compose v2 plugin, Go 1.25.5 or newer, Node.js 22, npm, Make, and OpenSSL. Commands start from the repository root unless a working directory is shown.

### 1. Start infrastructure

```bash
docker compose -f src/backend/docker-compose.yml up -d
docker compose -f src/backend/docker-compose.yml ps
```

This starts PostgreSQL on `5433`, Redis on `6379`, RabbitMQ on `5672`, and MailHog on `1025`. Wait for PostgreSQL, Redis, and RabbitMQ to become healthy before starting the backend.

### 2. Start the backend

Generate a persistent local signing key once so issued tickets remain verifiable after a restart:

```bash
mkdir -p .devops/local
umask 077
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out .devops/local/tickets.pem
```

Keep the following environment in the backend terminal:

```bash
export APP_MODE=all
export DB_HOST=localhost DB_PORT=5433
export DB_USER=unihub DB_PASSWORD=unihub_secret DB_NAME=unihub_workshop
export DB_SSLMODE=disable
export REDIS_ADDR=localhost:6379
export RABBITMQ_URL=amqp://guest:guest@localhost:5672/
export SMTP_HOST=localhost SMTP_PORT=1025 SMTP_USER= SMTP_PASS=
export AUTH_SECRET="$(openssl rand -hex 32)"
export RSA_PRIVATE_KEY="$(cat .devops/local/tickets.pem)"
export SERVER_PORT=8080 WEB_URL=http://localhost:3000
export CORS_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
export RUN_SEED=true

cd src/backend
go run ./cmd/server
```

The server loads `.env` from its working directory, with exported variables taking precedence. See [the backend environment template](src/backend/.env.example) for optional settings. Keep `AUTH_SECRET` stable across restarts to retain sessions; API/worker instances in one environment must share the auth secret and signing key. `RUN_SEED=true` is for a local demo database.

### 3. Start the web application

In another terminal:

```bash
cp src/web/.env.example src/web/.env.local
cd src/web
npm ci
npm run dev
```

The web app uses `API_INTERNAL_URL=http://localhost:8080` to forward authenticated requests to the backend.

| Service | Local address |
| --- | --- |
| Web | http://localhost:3000 |
| API health / readiness | http://localhost:8080/health / http://localhost:8080/ready |
| API metrics | http://localhost:8080/metrics |
| RabbitMQ management | http://localhost:15672 |
| MailHog inbox | http://localhost:8025 |

Seed accounts use password `123456`: admin `admin`, staff `staff01`, and student `21127001`. An admin creates business accounts at `/admin/companies` and approves them before publication. These accounts and infrastructure credentials are for local development.

### 4. Start the mobile application

```bash
cp src/mobile/.env.example src/mobile/.env
cd src/mobile
npm ci
npm start
```

Set `EXPO_PUBLIC_API_URL` to `http://10.0.2.2:8080` for an Android emulator, or the backend computer's LAN address for a physical device. Staff must authenticate and fetch the signing public key while online before offline verification.

### Other local profiles

`make dev` starts the root Compose profile: eight API replicas, two workers, two web replicas, and an Nginx gateway, alongside infrastructure. It requires a Compose version supporting `include`. Supply a shared `AUTH_SECRET` and `RSA_PRIVATE_KEY` through an ignored `docker-compose.override.yml`, and size `DB_MAX_CONNS`/`DB_MIN_CONNS` against the database connection budget before load testing.

```bash
make help
make dev
make dev-ps
make dev-logs
make dev-down
```

Run this profile and component-based development separately because their published ports overlap.

Generate a CSV for the admin import screen:

```bash
python3 scripts/generate_12k_students.py --count 12000 --output /tmp/unihub-students.csv
```

## Configuration

Templates are tracked; local environment files, keys, generated data, and test reports are ignored.

| Variable | Purpose |
| --- | --- |
| `APP_MODE` | `api`, `worker`, `all`, or `migrate`. |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | PostgreSQL connection. |
| `DB_MAX_CONNS`, `DB_MIN_CONNS` | Per-process connection pool limits. |
| `REDIS_ADDR`, `REDIS_PASSWORD` | Redis connection. |
| `RABBITMQ_URL` | Message broker connection. |
| `AUTH_SECRET`, `RSA_PRIVATE_KEY` | JWT sessions and RSA ticket signing. |
| `AUTO_MIGRATE`, `RUN_SEED` | Migration and demo seed controls. Kubernetes runs migrations in a separate Job. |
| `WEB_URL`, `CORS_ORIGINS`, `MEDIA_DIR` | Reset links, browser origins, and uploaded media. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `SMTP_USER`, `SMTP_PASS` | Notification and password reset email. |
| `GEMINI_API_KEY`, `GEMINI_MODEL` | Optional document extraction service. |
| `API_INTERNAL_URL` | Web server's backend URL. |
| `EXPO_PUBLIC_API_URL` | Mobile backend URL. |

## Tests and validation

### Application checks

```bash
make test
python3 -m unittest discover -s scripts/tests -v
```

Go integration tests require `TEST_DATABASE_URL` to name a dedicated database ending in `_test`. Tests modify that database. Without the variable, database integration tests are skipped.

With local PostgreSQL running, create the test database once and run the full suite:

```bash
docker compose -f src/backend/docker-compose.yml exec -T postgres createdb -U unihub unihub_test
TEST_DATABASE_URL='postgres://unihub:unihub_secret@localhost:5433/unihub_test?sslmode=disable' make test
```

Integration coverage includes concurrent migrations, community/business permissions, registration and seat consistency, check-in, workshop revisions, and session revocation.

After `npm ci`, run these checks in the indicated directories:

```bash
# src/web
npm run typecheck
npm run lint
npm run build

# src/mobile
npm run typecheck
```

### DevOps checks and performance gate

`make devops-check` requires Bash, Helm, kubectl, Python, and PyYAML. It lints/renders Helm profiles, validates resource relationships and CI dependencies, checks shell syntax, and runs Python tests. ShellCheck, Argo CLI, and kubeconform add checks when installed. Offline validation does not exercise controllers on a cluster.

```bash
make devops-check
```

The [k6 scenario](src/backend/tests/k6_load_test.js) browses with 20 virtual users, then submits 50 registrations. It requires p95 tagged API latency below 200 ms, HTTP errors below 0.5%, every check to pass, and every registration to reach `SUCCESS`.

Recorded local validation on **9 October 2026** used PostgreSQL 16, Redis 7, and RabbitMQ 3.13:

| Check | Recorded result |
| --- | --- |
| Go integration/race suite | 39 tests/subtests passed; optional preview-server test skipped. |
| k6 workload | 20 VUs, 50 completed registrations, 8,846 HTTP requests/checks. |
| Tagged API p95 latency | 3.71 ms. |
| HTTP errors / failed checks | 0% / 0. |
| Database result | 50 `SUCCESS` registrations on a 50-seat workshop; zero seats remaining. |
| Failure cases | Invalid JWT failed the gate; broker loss made worker health return 503. |

These results describe that local workload. Earlier 2,000-user benchmarks predate the current reservation and community changes; current capacity still needs a new load test. Full ArgoCD/Workflows bootstrap, image publication/promotion, autoscaling, log ingestion, and production deployment have not been validated end-to-end on a live cluster.

## Kubernetes demo

The bootstrap requires Docker, kubectl, Helm, Git, OpenSSL, Python with PyYAML, and either kind or minikube. It requires a clean committed checkout, builds images for the current commit, and configures ArgoCD against a Git snapshot served inside the demo cluster.

```bash
make devops-demo-kind
# Alternatively, minikube requests 4 CPUs and 8 GiB RAM.
make devops-demo
```

Choose one driver. The script installs ArgoCD, Argo Workflows, monitoring, KEDA, and logging, deploys the app, and runs the authenticated k6 gate. Generated credentials and kubeconfig stay local.

After bootstrap succeeds, use the demo kubeconfig and run each port-forward in a separate terminal:

```bash
export KUBECONFIG="$PWD/.devops/kubeconfig"
kubectl -n unihub-demo port-forward svc/unihub-web-service 3000:80
kubectl -n unihub-demo port-forward svc/unihub-api-service 8080:80
kubectl -n argocd port-forward svc/argocd-server 8443:443
kubectl -n argo port-forward svc/argo-workflows-server 2746:2746
kubectl -n monitoring port-forward svc/monitoring-grafana 3001:80
kubectl -n logging port-forward svc/opensearch-dashboards 5601:5601
```

The app runs at http://localhost:3000, ArgoCD at https://localhost:8443, Workflows at https://localhost:2746, Grafana at http://localhost:3001, and OpenSearch Dashboards at http://localhost:5601. Retrieve controller/dashboard credentials from Kubernetes Secrets. The local OpenSearch demo disables security and is intended for access through local port-forwarding.

## CI and GitOps

```text
Commit SHA -> parallel Go/Python/web/manifest checks -> backend/web image builds
           -> isolated candidate release -> authenticated k6 gate
           -> optional staging image promotion in Git -> ArgoCD sync
```

- [GitHub Actions](.github/workflows/devops-ci.yaml) runs PR checks with PostgreSQL. Trusted main-branch runs submit Argo CI through a self-hosted runner labeled `unihub-ci`; configure `CONTAINER_REGISTRY` before enabling submission.
- [Argo Workflows](deploy/argo-workflows/ci-pipeline.yaml) builds images tagged with the full SHA, deploys a candidate to `unihub-ci`, checks async completion with k6, and cleans up on exit. Promotion rejects a source SHA that is no longer the head of `main`.
- [ArgoCD](deploy/argocd/) auto-syncs and self-heals staging. Production sync is manual. Helm image overlays separate staging, production, and demo releases.
- [Helm](deploy/helm/unihub/) packages API, worker, web, migrations, Services, ingress, storage, and optional datastores. Sync waves order configuration, infrastructure, migrations, applications, and routing/scaling.
- API scaling uses CPU/memory HPA or optional KEDA/Prometheus. Worker KEDA reads RabbitMQ backlog. The chart avoids two API autoscalers targeting the same Deployment.
- Prometheus PodMonitors, Grafana dashboards, and alerts cover latency, errors, visitors, queues, and worker availability. Fluent Bit/OpenSearch manifests collect structured logs; correlation IDs connect registration enqueue and worker processing.

Submit a workflow after configuring the cluster:

```bash
IMAGE_REGISTRY=registry.example.com/team/unihub make devops-ci
```

Replace the example registry. The cluster needs the Workflows controller, `unihub-ci` namespace/RBAC and WorkflowTemplate, registry credentials, and Git credentials if promotion is enabled.

Before staging/production sync, create the application Secret outside Git and configure images, ingress/TLS, web URL, SMTP, and storage. API replicas share media/import data: those profiles require RWX storage, and production expects an existing `unihub-media` claim. In-cluster datastores use single-node templates with persistence; backup/restore and high availability require environment-specific work. Legacy Jenkins and raw Kubernetes artifacts are retained alongside the primary Helm/Argo path.

## Repository layout

```text
.github/workflows/          GitHub CI entry point
deploy/
  argo-workflows/           CI DAG, controller values and RBAC
  argocd/                  Projects, applications and infrastructure bootstrap
  helm/unihub/             Chart, environment values and image overlays
  docker/                  Compose load-testing profile and Nginx
  logging/                 Fluent Bit, OpenSearch and retention manifests
  observability/metrics/   Grafana dashboard and Prometheus alerts
scripts/
  devops/                  Validation, demo, candidate, load-test and promotion tools
  dev/                     Backend startup helpers
  benchmark/               Concurrency benchmark runner
  tests/                   Python tooling regression tests
src/
  backend/                 Go API/workers, migrations and integration tests
  web/                     Next.js application and authenticated API proxy
  mobile/                  Expo staff check-in application
docs/images/               Project illustrations
```

`README.md` is the tracked documentation entry point. Local guides, plans, specifications, and nested README files are ignored.

Author: [Đinh Tuấn Duy](https://github.com/dtduy23).
