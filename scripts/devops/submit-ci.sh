#!/usr/bin/env bash
set -euo pipefail
: "${IMAGE_REGISTRY:?Set IMAGE_REGISTRY to your writable container registry namespace}"
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
revision=${GIT_SHA:-$(git -C "$repo_root" rev-parse HEAD)}
repository=${GIT_REPOSITORY:-https://github.com/dtduy23/Unihub-workshop.git}
command -v argo >/dev/null || { echo "Install the Argo CLI matching the controller" >&2; exit 1; }
[[ "$revision" =~ ^[0-9a-f]{40}$ ]] || { echo "GIT_SHA must be a full immutable commit SHA" >&2; exit 1; }
workflow=$(argo submit -n unihub-ci --from workflowtemplate/unihub-ci \
  -p "revision=$revision" -p "repository=$repository" -p "image-registry=$IMAGE_REGISTRY" \
  -p "promote=${PROMOTE:-false}" -o name)
argo watch -n unihub-ci "$workflow"
phase=$(argo get -n unihub-ci "$workflow" -o json | python3 -c 'import json,sys; print(json.load(sys.stdin)["status"]["phase"])')
[[ "$phase" == Succeeded ]] || { echo "Workflow $workflow ended with $phase" >&2; exit 1; }
