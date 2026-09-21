#!/bin/bash
# unihub-deploy.sh — Clean staged Kubernetes demo deploy for UniHub Workshop
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K8S_DIR="$SCRIPT_DIR/deploy/k8s"

echo ""
echo "=============================================="
echo "  UniHub Workshop — Kubernetes Demo Deploy    "
echo "=============================================="
echo ""

# ─── Step 1: Enable ingress addon ───────────────
echo ">>> [1/6] Enabling minikube ingress addon..."
minikube addons enable ingress
minikube addons enable metrics-server
echo "    Done."

# ─── Step 2: Namespace ──────────────────────────
echo ">>> [2/6] Creating namespace..."
kubectl apply -f "$K8S_DIR/namespace.yaml"

# ─── Step 3: Secret ─────────────────────────────
echo ">>> [3/6] Creating secrets..."
kubectl create secret generic unihub-secret \
  --namespace=unihub \
  --from-literal=DB_PASSWORD=unihub_pass \
  --from-literal=JWT_SECRET=demo-jwt-secret-change-in-prod \
  --from-literal=GEMINI_API_KEY=demo-key \
  --dry-run=client -o yaml | kubectl apply -f -

# ─── Step 4: ConfigMap ──────────────────────────
echo ">>> [4/6] Applying ConfigMap..."
kubectl apply -f "$K8S_DIR/configmap.yaml"

# ─── Step 5: Infrastructure (PostgreSQL, Redis, RabbitMQ) ───
echo ">>> [5/6] Deploying infrastructure..."
kubectl apply -f "$K8S_DIR/postgres.yaml"
kubectl apply -f "$K8S_DIR/redis.yaml"
kubectl apply -f "$K8S_DIR/rabbitmq/rabbitmq.yaml"

echo "    Waiting for PostgreSQL..."
kubectl wait --for=condition=ready pod -l app=postgres -n unihub --timeout=120s
echo "    Waiting for Redis..."
kubectl wait --for=condition=ready pod -l app=redis -n unihub --timeout=60s
echo "    Waiting for RabbitMQ..."
kubectl wait --for=condition=ready pod -l app=rabbitmq -n unihub --timeout=120s
echo "    Infrastructure ready!"

# ─── Step 6: Application ────────────────────────
echo ">>> [6/6] Deploying UniHub application..."
kubectl apply -f "$K8S_DIR/serviceaccount.yaml"
kubectl apply -f "$K8S_DIR/deployment-api.yaml"
kubectl apply -f "$K8S_DIR/deployment-worker.yaml"
kubectl apply -f "$K8S_DIR/deployment-web.yaml"
kubectl apply -f "$K8S_DIR/service.yaml"
kubectl apply -f "$K8S_DIR/hpa.yaml"

echo "    Waiting for API pods to be ready..."
kubectl wait --for=condition=ready pod -l app=unihub-api -n unihub --timeout=300s
echo "    Waiting for Web pods to be ready..."
kubectl wait --for=condition=ready pod -l app=unihub-web -n unihub --timeout=300s
echo "    Application ready!"

# ─── Access Info ────────────────────────────────
MINIKUBE_IP=$(minikube ip)
echo ""
echo "=============================================="
echo "  ✅  DEPLOYMENT COMPLETE!"
echo "=============================================="
echo ""
echo "  kubectl get pods -n unihub"
echo ""
echo "  Access via port-forward:"
echo "  → API:  kubectl port-forward svc/unihub-api-service 8080:80 -n unihub"
echo "           http://localhost:8080/health"
echo ""
echo "  → Web:  kubectl port-forward svc/unihub-web-service 3000:80 -n unihub"
echo "           http://localhost:3000"
echo ""
echo "  → RabbitMQ UI: kubectl port-forward svc/rabbitmq 15672:15672 -n unihub"
echo "                  http://localhost:15672  (guest/guest)"
echo ""
echo "  → Grafana: kubectl port-forward svc/prometheus-grafana 3000:80 -n monitoring"
echo "              http://localhost:3000  (admin/prom-operator)"
echo ""
echo "  → Prometheus: kubectl port-forward svc/prometheus-kube-prometheus-prometheus 9090:9090 -n monitoring"
echo "                 http://localhost:9090"
echo ""
echo "  HPA demo (auto-scaling):"
echo "  kubectl get hpa -n unihub -w"
echo "=============================================="
