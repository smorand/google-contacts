locals {
  # Load configuration from config.yaml
  config_file = yamldecode(file("${path.root}/../config.yaml"))
  config      = local.config_file

  # Global fields
  prefix = local.config.prefix
  env    = local.config.env

  # GCP configuration
  gcp = lookup(local.config, "gcp", {})

  # GCP Project ID (explicit from config)
  project_id = local.gcp.project_id

  # GCP Location
  location = local.gcp.location

  # Detect if multi-region
  is_multi_region = contains(["us", "eu", "asia"], local.location)

  # Compute location_id for naming
  location_id = local.is_multi_region ? local.location : (
    "${substr(local.location, 0, 1)}${substr(split("-", local.location)[1], 0, 1)}${regex("\\d+", local.location)}"
  )

  # State backend naming (uses prefix-iac-location_id-env)
  state_bucket_name = "${local.prefix}-iac-${local.location_id}-${local.env}"

  # Services to enable
  services = lookup(local.gcp, "services", [
    "storage.googleapis.com",
    "logging.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com"
  ])

  # Custom domain configuration
  cloud_run_config = lookup(lookup(local.gcp, "resources", {}), "cloud_run", {})
  custom_domain    = lookup(local.cloud_run_config, "custom_domain", "")
  has_custom_domain = local.custom_domain != ""

  # Base domain extraction (e.g., "contacts.mcp.scm-platform.org" -> "scm-platform.org")
  domain_parts = local.has_custom_domain ? split(".", local.custom_domain) : []
  base_domain  = local.has_custom_domain ? join(".", slice(local.domain_parts, length(local.domain_parts) - 2, length(local.domain_parts))) : ""

  # DNS zone name (e.g., "scm-platform.org" -> "scm-platform-org")
  dns_zone_name = local.has_custom_domain ? replace(local.base_domain, ".", "-") : ""
}
