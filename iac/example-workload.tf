# Example workload: Cloud Run service with service account and permissions
# This file demonstrates the file organization pattern for resource sets

# Local definitions for this workload
locals {
  api_name    = "api"
  api_sa_name = "${local.api_name}-sa"  # Service account name

  # Docker image (uses GCP project_id)
  api_image = "gcr.io/${local.project_id}/${local.api_name}:latest"

  # Resource limits from config (check cloud_run config first, then resources)
  api_cpu    = lookup(local.cloud_run_config, "cpu", lookup(local.resources, "cpu", "1000m"))
  api_memory = lookup(local.cloud_run_config, "memory", lookup(local.resources, "memory", "512Mi"))
}

# Service account for the API workload
resource "google_service_account" "api" {
  account_id   = local.api_sa_name
  display_name = "API Service Account"
  description  = "Service account for API workload"
}

# Cloud Run service
resource "google_cloud_run_service" "api" {
  name     = local.api_name
  location = local.cloud_run_region  # Uses Cloud Run region from config

  template {
    spec {
      service_account_name = google_service_account.api.email

      containers {
        image = local.api_image

        resources {
          limits = {
            cpu    = local.api_cpu
            memory = local.api_memory
          }
        }

        # Inject standard env vars
        env {
          name  = "PROJECT_ID"
          value = local.project_id
        }

        env {
          name  = "ENVIRONMENT"
          value = local.env
        }

        # Inject custom parameters from config
        dynamic "env" {
          for_each = local.parameters
          content {
            name  = upper(replace(env.key, "-", "_"))
            value = tostring(env.value)
          }
        }
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}

# IAM permissions for the API service account
resource "google_project_iam_member" "api_viewer" {
  project = local.project_id
  role    = "roles/viewer"
  member  = "serviceAccount:${google_service_account.api.email}"
}

# Make Cloud Run service publicly accessible (adjust as needed)
resource "google_cloud_run_service_iam_member" "api_public" {
  service  = google_cloud_run_service.api.name
  location = google_cloud_run_service.api.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Outputs (inline in each resource file)
output "api_url" {
  description = "API service URL"
  value       = google_cloud_run_service.api.status[0].url
}

output "api_service_account" {
  description = "API service account email"
  value       = google_service_account.api.email
}
