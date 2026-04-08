# Custom Domain: Cloud Run domain mapping + DNS CNAME record
# This file contains resources for mapping a custom domain to the Cloud Run service.
# All resources are conditional based on custom_domain in config.yaml.
#
# Resources:
# - Data source for the DNS zone (created in init/)
# - Cloud Run domain mapping (provisions Google-managed SSL)
# - DNS CNAME record pointing to ghs.googlehosted.com

# ============================================
# DATA SOURCES
# ============================================

# Reference the DNS zone created in init/
data "google_dns_managed_zone" "domain_zone" {
  count = local.has_custom_domain ? 1 : 0

  name    = local.dns_zone_name
  project = local.project_id
}

# ============================================
# CLOUD RUN DOMAIN MAPPING
# ============================================

resource "google_cloud_run_domain_mapping" "mcp" {
  count = local.has_custom_domain ? 1 : 0

  location = google_cloud_run_v2_service.mcp.location
  name     = local.custom_domain

  metadata {
    namespace = local.project_id
  }

  spec {
    route_name = google_cloud_run_v2_service.mcp.name
  }

  depends_on = [
    google_cloud_run_v2_service.mcp,
  ]
}

# ============================================
# DNS CNAME RECORD
# ============================================

resource "google_dns_record_set" "mcp_cname" {
  count = local.has_custom_domain ? 1 : 0

  name         = "${local.custom_domain}."
  managed_zone = data.google_dns_managed_zone.domain_zone[0].name
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["ghs.googlehosted.com."]
}

# ============================================
# OUTPUTS
# ============================================

output "custom_domain_url" {
  description = "MCP server custom domain URL"
  value       = local.has_custom_domain ? "https://${local.custom_domain}" : ""
}

output "oauth2_callback_url" {
  description = "OAuth2 callback URL to configure in Google Cloud Console Credentials page"
  value       = local.has_custom_domain ? "https://${local.custom_domain}/oauth/callback" : ""
}

output "oauth2_action_required" {
  description = "Action required: update OAuth2 callback URL in Google Cloud Console"
  value       = local.has_custom_domain ? "ACTION REQUIRED: Add https://${local.custom_domain}/oauth/callback as an authorized redirect URI in Google Cloud Console > APIs & Services > Credentials > OAuth 2.0 Client ID. Failure to do this will cause redirect_uri_mismatch errors on all OAuth2 flows." : ""
}
