#!/usr/bin/env bash
set -euo pipefail
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$repo_root"
work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT
renders=()
for environment in default staging prod demo; do
  values=()
  [[ "$environment" == default ]] || values=(-f "deploy/helm/unihub/values-$environment.yaml")
  helm lint deploy/helm/unihub "${values[@]}" --strict
  helm template unihub deploy/helm/unihub -n "unihub-$environment" "${values[@]}" > "$work_dir/$environment.yaml"
  renders+=("$work_dir/$environment.yaml")
done
helm template unihub deploy/helm/unihub -n unihub-staging -f deploy/helm/unihub/values-staging.yaml --set api.autoscaling.keda.enabled=true > "$work_dir/keda.yaml"
renders+=("$work_dir/keda.yaml")
python3 scripts/devops/validate_manifests.py "${renders[@]}"
kubectl kustomize deploy/logging > "$work_dir/logging.yaml"
# Native resources can additionally be checked against Kubernetes JSON schemas.
if command -v kubeconform >/dev/null 2>&1; then
  kubeconform -strict -summary -ignore-missing-schemas "${renders[@]}" "$work_dir/logging.yaml"
fi
for path in scripts/devops/*.sh; do bash -n "$path"; done
if command -v shellcheck >/dev/null 2>&1; then shellcheck scripts/devops/*.sh; fi
if command -v argo >/dev/null 2>&1; then argo lint --offline deploy/argo-workflows/ci-pipeline.yaml; fi
python3 -m unittest discover -s scripts/tests -v
