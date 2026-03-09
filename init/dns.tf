# Cloud DNS Zone: Managed zone for custom domain
# This file contains the foundational DNS zone resource.
# The zone is created conditionally based on custom_domain in config.yaml.
#
# Resources:
# - Cloud DNS managed zone for the base domain (e.g., scm-platform.org)

# ============================================
# DNS MANAGED ZONE
# ============================================

resource "google_dns_managed_zone" "domain_zone" {
  count = local.has_custom_domain ? 1 : 0

  name        = local.dns_zone_name
  dns_name    = "${local.base_domain}."
  description = "DNS zone for ${local.base_domain}"

  labels = {
    environment = local.env
    managed_by  = "terraform"
  }

  depends_on = [
    google_project_service.required_apis["dns.googleapis.com"],
  ]
}

# ============================================
# OUTPUTS
# ============================================

output "dns_zone_name" {
  description = "Cloud DNS zone name"
  value       = local.has_custom_domain ? google_dns_managed_zone.domain_zone[0].name : ""
}

output "dns_zone_dns_name" {
  description = "Cloud DNS zone DNS name"
  value       = local.has_custom_domain ? google_dns_managed_zone.domain_zone[0].dns_name : ""
}

output "dns_zone_nameservers" {
  description = "Nameserver records for the DNS zone. Verify these match your domain registrar delegation."
  value       = local.has_custom_domain ? google_dns_managed_zone.domain_zone[0].name_servers : []
}
