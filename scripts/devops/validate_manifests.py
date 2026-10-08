#!/usr/bin/env python3
"""Validate rendered resource relationships without a Kubernetes API server."""
import argparse
from pathlib import Path
import yaml

def documents(path):
    return [doc for doc in yaml.safe_load_all(Path(path).read_text()) if doc]

def validate_chart(path):
    docs = documents(path)
    identities = [(d["apiVersion"], d["kind"], d["metadata"]["name"]) for d in docs]
    assert len(identities) == len(set(identities)), f"Duplicate resource in {path}"
    workloads = [d for d in docs if d["kind"] in {"Deployment", "StatefulSet"}]
    for svc in (d for d in docs if d["kind"] == "Service"):
        selector = svc["spec"]["selector"]
        matched = [w for w in workloads if all(w["spec"]["template"]["metadata"]["labels"].get(k) == v for k,v in selector.items())]
        assert matched, f"Service {svc['metadata']['name']} selects no workload"
        ports = {p.get("name") for w in matched for c in w["spec"]["template"]["spec"]["containers"] for p in c.get("ports", [])}
        for port in svc["spec"]["ports"]:
            assert not isinstance(port["targetPort"], str) or port["targetPort"] in ports, f"Service targets missing port: {port}"
    controlled = set()
    for d in docs:
        if d["kind"] in {"HorizontalPodAutoscaler", "ScaledObject"}:
            target = d["spec"]["scaleTargetRef"]["name"]
            assert target not in controlled, f"Multiple autoscalers control {target}"
            controlled.add(target)
    for w in workloads:
        if w["metadata"]["name"] in {"unihub-api", "unihub-worker"}:
            container = w["spec"]["template"]["spec"]["containers"][0]
            assert container["readinessProbe"]["httpGet"]["path"] == "/ready"
            assert container["livenessProbe"]["httpGet"]["path"] == "/health"
            assert w["metadata"]["annotations"]["argocd.argoproj.io/sync-wave"] == "3"
    config = next(d for d in docs if d["kind"] == "ConfigMap" and d["metadata"]["name"] == "unihub-config")
    for key in {"DB_PASSWORD", "RABBITMQ_URL", "AUTH_SECRET", "RSA_PRIVATE_KEY", "REDIS_PASSWORD"}:
        assert key not in config["data"], f"Credential {key} belongs in a Secret"
    for doc in docs:
        if doc["kind"] == "ConfigMap":
            assert all(isinstance(v, str) for v in doc.get("data", {}).values())
    return len(docs)

def validate_gitops(root):
    root = Path(root)
    staging = documents(root / "deploy/argocd/application-staging.yaml")[0]
    prod = documents(root / "deploy/argocd/application-prod.yaml")[0]
    assert staging["spec"]["syncPolicy"]["automated"] == {"prune": True, "selfHeal": True}
    assert "automated" not in prod["spec"]["syncPolicy"]
    pipeline = documents(root / "deploy/argo-workflows/ci-pipeline.yaml")[0]["spec"]
    tasks = {task["name"]: task for task in pipeline["templates"][0]["dag"]["tasks"]}
    for build in ("build-backend", "build-web"):
        assert all(gate + ".Succeeded" in tasks[build]["depends"] for gate in ("go-tests", "python-tests", "web-tests", "manifests"))
    assert tasks["promote"]["depends"] == "performance.Succeeded"
    assert pipeline["onExit"] == "cleanup" and pipeline.get("synchronization")
    return len(tasks)

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--root", default=".")
    p.add_argument("renders", nargs="+")
    args = p.parse_args()
    for path in args.renders:
        print(f"Validated {validate_chart(path)} rendered resources: {path}")
    print(f"Validated {validate_gitops(args.root)} CI DAG tasks and GitOps policies")
if __name__ == "__main__": main()
