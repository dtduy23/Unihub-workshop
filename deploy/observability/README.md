# UniHub Observability Stack

Thư mục quản lý cấu hình Telemetry, Centralized Logging và Monitoring:

* `logging/`:
  * Fluentbit DaemonSet với JSON parsing (`correlation_id`, `student_id`, `service`, `duration`).
  * OpenSearch StatefulSet & OpenSearch Dashboards.
* `metrics/`:
  * Prometheus PodMonitor (scrapes metrics từ API & Worker).
  * Grafana Dashboards cấu hình sẵn cho UniHub Concurrency & Queue Backlog.
