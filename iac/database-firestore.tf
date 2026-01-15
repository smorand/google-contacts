# Firestore Database: API Keys storage for MCP server authentication
# This file contains the Firestore database configuration for API key management
#
# Resources:
# - Firestore database in Native mode
# - Composite indexes for efficient queries
#
# Collection structure (documented, not created by Terraform):
# api_keys/{api_key_uuid}
#   ├── refresh_token: string     # OAuth refresh token for Google API access
#   ├── user_email: string        # Email from OAuth flow
#   ├── created_at: string        # ISO 8601 timestamp
#   └── description: string       # Optional key description

# ============================================
# LOCALS
# ============================================

locals {
  # Firestore configuration from config.yaml
  firestore_config   = lookup(local.gcp_resources, "firestore", {})
  firestore_database = lookup(local.firestore_config, "database_id", "(default)")
  firestore_location = lookup(local.firestore_config, "location_id", "eur3")
}

# ============================================
# FIRESTORE DATABASE
# ============================================

# Note: Firestore database can only be created once per project.
# If the default database already exists, this resource will fail.
# In that case, comment out or import the existing database.

resource "google_firestore_database" "main" {
  project     = local.project_id
  name        = local.firestore_database
  location_id = local.firestore_location
  type        = "FIRESTORE_NATIVE"

  # Concurrency mode for better performance
  concurrency_mode = "OPTIMISTIC"

  # App Engine integration (disabled)
  app_engine_integration_mode = "DISABLED"

  # Point-in-time recovery (optional, disabled for cost savings)
  point_in_time_recovery_enablement = "POINT_IN_TIME_RECOVERY_DISABLED"

  # Deletion policy - prevent accidental deletion
  delete_protection_state = "DELETE_PROTECTION_ENABLED"
}

# ============================================
# FIRESTORE INDEXES
# ============================================

# Index for listing API keys by creation date (for admin purposes)
resource "google_firestore_index" "api_keys_created_at" {
  project    = local.project_id
  database   = google_firestore_database.main.name
  collection = "api_keys"

  fields {
    field_path = "created_at"
    order      = "DESCENDING"
  }

  fields {
    field_path = "__name__"
    order      = "DESCENDING"
  }

  depends_on = [google_firestore_database.main]
}

# Index for finding keys by user email (for listing user's keys)
resource "google_firestore_index" "api_keys_user_email" {
  project    = local.project_id
  database   = google_firestore_database.main.name
  collection = "api_keys"

  fields {
    field_path = "user_email"
    order      = "ASCENDING"
  }

  fields {
    field_path = "created_at"
    order      = "DESCENDING"
  }

  fields {
    field_path = "__name__"
    order      = "DESCENDING"
  }

  depends_on = [google_firestore_database.main]
}

# ============================================
# OUTPUTS
# ============================================

output "firestore_database_name" {
  description = "Firestore database name"
  value       = google_firestore_database.main.name
}

output "firestore_location" {
  description = "Firestore database location"
  value       = google_firestore_database.main.location_id
}
