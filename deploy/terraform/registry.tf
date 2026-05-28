# ============================================
# ARTIFACT REGISTRY — Docker Image Repository
# ============================================

resource "google_artifact_registry_repository" "docker" {
  location      = var.region
  repository_id = "${var.app_name}-docker"
  description   = "Docker images for UniHub Workshop"
  format        = "DOCKER"

  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"

    most_recent_versions {
      keep_count = 10
    }
  }

  depends_on = [google_project_service.apis]
}
