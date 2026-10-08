#!/usr/bin/env python3
"""Adapt a temporary checkout for the in-cluster local Git demo."""
import argparse
from pathlib import Path
import yaml
from promote_images import image_values

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("checkout")
    parser.add_argument("revision")
    args = parser.parse_args()
    root = Path(args.checkout)
    remote = "https://github.com/dtduy23/Unihub-workshop.git"
    local = "http://git-demo.unihub-demo.svc.cluster.local:8080/repo.git"
    for path in (root / "deploy/argocd").rglob("*.yaml"):
        path.write_text(path.read_text().replace(remote, local))
    for project in ["project.yaml", "project-infra.yaml"]:
        path = root / "deploy/argocd" / project
        value = yaml.safe_load(path.read_text())
        if local not in value["spec"]["sourceRepos"]:
            value["spec"]["sourceRepos"].append(local)
        path.write_text(yaml.safe_dump(value, sort_keys=False))
    path = root / "deploy/argocd/app-of-apps.yaml"
    value = yaml.safe_load(path.read_text())
    value["spec"]["source"]["directory"]["include"] = "{application-demo.yaml,apps/*.yaml}"
    path.write_text(yaml.safe_dump(value, sort_keys=False))
    path = root / "deploy/argocd/apps/metrics-server.yaml"
    value = yaml.safe_load(path.read_text())
    # Local kind/minikube kubelet certificates are not trusted by metrics-server.
    value["spec"]["source"]["helm"]["values"] = "args: [--kubelet-insecure-tls]\n"
    path.write_text(yaml.safe_dump(value, sort_keys=False))
    # image_values validates the SHA; local repositories intentionally omit a registry.
    values = image_values("local", args.revision).replace("local/unihub-", "unihub-")
    (root / "deploy/helm/unihub/images-demo.yaml").write_text(values)
    path = root / "deploy/helm/unihub/values-demo.yaml"
    value = yaml.safe_load(path.read_text())
    value["config"]["dbName"] = "unihub_demo_test"
    path.write_text(yaml.safe_dump(value, sort_keys=False))
if __name__ == "__main__": main()
