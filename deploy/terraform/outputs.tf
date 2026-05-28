# ============================================
# OUTPUTS — Giá trị xuất ra sau terraform apply
# ============================================

# --- VPC ---
output "vpc_name" {
  description = "VPC network name"
  value       = google_compute_network.vpc.name
}

output "subnet_name" {
  description = "Subnet name"
  value       = google_compute_subnetwork.main.name
}

# --- Cloud SQL ---
output "cloudsql_instance_name" {
  description = "Cloud SQL instance name"
  value       = google_sql_database_instance.postgres.name
}

output "cloudsql_private_ip" {
  description = "Cloud SQL private IP address"
  value       = google_sql_database_instance.postgres.private_ip_address
}

output "cloudsql_connection_name" {
  description = "Cloud SQL connection name (for Cloud SQL Proxy)"
  value       = google_sql_database_instance.postgres.connection_name
}

# --- Redis ---
output "redis_host" {
  description = "Memorystore Redis host"
  value       = google_redis_instance.cache.host
}

output "redis_port" {
  description = "Memorystore Redis port"
  value       = google_redis_instance.cache.port
}

# --- GKE ---
output "gke_cluster_name" {
  description = "GKE cluster name"
  value       = google_container_cluster.autopilot.name
}

output "gke_cluster_endpoint" {
  description = "GKE cluster endpoint"
  value       = google_container_cluster.autopilot.endpoint
  sensitive   = true
}

# --- Artifact Registry ---
output "docker_registry_url" {
  description = "Docker registry URL for pushing images"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.docker.repository_id}"
}

# --- Service Accounts ---
output "workload_sa_email" {
  description = "GKE workload service account email"
  value       = google_service_account.gke_workload.email
}

output "cicd_sa_email" {
  description = "CI/CD service account email"
  value       = google_service_account.cicd.email
}

# --- Connection Strings (cho K8s ConfigMap) ---
output "db_connection_string" {
  description = "PostgreSQL connection string template (password excluded)"
  value       = "host=${google_sql_database_instance.postgres.private_ip_address} port=5432 user=${var.db_user} dbname=${var.db_name} sslmode=require"
}

output "redis_addr" {
  description = "Redis address for app config"
  value       = "${google_redis_instance.cache.host}:${google_redis_instance.cache.port}"
}

# --- Kubectl config command ---
output "kubectl_config_command" {
  description = "Command to configure kubectl"
  value       = "gcloud container clusters get-credentials ${google_container_cluster.autopilot.name} --region ${var.region} --project ${var.project_id}"
}

# --- Docker push command ---
output "docker_push_command" {
  description = "Command to push Docker image"
  value       = "docker push ${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.docker.repository_id}/unihub-api:latest"
}
