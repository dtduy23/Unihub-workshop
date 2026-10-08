#!/usr/bin/env bash
set -euo pipefail
: "${CI_NAMESPACE:?}"
[[ "$CI_NAMESPACE" == unihub-ci ]] || { echo "Refusing cleanup outside unihub-ci" >&2; exit 1; }
helm uninstall unihub-ci -n "$CI_NAMESPACE" --ignore-not-found --wait --timeout 3m
kubectl -n "$CI_NAMESPACE" delete job unihub-migrate --ignore-not-found --wait=true
# StatefulSet volume claims outlive Helm uninstalls, so remove only CI-labeled claims.
kubectl -n "$CI_NAMESPACE" delete pvc -l app.kubernetes.io/instance=unihub-ci --wait=true --timeout=120s
kubectl -n "$CI_NAMESPACE" delete secret unihub-ci-secret --ignore-not-found
