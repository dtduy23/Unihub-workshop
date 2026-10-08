#!/usr/bin/env bash
set -euo pipefail
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$repo_root"
driver=${1:-minikube}
profile=unihub-devops
for command in docker kubectl helm git python3 openssl; do
  command -v "$command" >/dev/null || { echo "Missing prerequisite: $command" >&2; exit 1; }
done
python3 -c 'import yaml' || { echo "Install PyYAML first" >&2; exit 1; }
[[ -z $(git status --porcelain) ]] || { echo "Commit the source before running the reproducible demo" >&2; exit 1; }
revision=$(git rev-parse HEAD)
mkdir -p .devops
export KUBECONFIG="$repo_root/.devops/kubeconfig"
case "$driver" in
  kind)
    command -v kind >/dev/null || { echo "Install kind first" >&2; exit 1; }
    if ! kind get clusters | grep -qx "$profile"; then
      kind create cluster --name "$profile" --image kindest/node:v1.32.2 --kubeconfig "$KUBECONFIG"
    else
      kind export kubeconfig --name "$profile" --kubeconfig "$KUBECONFIG"
    fi
    ;;
  minikube)
    minikube start -p "$profile" --driver=docker --kubernetes-version=v1.32.2 --cpus=4 --memory=8192
    minikube update-context -p "$profile"
    ;;
  *) echo "Use kind or minikube" >&2; exit 1;;
esac
# Build exactly the source represented by the local repository snapshot.
docker build -t "unihub-backend:$revision" src/backend
docker build -t "unihub-web:$revision" src/web
if [[ "$driver" == kind ]]; then
  kind load docker-image --name "$profile" "unihub-backend:$revision" "unihub-web:$revision"
else
  minikube image load -p "$profile" "unihub-backend:$revision" "unihub-web:$revision"
fi
helm repo add argo https://argoproj.github.io/argo-helm --force-update
helm repo update argo
helm upgrade --install argocd argo/argo-cd --version 7.8.13 -n argocd --create-namespace --wait --timeout 8m
kubectl patch configmap argocd-cm -n argocd --type merge --patch-file deploy/argocd/health-config.yaml
kubectl apply -f deploy/argo-workflows/rbac.yaml
helm upgrade --install argo-workflows argo/argo-workflows --version 0.45.0 -n argo --create-namespace -f deploy/argo-workflows/controller-values.yaml --wait --timeout 8m
kubectl apply -f deploy/argo-workflows/ci-pipeline.yaml
kubectl create namespace unihub-demo --dry-run=client -o yaml | kubectl apply -f -
bash scripts/devops/create-secrets.sh unihub-demo
kubectl apply -f deploy/argocd/demo-repository.yaml
# Wait for a running pod so kubectl cp can fill the repository before readiness.
kubectl wait -n unihub-demo --for=jsonpath='{.status.phase}'=Running pod -l app=git-demo --timeout=180s
work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT
git clone --quiet --no-hardlinks "$repo_root" "$work_dir/checkout"
git -C "$work_dir/checkout" checkout -B main "$revision"
python3 scripts/devops/snapshot-demo.py "$work_dir/checkout" "$revision"
git -C "$work_dir/checkout" -c user.name='UniHub Demo' -c user.email='demo@unihub.example' add deploy
git -C "$work_dir/checkout" -c user.name='UniHub Demo' -c user.email='demo@unihub.example' commit --quiet -m 'demo: point GitOps to local snapshot and immutable images'
snapshot_revision=$(git -C "$work_dir/checkout" rev-parse HEAD)
git clone --quiet --bare "$work_dir/checkout" "$work_dir/repo.git"
git -C "$work_dir/repo.git" update-server-info
pod=$(kubectl get pod -n unihub-demo -l app=git-demo -o jsonpath='{.items[0].metadata.name}')
kubectl cp "$work_dir/repo.git" "unihub-demo/$pod:/srv"
kubectl rollout status deploy/git-demo -n unihub-demo --timeout=180s
kubectl apply -f "$work_dir/checkout/deploy/argocd/project.yaml" -f "$work_dir/checkout/deploy/argocd/project-infra.yaml"
kubectl apply -f "$work_dir/checkout/deploy/argocd/app-of-apps.yaml"
kubectl wait -n argocd --for=jsonpath='{.status.health.status}'=Healthy application/unihub-platform --timeout=20m
kubectl annotate -n argocd application/unihub-demo argocd.argoproj.io/refresh=hard --overwrite
kubectl wait -n argocd --for=jsonpath='{.status.operationState.syncResult.revision}'="$snapshot_revision" application/unihub-demo --timeout=20m
kubectl wait -n argocd --for=jsonpath='{.status.operationState.phase}'=Succeeded application/unihub-demo --timeout=20m
kubectl wait -n argocd --for=jsonpath='{.status.health.status}'=Healthy application/unihub-demo --timeout=10m
bash scripts/devops/run-loadtest.sh unihub-demo "unihub-backend:$revision"
echo "Demo is ready. Use KUBECONFIG=$KUBECONFIG and the port-forward commands in docs/DEVOPS_PLAYBOOK.md."
