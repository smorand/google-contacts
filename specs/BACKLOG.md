# Backlog

## BL-001: Separate Shared-Infra GCP Project for DNS Zone

- **Idea:** Move the `scm-platform.org` Cloud DNS zone to a dedicated GCP project (e.g., `scm-platform-infra`) that is never changed, instead of managing it in each service project's init/ layer.
- **Description:** Currently (per spec `2026-02-22_13:43:06-custom-domain-mapping.md`), the DNS zone lives in the same project as the Cloud Run service, managed by init/ Terraform. This means changing `project_id` requires recreating the zone in the new project, which may cause nameserver changes and require Squarespace delegation updates. A dedicated infra project would eliminate this risk entirely: the zone stays permanent, and each service project only creates CNAME records in the shared zone via cross-project references.
- **Rationale for deferral:** The current approach (Approach A) is simpler and sufficient for the current single-service use case. The user rarely changes `project_id`. If multiple services start sharing the `scm-platform.org` zone (SC-003), this becomes more compelling.
- **Suggested by:** Spec `2026-02-22_13:43:06-custom-domain-mapping.md`, Phase 2 Approach B
