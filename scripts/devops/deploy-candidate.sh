#!/usr/bin/env bash
set -euo pipefail
: "${CI_NAMESPACE:?} ${IMAGE_REGISTRY:?} ${GIT_SHA:?}"
if [[ "$CI_NAMESPACE" != unihub-ci ]]; then
  echo "Candidate deployments are restricted to unihub-ci" >&2; exit 1
fi
bash scripts/devops/create-secrets.sh "$CI_NAMESPACE" unihub-ci-secret
# Helm hooks are recreated by Helm; ArgoCD uses the same job as a Sync hook.
helm upgrade --install unihub-ci deploy/helm/unihub -n "$CI_NAMESPACE" \
  --set secrets.existingSecret=unihub-ci-secret --set imagePullSecrets[0].name=registry-credentials \
  --set api.image.repository="$IMAGE_REGISTRY/unihub-backend" --set api.image.tag="$GIT_SHA" \
  --set worker.image.repository="$IMAGE_REGISTRY/unihub-backend" --set worker.image.tag="$GIT_SHA" \
  --set web.image.repository="$IMAGE_REGISTRY/unihub-web" --set web.image.tag="$GIT_SHA" \
  --set config.dbName=unihub_ci_test --set-string config.runSeed=false \
  --set media.enabled=false --set worker.autoscaling.enabled=false --set podMonitor.enabled=false \
  --wait --timeout 8m
kubectl -n "$CI_NAMESPACE" rollout status deploy/unihub-worker --timeout=180s
