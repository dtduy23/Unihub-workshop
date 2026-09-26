# UniHub Helm Charts

This directory contains the production-grade Helm Chart packages for the UniHub Platform ecosystem.

## Chart Architecture: `unihub`

The `unihub` chart packages the entire multi-tier architecture:
- **Backend API Server**: High-throughput Go REST API with HPA autoscaling, liveness/readiness probes, and Prometheus metrics endpoint.
- **Background Worker**: Asynchronous batch import and email notification worker.
- **Frontend Web**: Next.js user interface.
- **In-cluster Infrastructure (Dev/Local)**: PostgreSQL, Redis, and RabbitMQ (can be toggled on/off via `values.yaml`).
- **Observability**: Google Managed Prometheus / Prometheus Operator `PodMonitoring`.

```text
deploy/helm/
└── unihub/
    ├── .helmignore
    ├── Chart.yaml              # Chart metadata (v0.1.0, appVersion: 1.0.0)
    ├── values.yaml             # Default configuration (Local / KinD / Minikube)
    ├── values-staging.yaml     # Staging overrides (Let's Encrypt TLS, NGINX Ingress)
    ├── values-prod.yaml        # Production overrides (Cloud SQL, Memorystore, GCE Ingress)
    └── templates/
        ├── _helpers.tpl        # Common naming and label definitions
        ├── serviceaccount.yaml # Service accounts with GCP Workload Identity support
        ├── configmap.yaml      # Non-sensitive runtime environment variables
        ├── secret.yaml         # Sensitive runtime credentials
        ├── deployment-api.yaml # Backend API workload
        ├── service-api.yaml    # Backend Service (NodePort / ClusterIP)
        ├── hpa-api.yaml        # Horizontal Pod Autoscaler (CPU 70%)
        ├── deployment-worker.yaml # Async Worker workload
        ├── deployment-web.yaml # Frontend Web workload
        ├── service-web.yaml    # Frontend Web Service
        ├── ingress.yaml        # Path-based and Host-based Ingress routing
        ├── podmonitor.yaml     # Google Managed Prometheus scraping CRD
        ├── NOTES.txt           # Post-installation status and connection guide
        └── infra/              # Dev/Local in-cluster datastores
            ├── postgres.yaml
            ├── redis.yaml
            └── rabbitmq.yaml
```

---

## Quick Start & Usage

### 1. Lint the Chart
Validate template syntax and conventions:
```bash
# Validate default values
helm lint deploy/helm/unihub

# Validate staging values
helm lint deploy/helm/unihub -f deploy/helm/unihub/values-staging.yaml

# Validate production values
helm lint deploy/helm/unihub -f deploy/helm/unihub/values-prod.yaml
```

### 2. Render Manifests (Dry Run)
Preview the generated Kubernetes manifests without connecting to a cluster:
```bash
helm template unihub deploy/helm/unihub
```

### 3. Deploy to Cluster

#### Local Development / Minikube:
```bash
helm upgrade --install unihub deploy/helm/unihub \
  --namespace unihub \
  --create-namespace
```

#### Staging Environment:
```bash
helm upgrade --install unihub-staging deploy/helm/unihub \
  --namespace unihub-staging \
  --create-namespace \
  -f deploy/helm/unihub/values-staging.yaml
```

#### Production Environment (GKE with Cloud SQL & Memorystore):
```bash
helm upgrade --install unihub-prod deploy/helm/unihub \
  --namespace unihub \
  --create-namespace \
  -f deploy/helm/unihub/values-prod.yaml
```

---

## Configuration Reference

| Parameter | Description | Default (Local) | Production (`values-prod.yaml`) |
| :--- | :--- | :--- | :--- |
| `global.environment` | Target environment name | `development` | `production` |
| `api.replicaCount` | Replicas for API (when HPA disabled) | `2` | `4` |
| `api.autoscaling.enabled` | Enable Horizontal Pod Autoscaler | `true` | `true` (min 4, max 20) |
| `api.service.type` | Service exposure type | `NodePort` (`30080`) | `ClusterIP` |
| `postgres.enabled` | Deploy in-cluster PostgreSQL | `true` | `false` (uses Cloud SQL) |
| `redis.enabled` | Deploy in-cluster Redis | `true` | `false` (uses Memorystore) |
| `rabbitmq.enabled` | Deploy in-cluster RabbitMQ StatefulSet | `true` | `false` (uses external cluster) |
| `ingress.enabled` | Enable Kubernetes Ingress | `false` | `true` (`unihub.edu.vn`) |
| `podMonitor.enabled` | Enable Prometheus scraping CRD | `true` | `true` |
