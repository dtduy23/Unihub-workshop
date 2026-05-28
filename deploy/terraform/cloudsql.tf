# ============================================
# CLOUD SQL — PostgreSQL 15 (Private IP)
# ============================================

resource "google_sql_database_instance" "postgres" {
  name                = "${var.app_name}-postgres"
  database_version    = "POSTGRES_15"
  region              = var.region
  deletion_protection = false # Tắt khi dùng credit, bật ON khi production thật

  settings {
    tier              = var.db_tier
    availability_type = "ZONAL" # Dùng REGIONAL nếu cần HA
    disk_size         = 10
    disk_type         = "PD_SSD"
    disk_autoresize   = true

    ip_configuration {
      ipv4_enabled                                  = false # Chỉ Private IP
      private_network                               = google_compute_network.vpc.id
      enable_private_path_for_google_cloud_services = true
    }

    backup_configuration {
      enabled                        = true
      start_time                     = "03:00" # 3 AM UTC = 10 AM VN
      point_in_time_recovery_enabled = true
      backup_retention_settings {
        retained_backups = 7
      }
    }

    database_flags {
      name  = "max_connections"
      value = "100"
    }

    database_flags {
      name  = "log_min_duration_statement"
      value = "1000" # Log queries > 1 second
    }

    maintenance_window {
      day          = 7 # Sunday
      hour         = 3 # 3 AM UTC
      update_track = "stable"
    }
  }

  depends_on = [google_service_networking_connection.private_vpc_connection]
}

# ========== Database ==========
resource "google_sql_database" "main" {
  name     = var.db_name
  instance = google_sql_database_instance.postgres.name
}

# ========== Database User ==========
resource "google_sql_user" "app" {
  name     = var.db_user
  instance = google_sql_database_instance.postgres.name
  password = var.db_password
}
