# ============================================
# GKE — Autopilot Cluster
# ============================================

resource "google_container_cluster" "autopilot" {
  name     = "${var.app_name}-cluster"
  location = var.region

  # Autopilot mode — Google quản lý node pool
  enable_autopilot = true

  network    = google_compute_network.vpc.id
  subnetwork = google_compute_subnetwork.main.id

  ip_allocation_policy {
    cluster_secondary_range_name  = "pods"
    services_secondary_range_name = "services"
  }

  # Private cluster — nodes không có Public IP
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false # Cho phép kubectl từ bên ngoài
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }

  # Logging & Monitoring
  logging_config {
    enable_components = ["SYSTEM_COMPONENTS", "WORKLOADS"]
  }

  monitoring_config {
    enable_components = ["SYSTEM_COMPONENTS", "STORAGE", "POD", "DEPLOYMENT", "STATEFULSET"]
    managed_prometheus {
      enabled = true
    }
  }

  # Release channel
  release_channel {
    channel = "REGULAR"
  }

  # Maintenance window
  maintenance_policy {
    recurring_window {
      start_time = "2026-01-01T03:00:00Z"
      end_time   = "2026-01-01T07:00:00Z"
      recurrence = "FREQ=WEEKLY;BYDAY=SU"
    }
  }

  # Workload Identity — an toàn hơn Service Account keys
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  deletion_protection = false # Tắt khi dùng credit

  depends_on = [
    google_project_service.apis,
    google_compute_subnetwork.main,
  ]
}
