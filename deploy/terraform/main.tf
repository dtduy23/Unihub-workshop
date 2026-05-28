# ============================================
# MAIN — Provider, Backend, API Enables
# ============================================

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.0"
    }
  }

  # Remote state trên GCS — bỏ comment khi đã tạo bucket
  # backend "gcs" {
  #   bucket = "unihub-terraform-state"
  #   prefix = "terraform/state"
  # }
}

# ========== Providers ==========
provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

# ========== Enable required GCP APIs ==========
locals {
  required_apis = [
    "compute.googleapis.com",           # VPC, Firewall, Load Balancer
    "container.googleapis.com",          # GKE
    "sqladmin.googleapis.com",           # Cloud SQL
    "redis.googleapis.com",              # Memorystore Redis
    "artifactregistry.googleapis.com",   # Docker Image Registry
    "secretmanager.googleapis.com",      # Secret Manager
    "dns.googleapis.com",               # Cloud DNS
    "servicenetworking.googleapis.com",  # Private Service Access (Cloud SQL, Redis)
    "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com",
  ]
}

resource "google_project_service" "apis" {
  for_each = toset(local.required_apis)

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}
