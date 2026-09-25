# UniHub Helm Charts

Thư mục chứa các Helm Chart đóng gói cho hệ sinh thái UniHub Platform.

## Cấu trúc dự kiến:
```text
deploy/helm/
└── unihub/                 # Chart chính của UniHub Platform
    ├── Chart.yaml          # Metadata, chart version, app version
    ├── values.yaml         # Cấu hình mặc định (Local / KinD)
    ├── values-staging.yaml # Biến cấu hình môi trường Staging
    ├── values-prod.yaml    # Biến cấu hình môi trường Production
    └── templates/          # K8s manifest templates
        ├── deployment-api.yaml
        ├── deployment-worker.yaml
        ├── deployment-web.yaml
        ├── service.yaml
        ├── ingress.yaml
        ├── hpa.yaml
        ├── configmap.yaml
        ├── secret.yaml
        └── podmonitor.yaml
```
