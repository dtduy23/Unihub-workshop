# ============================================
# MEMORYSTORE — Redis 7 (Private IP)
# ============================================

resource "google_redis_instance" "cache" {
  name               = "${var.app_name}-redis"
  tier               = "BASIC" # BASIC cho dev, STANDARD_HA cho prod
  memory_size_gb     = var.redis_memory_size_gb
  region             = var.region
  redis_version      = var.redis_version
  authorized_network = google_compute_network.vpc.id

  redis_configs = {
    maxmemory-policy = "allkeys-lru"
    notify-keyspace-events = "Ex" # Cho TTL expiration events
  }

  maintenance_policy {
    weekly_maintenance_window {
      day = "SUNDAY"
      start_time {
        hours   = 3
        minutes = 0
      }
    }
  }

  depends_on = [google_project_service.apis]
}
