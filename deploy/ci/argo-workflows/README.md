# Argo Workflows CI Pipeline

Template Cloud-Native CI Pipeline theo mô hình DAG (Directed Acyclic Graph) cho UniHub Platform:

* `ci-pipeline.yaml`:
  * **DAG Step 1**: Go Lint & Data Race Detection (`go test -race ./...`).
  * **DAG Step 2**: Python Validation.
  * **DAG Step 3**: Multi-stage Container Build (Alpine/Distroless).
  * **DAG Step 4**: k6 Performance Regression Gate (Threshold: p95 < 200ms, error rate < 0.5%).
