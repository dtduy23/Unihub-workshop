# ============================================
# IAM — Service Accounts & Roles
# ============================================

# ========== GKE Workload Service Account ==========
# Service Account mà các Pod sẽ sử dụng (qua Workload Identity)
resource "google_service_account" "gke_workload" {
  account_id   = "${var.app_name}-workload"
  display_name = "UniHub GKE Workload SA"
  description  = "Service account for UniHub pods running on GKE"
}

# --- Quyền đọc Secret Manager ---
resource "google_project_iam_member" "workload_secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.gke_workload.email}"
}

# --- Quyền đọc Artifact Registry (pull images) ---
resource "google_project_iam_member" "workload_artifact_reader" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.gke_workload.email}"
}

# --- Quyền Cloud SQL Client ---
resource "google_project_iam_member" "workload_cloudsql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.gke_workload.email}"
}

# --- Quyền ghi Log ---
resource "google_project_iam_member" "workload_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.gke_workload.email}"
}

# --- Quyền ghi Metrics ---
resource "google_project_iam_member" "workload_metric_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.gke_workload.email}"
}

# --- Workload Identity Binding ---
# Cho phép K8s Service Account "unihub-api" trong namespace "unihub"
# sử dụng GCP Service Account này
resource "google_service_account_iam_member" "workload_identity_api" {
  service_account_id = google_service_account.gke_workload.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[unihub/unihub-api]"
}

resource "google_service_account_iam_member" "workload_identity_worker" {
  service_account_id = google_service_account.gke_workload.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[unihub/unihub-worker]"
}

# ========== CI/CD Service Account (Jenkins) ==========
resource "google_service_account" "cicd" {
  account_id   = "${var.app_name}-cicd"
  display_name = "UniHub CI/CD SA"
  description  = "Service account for Jenkins CI/CD pipeline"
}

# --- Quyền push images ---
resource "google_project_iam_member" "cicd_artifact_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.cicd.email}"
}

# --- Quyền deploy lên GKE ---
resource "google_project_iam_member" "cicd_gke_developer" {
  project = var.project_id
  role    = "roles/container.developer"
  member  = "serviceAccount:${google_service_account.cicd.email}"
}
