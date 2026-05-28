# ============================================
# CLOUD DNS — Zone & Records
# ============================================

# Chỉ tạo khi có domain_name
resource "google_dns_managed_zone" "main" {
  count = var.domain_name != "" ? 1 : 0

  name        = "${var.app_name}-zone"
  dns_name    = "${var.domain_name}."
  description = "DNS zone for UniHub Workshop"

  dnssec_config {
    state = "on"
  }

  depends_on = [google_project_service.apis]
}

# A record cho api subdomain — sẽ trỏ vào Load Balancer IP
# Uncomment sau khi có Ingress IP từ GKE
# resource "google_dns_record_set" "api" {
#   count = var.domain_name != "" ? 1 : 0
#
#   name         = "api.${var.domain_name}."
#   managed_zone = google_dns_managed_zone.main[0].name
#   type         = "A"
#   ttl          = 300
#   rrdatas      = ["<LOAD_BALANCER_IP>"]
# }
