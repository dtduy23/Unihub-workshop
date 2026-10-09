#!/usr/bin/env bash
set -euo pipefail
namespace=${1:-unihub-demo}
image=${2:?Usage: run-loadtest.sh NAMESPACE BACKEND_IMAGE}
case "$namespace" in unihub-demo|unihub-ci) ;; *) echo "Load tests require a disposable demo/CI namespace" >&2; exit 1;; esac
secret=unihub-secret
[[ "$namespace" == unihub-ci ]] && secret=unihub-ci-secret
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
kubectl -n "$namespace" create configmap unihub-k6-script --from-file="k6_load_test.js=$repo_root/src/backend/tests/k6_load_test.js" --dry-run=client -o yaml | kubectl apply -f -
work_file=$(mktemp)
trap 'rm -f "$work_file"' EXIT
python3 - "$namespace" "$image" "$secret" > "$work_file" <<'PYJOB'
import json,sys
namespace,image,secret=sys.argv[1:]
job={"apiVersion":"batch/v1","kind":"Job","metadata":{"generateName":"unihub-k6-","namespace":namespace},"spec":{"backoffLimit":0,"activeDeadlineSeconds":180,"ttlSecondsAfterFinished":3600,"template":{"spec":{"restartPolicy":"Never","automountServiceAccountToken":False,"securityContext":{"runAsUser":10001,"runAsGroup":10001,"fsGroup":10001},"initContainers":[{"name":"fixture","image":image,"command":["/app/loadtest-seed"],"envFrom":[{"configMapRef":{"name":"unihub-config"}},{"secretRef":{"name":secret}}],"env":[{"name":"FIXTURE_FILE","value":"/fixtures/sessions.json"}],"volumeMounts":[{"name":"fixtures","mountPath":"/fixtures"}]}],"containers":[{"name":"k6","image":"grafana/k6:0.57.0","command":["k6","run","/scripts/k6_load_test.js"],"volumeMounts":[{"name":"fixtures","mountPath":"/fixtures"},{"name":"script","mountPath":"/scripts"}],"resources":{"requests":{"cpu":"100m","memory":"128Mi"},"limits":{"cpu":"1000m","memory":"512Mi"}}}],"volumes":[{"name":"fixtures","emptyDir":{}},{"name":"script","configMap":{"name":"unihub-k6-script"}}]}}}}
if namespace == "unihub-ci":
    job["spec"]["template"]["spec"]["imagePullSecrets"] = [{"name": "registry-credentials"}]
print(json.dumps(job))
PYJOB
job=$(kubectl create -f "$work_file" -o jsonpath='{.metadata.name}')
if ! kubectl wait -n "$namespace" --for=condition=complete "job/$job" --timeout=180s; then
  kubectl logs -n "$namespace" "job/$job" -c fixture || true
  kubectl logs -n "$namespace" "job/$job" -c k6 || true
  exit 1
fi
kubectl logs -n "$namespace" "job/$job" -c k6
