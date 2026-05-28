# ============================================
# VARIABLES — Biến đầu vào cho Terraform
# ============================================

# --- GCP Project ---
variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "asia-southeast1"
}

variable "zone" {
  description = "GCP Zone"
  type        = string
  default     = "asia-southeast1-a"
}

# --- Naming ---
variable "app_name" {
  description = "Application name prefix"
  type        = string
  default     = "unihub"
}

variable "environment" {
  description = "Environment (dev, staging, prod)"
  type        = string
  default     = "prod"
}

# --- VPC ---
variable "vpc_cidr" {
  description = "Primary CIDR range for the subnet"
  type        = string
  default     = "10.0.0.0/20"
}

variable "pods_cidr" {
  description = "Secondary CIDR for GKE Pods"
  type        = string
  default     = "10.4.0.0/14"
}

variable "services_cidr" {
  description = "Secondary CIDR for GKE Services"
  type        = string
  default     = "10.8.0.0/20"
}

# --- Cloud SQL ---
variable "db_tier" {
  description = "Cloud SQL machine tier"
  type        = string
  default     = "db-f1-micro"
}

variable "db_name" {
  description = "Database name"
  type        = string
  default     = "unihub_workshop"
}

variable "db_user" {
  description = "Database user"
  type        = string
  default     = "unihub"
}

variable "db_password" {
  description = "Database password"
  type        = string
  sensitive   = true
}

# --- Memorystore Redis ---
variable "redis_memory_size_gb" {
  description = "Redis memory size in GB"
  type        = number
  default     = 1
}

variable "redis_version" {
  description = "Redis version"
  type        = string
  default     = "REDIS_7_0"
}

# --- GKE ---
variable "gke_min_nodes" {
  description = "Minimum nodes in GKE node pool (not used with Autopilot)"
  type        = number
  default     = 1
}

# --- Secrets ---
variable "auth_secret" {
  description = "JWT Authentication Secret"
  type        = string
  sensitive   = true
}

variable "rsa_private_key" {
  description = "RSA Private Key for ticket signing"
  type        = string
  sensitive   = true
}

variable "payment_webhook_secret" {
  description = "Payment webhook verification secret"
  type        = string
  sensitive   = true
}

variable "smtp_user" {
  description = "SMTP email user"
  type        = string
  default     = ""
}

variable "smtp_pass" {
  description = "SMTP email password"
  type        = string
  sensitive   = true
  default     = ""
}

variable "gemini_api_key" {
  description = "Google Gemini API Key"
  type        = string
  sensitive   = true
  default     = ""
}

# --- Domain ---
variable "domain_name" {
  description = "Domain name for the application (e.g., unihub-test.dev)"
  type        = string
  default     = ""
}
