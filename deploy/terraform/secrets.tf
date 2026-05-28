# ============================================
# SECRET MANAGER — Quản lý secrets an toàn
# ============================================

# --- Database Password ---
resource "google_secret_manager_secret" "db_password" {
  secret_id = "${var.app_name}-db-password"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "db_password" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = var.db_password
}

# --- JWT Auth Secret ---
resource "google_secret_manager_secret" "auth_secret" {
  secret_id = "${var.app_name}-auth-secret"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "auth_secret" {
  secret      = google_secret_manager_secret.auth_secret.id
  secret_data = var.auth_secret
}

# --- RSA Private Key ---
resource "google_secret_manager_secret" "rsa_private_key" {
  secret_id = "${var.app_name}-rsa-private-key"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "rsa_private_key" {
  secret      = google_secret_manager_secret.rsa_private_key.id
  secret_data = var.rsa_private_key
}

# --- Payment Webhook Secret ---
resource "google_secret_manager_secret" "payment_webhook_secret" {
  secret_id = "${var.app_name}-payment-webhook-secret"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "payment_webhook_secret" {
  secret      = google_secret_manager_secret.payment_webhook_secret.id
  secret_data = var.payment_webhook_secret
}

# --- SMTP Password ---
resource "google_secret_manager_secret" "smtp_pass" {
  secret_id = "${var.app_name}-smtp-pass"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "smtp_pass" {
  secret      = google_secret_manager_secret.smtp_pass.id
  secret_data = var.smtp_pass
}

# --- Gemini API Key ---
resource "google_secret_manager_secret" "gemini_api_key" {
  secret_id = "${var.app_name}-gemini-api-key"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "gemini_api_key" {
  secret      = google_secret_manager_secret.gemini_api_key.id
  secret_data = var.gemini_api_key
}
