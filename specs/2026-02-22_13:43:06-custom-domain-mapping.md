# Custom Domain Mapping for Google Contacts MCP Server -- Specification Document

> Generated on: 2026-02-22
> Version: 1.0
> Status: Draft

## 1. Executive Summary

This specification defines the infrastructure changes required to expose the existing Google Contacts MCP Server (deployed on Cloud Run) through a custom domain: `contacts.mcp.scm-platform.org`. All changes are infrastructure-only (Terraform), with no modifications to application code. The DNS zone for `scm-platform.org` will be managed in the `init/` Terraform layer (foundational infrastructure), while the Cloud Run domain mapping and DNS record will be managed in the `iac/` layer. The domain will be fully configurable via `config.yaml`.

After deployment, the OAuth2 callback URL in the Google Cloud Console must be updated manually to reflect the new custom domain.

## 2. Scope

### 2.1 In Scope

- Cloud DNS zone creation for `scm-platform.org` in `init/` Terraform
- Cloud Run domain mapping resource in `iac/` Terraform
- DNS CNAME record for `contacts.mcp.scm-platform.org` in `iac/` Terraform
- Google-managed SSL certificate (automatic via Cloud Run domain mapping)
- `config.yaml` update with domain configuration
- `BASE_URL` environment variable update to use the custom domain
- Terraform outputs for the new URL and OAuth2 callback URL
- Documentation of required OAuth2 callback URL change in Google Cloud Console
- Migration procedure for `project_id` changes

### 2.2 Out of Scope (Non-Goals)

- Application code changes (Go code remains unchanged)
- Load balancer setup (using Cloud Run domain mapping, not Global HTTPS LB)
- CDN or WAF configuration
- Multi-region deployment
- Separate shared-infra GCP project for DNS (deferred -- see Backlog)
- Wildcard domain mapping
- Automated OAuth2 callback URL updates in Google Cloud Console (manual step)

## 3. User Personas & Actors

| Actor | Description |
|-------|-------------|
| **Platform Operator** | Sebastien (or future ops) -- deploys and maintains infrastructure via Terraform. Runs `make plan` / `make deploy`. |
| **MCP Client** | AI assistants (e.g., Claude) that connect to the MCP server via the custom domain URL. |
| **OAuth2 End User** | Users who authenticate via the OAuth2 flow to grant Google Contacts access. |

## 4. Usage Scenarios

### SC-001: Deploy Custom Domain for the First Time

**Actor:** Platform Operator
**Preconditions:**
- `scm-platform.org` nameservers are delegated to Google Cloud DNS (already done)
- GCP project exists and Cloud Run service is deployed
- `init/` Terraform has been deployed at least once
- `config.yaml` has been updated with domain configuration

**Flow:**
1. Operator adds domain configuration to `config.yaml` (the `custom_domain` field)
2. Operator runs `make init-plan` to preview DNS zone creation
3. System shows the Cloud DNS zone resource to be created (or imported)
4. Operator runs `make init-deploy` to create the DNS zone
5. System creates the `scm-platform.org` zone in Cloud DNS and outputs the zone name
6. Operator runs `make plan` to preview domain mapping and DNS record changes
7. System shows: Cloud Run domain mapping, CNAME record, updated `BASE_URL` env var
8. Operator runs `make deploy` to apply changes
9. System creates the Cloud Run domain mapping (which provisions a Google-managed SSL certificate)
10. System creates the CNAME record `contacts.mcp.scm-platform.org` pointing to `ghs.googlehosted.com.`
11. System updates the Cloud Run service `BASE_URL` to `https://contacts.mcp.scm-platform.org`
12. System outputs the new MCP URL and the required OAuth2 callback URL
13. Operator manually updates the OAuth2 callback URL in Google Cloud Console (Credentials page)

**Postconditions:**
- `contacts.mcp.scm-platform.org` resolves to the Cloud Run service
- SSL certificate is provisioned (may take up to 24 hours for initial provisioning)
- OAuth2 callback URL uses the custom domain
- MCP clients can connect via `https://contacts.mcp.scm-platform.org`

**Exceptions:**
- EXC-001a: DNS zone already exists in another project -- Terraform import fails. Operator must manually import the zone or delete it from the other project first.
- EXC-001b: SSL certificate provisioning fails -- Cloud Run domain mapping shows status "pending". Operator must verify DNS propagation and wait (up to 24h). Terraform output will show the mapping status.
- EXC-001c: DNS propagation delay -- The CNAME record may take up to 48 hours to propagate globally. The old Cloud Run URL continues to work during this period.
- EXC-001d: Operator forgets to update OAuth2 callback URL -- OAuth2 flows will fail with a redirect_uri_mismatch error. The Terraform output explicitly reminds the operator.

**Cross-scenario notes:** This scenario is a prerequisite for SC-002 and SC-003.

### SC-002: Change project_id in config.yaml

**Actor:** Platform Operator
**Preconditions:**
- Custom domain is deployed and working (SC-001 completed)
- New GCP project exists with required APIs enabled
- Operator has updated `project_id` in `config.yaml`

**Flow:**
1. Operator updates `project_id` in `config.yaml`
2. Operator runs `make init-plan` in the new project context
3. System shows: new DNS zone to be created in the new project, new state backend, new service accounts
4. Operator runs `make init-deploy` against the new project
5. System creates the DNS zone `scm-platform.org` in the new project
6. **Operator updates nameservers at Squarespace** if the new zone has different NS records (system outputs the new NS records for verification)
7. Operator runs `make plan`
8. System shows: new Cloud Run service, domain mapping, CNAME record, all derived from config.yaml
9. Operator runs `make deploy`
10. System deploys Cloud Run, creates domain mapping, creates CNAME record -- all pointing to the new project
11. System outputs the new OAuth2 callback URL
12. Operator updates OAuth2 callback URL in Google Cloud Console

**Postconditions:**
- `contacts.mcp.scm-platform.org` resolves to the Cloud Run service in the new project
- All Terraform resources are project-agnostic (derived from `config.yaml`)
- Old project resources can be cleaned up separately

**Exceptions:**
- EXC-002a: Nameserver records differ between old and new zone -- Operator must update NS records at Squarespace. Terraform outputs the NS records for comparison. DNS propagation causes temporary downtime (up to 48h).
- EXC-002b: Old project DNS zone not destroyed before creating new one -- Both zones exist but only the one with active NS delegation receives queries. No conflict, but old resources should be cleaned up.
- EXC-002c: SSL certificate re-provisioning delay -- New domain mapping in new project requires a new SSL certificate. Same 24h provisioning window applies.

**Cross-scenario notes:** This is the most complex scenario. The spec must ensure all Terraform resources derive from `config.yaml` with zero hardcoded project references.

### SC-003: Add Another Subdomain for a Different Service

**Actor:** Platform Operator (of a different service project)
**Preconditions:**
- DNS zone for `scm-platform.org` exists in the init/ layer of some project
- The other service project has its own Terraform configuration

**Flow:**
1. Other service's Terraform references the shared DNS zone (via `google_dns_managed_zone` data source, cross-project)
2. Other service creates its own CNAME record (e.g., `email.mcp.scm-platform.org`)
3. Other service creates its own Cloud Run domain mapping

**Postconditions:**
- Multiple subdomains coexist under `scm-platform.org`
- Each service manages its own DNS record

**Exceptions:**
- EXC-003a: Cross-project DNS permissions not configured -- The other service's service account needs `roles/dns.admin` on the DNS zone's project. This must be configured manually or via the init/ Terraform of the zone-owning project.

**Cross-scenario notes:** This scenario is documented for future reference. The current spec only implements SC-001. The DNS zone design in init/ must not prevent this future use case.

### SC-004: MCP Client Connects via Custom Domain

**Actor:** MCP Client
**Preconditions:**
- Custom domain is deployed and SSL certificate is active
- DNS has propagated

**Flow:**
1. MCP Client sends an `initialize` request to `https://contacts.mcp.scm-platform.org/`
2. DNS resolves `contacts.mcp.scm-platform.org` to `ghs.googlehosted.com.`
3. Google's infrastructure routes the request to the Cloud Run service
4. Cloud Run responds with MCP session initialization
5. Client proceeds with normal MCP protocol (tool calls, etc.)

**Postconditions:**
- Session established via the custom domain
- All subsequent requests use the custom domain URL

**Exceptions:**
- EXC-004a: SSL certificate not yet provisioned -- Client receives a TLS error. Must wait for certificate provisioning.
- EXC-004b: DNS not yet propagated -- Client receives DNS resolution failure. Must wait for propagation.

### SC-005: OAuth2 Flow via Custom Domain

**Actor:** OAuth2 End User + MCP Client
**Preconditions:**
- Custom domain is deployed with valid SSL
- OAuth2 callback URL has been updated in Google Cloud Console
- MCP Client has initiated an OAuth2 authorization flow

**Flow:**
1. MCP Client discovers auth metadata at `https://contacts.mcp.scm-platform.org/.well-known/oauth-authorization-server`
2. Metadata returns endpoints using the custom domain as base URL
3. Client redirects user to `https://contacts.mcp.scm-platform.org/oauth/authorize`
4. Server redirects to Google OAuth consent screen
5. User grants consent
6. Google redirects to `https://contacts.mcp.scm-platform.org/oauth/callback`
7. Server exchanges code and redirects back to client with authorization code
8. Client exchanges code for tokens at `https://contacts.mcp.scm-platform.org/oauth/token`

**Postconditions:**
- Client has valid access and refresh tokens
- All OAuth2 URLs used the custom domain

**Exceptions:**
- EXC-005a: OAuth2 callback URL not updated in Google Cloud Console -- Google returns `redirect_uri_mismatch` error at step 6. Operator must update the callback URL.
- EXC-005b: Old Cloud Run URL still in OAuth2 credentials -- Same as EXC-005a. The old URL will stop working as `BASE_URL` because the server now constructs callback URLs using the custom domain.

## 5. Functional Requirements

### FR-001: Cloud DNS Zone in init/

- **Description:** The `init/` Terraform layer must create a Cloud DNS managed zone for `scm-platform.org`.
- **Inputs:** Domain name derived from `config.yaml` (the base domain portion of `custom_domain`)
- **Outputs:** Zone name, zone DNS name, nameserver records (as Terraform outputs)
- **Business Rules:**
  - The zone must be created in the same GCP project as other init/ resources
  - The zone name must be deterministic and derived from the domain (e.g., `scm-platform-org`)
  - The zone must output its nameserver records so the operator can verify Squarespace delegation
  - The zone must have labels consistent with existing resources (`environment`, `managed_by`)
- **Priority:** Must-have

### FR-002: Cloud Run Domain Mapping in iac/

- **Description:** The `iac/` Terraform layer must create a `google_cloud_run_domain_mapping` resource that maps the custom domain to the Cloud Run service.
- **Inputs:** Custom domain from `config.yaml`, Cloud Run service name
- **Outputs:** Domain mapping status, mapped URL
- **Business Rules:**
  - The domain mapping must reference the Cloud Run service by name
  - The domain mapping must be in the same region as the Cloud Run service
  - The domain mapping must use the `route_name` from the Cloud Run service
  - The mapping must depend on the Cloud Run service being deployed
- **Priority:** Must-have

### FR-003: DNS CNAME Record in iac/

- **Description:** The `iac/` Terraform layer must create a CNAME record in the Cloud DNS zone that points the custom domain to `ghs.googlehosted.com.` (the Google-hosted services endpoint for Cloud Run domain mapping).
- **Inputs:** Custom domain from `config.yaml`, DNS zone reference from init/
- **Outputs:** CNAME record FQDN
- **Business Rules:**
  - The CNAME record must point to `ghs.googlehosted.com.` (required by Cloud Run domain mapping)
  - The record TTL must be 300 seconds (5 minutes) for reasonable propagation/update speed
  - The record must be created in the DNS zone managed by init/
  - The zone must be referenced via a `google_dns_managed_zone` data source (not hardcoded)
- **Priority:** Must-have

### FR-004: config.yaml Domain Configuration

- **Description:** The `config.yaml` file must include a `custom_domain` field that specifies the full custom domain for the MCP server.
- **Inputs:** Operator-provided domain string
- **Outputs:** Used by Terraform locals to derive DNS zone, CNAME record, domain mapping, and BASE_URL
- **Business Rules:**
  - The field must be at the top level or under `gcp.resources.cloud_run`
  - The base domain (e.g., `scm-platform.org`) must be extractable programmatically for DNS zone creation
  - If the field is empty or absent, no domain mapping or DNS resources must be created (backward compatible)
  - The domain must be used as-is for the `BASE_URL` environment variable (prefixed with `https://`)
- **Priority:** Must-have

### FR-005: BASE_URL Environment Variable Update

- **Description:** The Cloud Run service `BASE_URL` environment variable must use the custom domain when configured, falling back to the auto-generated Cloud Run URL when no custom domain is set.
- **Inputs:** `custom_domain` from config.yaml, Cloud Run auto-generated URL
- **Outputs:** `BASE_URL` env var value on the Cloud Run service
- **Business Rules:**
  - When `custom_domain` is set: `BASE_URL = "https://${custom_domain}"`
  - When `custom_domain` is empty/absent: `BASE_URL` falls back to the current auto-generated URL pattern
  - This directly affects the OAuth2 callback URL constructed by the application (`${BASE_URL}/oauth/callback`)
- **Priority:** Must-have

### FR-006: Terraform Outputs for OAuth2 Callback URL

- **Description:** Terraform must output the OAuth2 callback URL that the operator needs to configure in Google Cloud Console.
- **Inputs:** BASE_URL value
- **Outputs:** Full callback URL string (e.g., `https://contacts.mcp.scm-platform.org/oauth/callback`)
- **Business Rules:**
  - Output must be named descriptively (e.g., `oauth2_callback_url`)
  - Output must include a description explaining that this URL must be added to the Google Cloud Console OAuth2 credentials
  - An additional output `oauth2_action_required` must display a human-readable message reminding the operator to update the callback URL
- **Priority:** Must-have

### FR-007: DNS Zone Nameserver Output in init/

- **Description:** The init/ Terraform must output the nameserver records of the created DNS zone so the operator can verify they match the Squarespace delegation.
- **Inputs:** Created DNS zone
- **Outputs:** List of nameserver hostnames
- **Business Rules:**
  - Output must be named `dns_zone_nameservers`
  - Output must include a description explaining their purpose
  - If nameservers differ from the current Squarespace delegation, the operator must update Squarespace manually
- **Priority:** Must-have

### FR-008: Conditional Resource Creation

- **Description:** All domain-related resources (DNS zone, domain mapping, CNAME record) must be conditionally created based on whether `custom_domain` is configured in `config.yaml`.
- **Inputs:** `custom_domain` field presence/value in config.yaml
- **Outputs:** Resources created or not
- **Business Rules:**
  - Use `count` or `for_each` with a conditional based on `custom_domain != ""`
  - When `custom_domain` is absent or empty, Terraform must not create any domain-related resources
  - This ensures backward compatibility -- existing deployments without a custom domain continue to work
- **Priority:** Must-have

### FR-009: GCP API Enablement for Cloud DNS

- **Description:** The `dns.googleapis.com` API must be enabled in the GCP project to use Cloud DNS.
- **Inputs:** GCP project ID
- **Outputs:** API enabled
- **Business Rules:**
  - Add `dns.googleapis.com` to the `services` list in `config.yaml`
  - The init/ Terraform already enables APIs from this list via `google_project_service`
- **Priority:** Must-have

## 6. Non-Functional Requirements

### 6.1 Performance

- DNS TTL of 300 seconds balances propagation speed with caching efficiency
- Cloud Run domain mapping adds no measurable latency (Google-managed routing)
- SSL termination is handled by Google's infrastructure, no performance impact on the application

### 6.2 Security

- Google-managed SSL certificate provides TLS 1.2+ encryption
- No changes to application-level authentication (OAuth2, API keys)
- The old auto-generated Cloud Run URL will continue to work (Cloud Run does not disable it) -- this is acceptable as the service already has its own auth layer

### 6.3 Reliability

- SSL certificate provisioning may take up to 24 hours -- during this period, the custom domain is not usable via HTTPS
- The old Cloud Run URL remains functional throughout the transition
- DNS propagation may take up to 48 hours globally
- **Rollback:** Removing `custom_domain` from config.yaml and running `make deploy` will destroy the domain mapping and CNAME record, reverting to the auto-generated URL

### 6.4 Observability

- Terraform outputs provide immediate feedback on deployed URLs and required actions
- Cloud Run domain mapping status is visible in Cloud Console and via `gcloud run domain-mappings list`
- DNS record correctness can be verified with `dig contacts.mcp.scm-platform.org`

### 6.5 Deployment

- All changes deployed via existing `make init-deploy` and `make deploy` workflow
- No new CI/CD pipeline required
- No changes to Docker build or application deployment

## 7. Data Model

No application data model changes. This specification affects only Terraform resources.

### 7.1 Terraform Resource Graph

```
init/ layer:
  google_dns_managed_zone.domain_zone (NEW)
    -> outputs: zone_name, dns_zone_nameservers

iac/ layer:
  data.google_dns_managed_zone.domain_zone (references init/ zone)
    -> google_dns_record_set.mcp_cname (NEW)
    -> google_cloud_run_domain_mapping.mcp (NEW)
       -> depends_on: google_cloud_run_v2_service.mcp

  google_cloud_run_v2_service.mcp (MODIFIED: BASE_URL env var)
```

### 7.2 config.yaml Changes

```yaml
# New field under gcp.resources.cloud_run:
gcp:
  resources:
    cloud_run:
      custom_domain: contacts.mcp.scm-platform.org  # NEW
  services:
    - dns.googleapis.com  # NEW (add to existing list)
```

## 8. Documentation Requirements

All documentation listed below must be updated as part of this change.

### 8.1 CLAUDE.md

- Update project structure to mention DNS/domain Terraform files
- Add `custom_domain` to the configuration reference
- Add note about OAuth2 callback URL update requirement

### 8.2 .agent_docs/terraform.md

- Document the new `init/dns.tf` file and its purpose
- Document the new domain mapping resources in `iac/`
- Document the `config.yaml` domain configuration
- Add the project_id migration procedure
- Update the deployment workflow section

### 8.3 .agent_docs/authentication.md

- Document the relationship between `custom_domain`, `BASE_URL`, and OAuth2 callback URL
- Document the manual step of updating the Google Cloud Console OAuth2 credentials

## 9. Traceability Matrix

| Scenario | Functional Req | E2E Tests (Happy) | E2E Tests (Failure) | E2E Tests (Edge) |
|----------|---------------|-------------------|---------------------|-------------------|
| SC-001 | FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009 | E2E-001, E2E-002, E2E-003, E2E-004 | E2E-009, E2E-010, E2E-011 | E2E-015, E2E-016, E2E-017 |
| SC-002 | FR-001, FR-004, FR-005, FR-007, FR-008 | E2E-005, E2E-006 | E2E-012 | E2E-018 |
| SC-003 | FR-001, FR-003, FR-007 | E2E-007 | E2E-013 | E2E-019 |
| SC-004 | FR-002, FR-003, FR-005 | E2E-008 | E2E-014 | E2E-020 |
| SC-005 | FR-005, FR-006 | E2E-004 | E2E-011 | E2E-017 |

## 10. End-to-End Test Suite

All tests are Terraform plan/apply validation tests. They verify infrastructure correctness, not application behavior (application behavior is unchanged).

### 10.1 Test Summary

| Test ID | Category | Scenario | FR refs | Priority |
|---------|----------|----------|---------|----------|
| E2E-001 | Core Journey | SC-001 | FR-001 | Critical |
| E2E-002 | Core Journey | SC-001 | FR-002, FR-003 | Critical |
| E2E-003 | Core Journey | SC-001 | FR-004, FR-005 | Critical |
| E2E-004 | Core Journey | SC-001, SC-005 | FR-006 | Critical |
| E2E-005 | Core Journey | SC-002 | FR-001, FR-004 | High |
| E2E-006 | Core Journey | SC-002 | FR-005, FR-007 | High |
| E2E-007 | Feature | SC-003 | FR-001, FR-003 | Medium |
| E2E-008 | Feature | SC-004 | FR-002, FR-003, FR-005 | High |
| E2E-009 | Error Handling | SC-001 | FR-001 | High |
| E2E-010 | Error Handling | SC-001 | FR-002 | High |
| E2E-011 | Error Handling | SC-001, SC-005 | FR-005, FR-006 | High |
| E2E-012 | Error Handling | SC-002 | FR-001, FR-007 | Medium |
| E2E-013 | Error Handling | SC-003 | FR-001, FR-003 | Medium |
| E2E-014 | Error Handling | SC-004 | FR-002, FR-003 | Medium |
| E2E-015 | Edge Case | SC-001 | FR-004, FR-008 | High |
| E2E-016 | Edge Case | SC-001 | FR-001, FR-009 | Medium |
| E2E-017 | Edge Case | SC-001, SC-005 | FR-005, FR-006 | Medium |
| E2E-018 | Edge Case | SC-002 | FR-004, FR-005 | Medium |
| E2E-019 | Edge Case | SC-003 | FR-001 | Low |
| E2E-020 | Edge Case | SC-004 | FR-002 | Low |

### 10.2 Test Specifications

#### E2E-001: DNS Zone Created in init/ with Correct Configuration

- **Category:** Core Journey
- **Scenario:** SC-001 -- Deploy Custom Domain for the First Time
- **Requirements:** FR-001
- **Preconditions:** config.yaml has `custom_domain: contacts.mcp.scm-platform.org`
- **Steps:**
  - Given a config.yaml with `custom_domain` set to `contacts.mcp.scm-platform.org`
  - When `terraform plan` is run in the `init/` directory
  - Then the plan must include a `google_dns_managed_zone` resource with:
    - DNS name: `scm-platform.org.`
    - Zone name derived from the domain (e.g., `scm-platform-org`)
    - Labels: `environment = prd`, `managed_by = terraform`
  - And the plan must include an output `dns_zone_nameservers`
- **Priority:** Critical

#### E2E-002: Cloud Run Domain Mapping and CNAME Record Created in iac/

- **Category:** Core Journey
- **Scenario:** SC-001 -- Deploy Custom Domain for the First Time
- **Requirements:** FR-002, FR-003
- **Preconditions:** init/ has been deployed (DNS zone exists)
- **Steps:**
  - Given init/ is deployed with the DNS zone for `scm-platform.org`
  - When `terraform plan` is run in the `iac/` directory
  - Then the plan must include a `google_cloud_run_domain_mapping` resource with:
    - Name: `contacts.mcp.scm-platform.org`
    - Location: `europe-west1`
    - Reference to `google_cloud_run_v2_service.mcp`
  - And the plan must include a `google_dns_record_set` resource with:
    - Name: `contacts.mcp.scm-platform.org.`
    - Type: `CNAME`
    - TTL: `300`
    - Rrdatas: `["ghs.googlehosted.com."]`
- **Priority:** Critical

#### E2E-003: BASE_URL Uses Custom Domain

- **Category:** Core Journey
- **Scenario:** SC-001 -- Deploy Custom Domain for the First Time
- **Requirements:** FR-004, FR-005
- **Preconditions:** config.yaml has `custom_domain` set
- **Steps:**
  - Given config.yaml with `custom_domain: contacts.mcp.scm-platform.org`
  - When `terraform plan` is run in `iac/`
  - Then the Cloud Run service `BASE_URL` env var must be `https://contacts.mcp.scm-platform.org`
  - And the env var must NOT contain the auto-generated Cloud Run URL
- **Priority:** Critical

#### E2E-004: Terraform Outputs Include OAuth2 Callback URL

- **Category:** Core Journey
- **Scenario:** SC-001, SC-005 -- OAuth2 callback URL documentation
- **Requirements:** FR-006
- **Preconditions:** Deployment completed
- **Steps:**
  - Given a successful `terraform apply` in `iac/`
  - When querying `terraform output oauth2_callback_url`
  - Then the output must be `https://contacts.mcp.scm-platform.org/oauth/callback`
  - And querying `terraform output oauth2_action_required` must return a human-readable message containing "Google Cloud Console" and "OAuth2" and "callback"
- **Priority:** Critical

#### E2E-005: project_id Change Regenerates All Resources

- **Category:** Core Journey
- **Scenario:** SC-002 -- Change project_id in config.yaml
- **Requirements:** FR-001, FR-004
- **Preconditions:** Custom domain deployed in project A, config.yaml updated to project B
- **Steps:**
  - Given config.yaml with a new `project_id`
  - When `terraform plan` is run in `init/` targeting the new project
  - Then the plan must include a new `google_dns_managed_zone` for the new project
  - And the plan must output the new zone's nameserver records
- **Priority:** High

#### E2E-006: project_id Change Updates BASE_URL and Outputs

- **Category:** Core Journey
- **Scenario:** SC-002 -- Change project_id in config.yaml
- **Requirements:** FR-005, FR-007
- **Preconditions:** config.yaml with new project_id, init/ deployed to new project
- **Steps:**
  - Given a new project_id in config.yaml and init/ deployed
  - When `terraform plan` is run in `iac/` targeting the new project
  - Then `BASE_URL` must still be `https://contacts.mcp.scm-platform.org` (unchanged, domain-based)
  - And the DNS CNAME record must still point to `ghs.googlehosted.com.`
  - And the `dns_zone_nameservers` output from init/ must list the new zone's NS records
- **Priority:** High

#### E2E-007: DNS Zone Supports Multiple Subdomains

- **Category:** Feature
- **Scenario:** SC-003 -- Add Another Subdomain for a Different Service
- **Requirements:** FR-001, FR-003
- **Preconditions:** DNS zone exists in init/
- **Steps:**
  - Given the `scm-platform.org` DNS zone created by init/
  - When a second Terraform configuration adds a CNAME record for `email.mcp.scm-platform.org`
  - Then the DNS zone must accept the new record without conflict
  - And both `contacts.mcp.scm-platform.org` and `email.mcp.scm-platform.org` must coexist
- **Priority:** Medium

#### E2E-008: MCP Client Connectivity via Custom Domain

- **Category:** Feature
- **Scenario:** SC-004 -- MCP Client Connects via Custom Domain
- **Requirements:** FR-002, FR-003, FR-005
- **Preconditions:** Domain mapping deployed, SSL provisioned, DNS propagated
- **Steps:**
  - Given the custom domain is fully deployed and SSL active
  - When an HTTP POST is sent to `https://contacts.mcp.scm-platform.org/` with an MCP initialize request
  - Then the response must be a valid MCP initialization response (status 200, JSON-RPC)
  - And the `Mcp-Session-Id` header must be present
- **Priority:** High

#### E2E-009: Terraform Plan Fails Gracefully When DNS Zone Already Exists

- **Category:** Error Handling
- **Scenario:** SC-001 -- Deploy Custom Domain for the First Time
- **Requirements:** FR-001
- **Preconditions:** DNS zone for `scm-platform.org` already exists in a different project
- **Steps:**
  - Given a DNS zone for `scm-platform.org` exists in another GCP project
  - When `terraform apply` is run in `init/`
  - Then Terraform must fail with a clear error indicating the zone already exists
  - And the error must not corrupt the Terraform state
  - And the operator must be able to resolve by importing or removing the conflicting zone
- **Priority:** High

#### E2E-010: Domain Mapping Fails When Cloud Run Service Does Not Exist

- **Category:** Error Handling
- **Scenario:** SC-001 -- Deploy Custom Domain for the First Time
- **Requirements:** FR-002
- **Preconditions:** Cloud Run service not yet deployed, domain mapping attempted
- **Steps:**
  - Given the Cloud Run service does not exist
  - When `terraform apply` is run in `iac/` with domain mapping enabled
  - Then the domain mapping must fail with a dependency error
  - And the `depends_on` relationship must ensure correct ordering
- **Priority:** High

#### E2E-011: Terraform Output Warns About OAuth2 Callback When Domain Changes

- **Category:** Error Handling
- **Scenario:** SC-001, SC-005 -- OAuth2 callback mismatch
- **Requirements:** FR-005, FR-006
- **Preconditions:** Custom domain deployed
- **Steps:**
  - Given the custom domain is deployed
  - When querying `terraform output`
  - Then the `oauth2_action_required` output must contain the exact callback URL to configure
  - And the message must mention that failure to update will cause `redirect_uri_mismatch` errors
- **Priority:** High

#### E2E-012: project_id Change Produces Different Nameservers Warning

- **Category:** Error Handling
- **Scenario:** SC-002 -- Change project_id in config.yaml
- **Requirements:** FR-001, FR-007
- **Preconditions:** DNS zone deployed in old project, new project_id in config.yaml
- **Steps:**
  - Given a new project_id and init/ re-deployed
  - When querying `terraform output dns_zone_nameservers` from init/
  - Then the output must list the nameservers for the new zone
  - And the operator must compare these with the Squarespace delegation
  - And if they differ, the operator must update Squarespace
- **Priority:** Medium

#### E2E-013: Cross-Project DNS Record Creation Without Permissions

- **Category:** Error Handling
- **Scenario:** SC-003 -- Add Another Subdomain for a Different Service
- **Requirements:** FR-001, FR-003
- **Preconditions:** DNS zone in project A, other service in project B without DNS permissions
- **Steps:**
  - Given a DNS zone in project A
  - When project B's Terraform tries to create a CNAME record in project A's zone without `roles/dns.admin`
  - Then Terraform must fail with a permission error
  - And the error must clearly indicate the required IAM role
- **Priority:** Medium

#### E2E-014: Custom Domain Inaccessible Before SSL Provisioning

- **Category:** Error Handling
- **Scenario:** SC-004 -- MCP Client Connects via Custom Domain
- **Requirements:** FR-002, FR-003
- **Preconditions:** Domain mapping created, SSL not yet provisioned
- **Steps:**
  - Given the domain mapping is created but SSL certificate is pending
  - When an HTTPS request is sent to `https://contacts.mcp.scm-platform.org/`
  - Then the request must fail with a TLS error (certificate not valid)
  - And the old Cloud Run URL must continue to work
- **Priority:** Medium

#### E2E-015: No Domain Resources Created When custom_domain is Empty

- **Category:** Edge Case
- **Scenario:** SC-001 -- Backward compatibility
- **Requirements:** FR-004, FR-008
- **Preconditions:** config.yaml with `custom_domain: ""` or field absent
- **Steps:**
  - Given config.yaml without a `custom_domain` field (or empty string)
  - When `terraform plan` is run in both `init/` and `iac/`
  - Then init/ must NOT create a `google_dns_managed_zone` resource
  - And iac/ must NOT create a `google_cloud_run_domain_mapping` resource
  - And iac/ must NOT create a `google_dns_record_set` resource
  - And `BASE_URL` must fall back to the auto-generated Cloud Run URL
- **Priority:** High

#### E2E-016: dns.googleapis.com API Enabled Before DNS Zone Creation

- **Category:** Edge Case
- **Scenario:** SC-001 -- API dependency
- **Requirements:** FR-001, FR-009
- **Preconditions:** Fresh GCP project, dns.googleapis.com not enabled
- **Steps:**
  - Given a GCP project without `dns.googleapis.com` enabled
  - When `terraform apply` is run in `init/`
  - Then the API must be enabled before the DNS zone creation
  - And the `google_dns_managed_zone` must have a `depends_on` to the API enablement resource
- **Priority:** Medium

#### E2E-017: BASE_URL Consistency Between Terraform Output and Application Behavior

- **Category:** Edge Case
- **Scenario:** SC-001, SC-005 -- BASE_URL and OAuth2 alignment
- **Requirements:** FR-005, FR-006
- **Preconditions:** Custom domain deployed
- **Steps:**
  - Given the deployed Cloud Run service
  - When the application starts and constructs OAuth2 metadata
  - Then the `authorization_endpoint` in `/.well-known/oauth-authorization-server` must use the custom domain
  - And the `resource` in `/.well-known/oauth-protected-resource` must use the custom domain
  - And these must match the Terraform output `oauth2_callback_url` base
- **Priority:** Medium

#### E2E-018: Changing custom_domain Value Updates All Resources

- **Category:** Edge Case
- **Scenario:** SC-002 -- Domain value change
- **Requirements:** FR-004, FR-005
- **Preconditions:** custom_domain set to one value, then changed to another
- **Steps:**
  - Given config.yaml initially with `custom_domain: contacts.mcp.scm-platform.org`
  - When `custom_domain` is changed to `api.mcp.scm-platform.org`
  - Then `terraform plan` must show:
    - Domain mapping destroyed and recreated with new name
    - CNAME record updated to new subdomain
    - `BASE_URL` updated to `https://api.mcp.scm-platform.org`
    - `oauth2_callback_url` output updated
- **Priority:** Medium

#### E2E-019: DNS Zone Allows Records from Multiple Terraform States

- **Category:** Edge Case
- **Scenario:** SC-003 -- Shared zone usage
- **Requirements:** FR-001
- **Preconditions:** DNS zone created by init/
- **Steps:**
  - Given the DNS zone created and managed by init/ Terraform state
  - When iac/ Terraform (separate state) creates a record in the same zone via data source
  - Then there must be no state conflict
  - And the record must be manageable independently from the zone
- **Priority:** Low

#### E2E-020: Domain Mapping Status Reflects Provisioning State

- **Category:** Edge Case
- **Scenario:** SC-004 -- Mapping status monitoring
- **Requirements:** FR-002
- **Preconditions:** Domain mapping just created
- **Steps:**
  - Given a freshly created domain mapping
  - When querying the domain mapping status via `gcloud run domain-mappings describe`
  - Then the status must indicate either "pending" (certificate provisioning) or "active"
  - And Terraform must not report errors for a "pending" status
- **Priority:** Low

## 11. Open Questions & TBDs

| ID | Question | Impact | Owner |
|----|----------|--------|-------|
| TBD-001 | Does the existing Cloud DNS zone for `scm-platform.org` already exist in the target GCP project, or must it be created? If it exists, it may need to be imported into Terraform state rather than created. | FR-001, SC-001 | Platform Operator -- verify with `gcloud dns managed-zones list --project=project-fb127223-bfef-43d1-94e` before first deployment |
| TBD-002 | Google Cloud Run domain mapping uses the v1 API (`google_cloud_run_domain_mapping`), not v2. The existing service uses `google_cloud_run_v2_service`. Verify compatibility between v1 domain mapping and v2 service resource. | FR-002 | Implementer -- test with `terraform plan` |
| TBD-003 | After project_id migration, do the new Cloud DNS zone nameservers always match the old ones? Google Cloud DNS does not guarantee the same NS records for newly created zones. If they differ, Squarespace delegation must be updated. | SC-002, EXC-002a | Platform Operator -- verify empirically or accept the risk |

## 12. Glossary

| Term | Definition |
|------|------------|
| **Cloud Run domain mapping** | A GCP resource (`google_cloud_run_domain_mapping`) that associates a custom domain with a Cloud Run service and provisions a Google-managed SSL certificate. |
| **Cloud DNS managed zone** | A GCP resource (`google_dns_managed_zone`) representing a DNS zone (e.g., `scm-platform.org`) hosted in Google Cloud DNS. |
| **CNAME record** | A DNS record that maps an alias (e.g., `contacts.mcp.scm-platform.org`) to a canonical name (e.g., `ghs.googlehosted.com`). |
| **ghs.googlehosted.com** | Google Hosted Services endpoint. Cloud Run domain mappings require a CNAME pointing to this address. |
| **BASE_URL** | An environment variable on the Cloud Run service that the MCP application uses to construct OAuth2 endpoint URLs and metadata. |
| **init/ layer** | The Terraform directory for foundational infrastructure (state backend, service accounts, DNS zones). Deployed once, rarely changes. |
| **iac/ layer** | The Terraform directory for application infrastructure (Cloud Run, domain mapping, DNS records). Deployed on every change. |
| **Nameserver delegation** | The configuration at a domain registrar (Squarespace) that points a domain's DNS resolution to specific nameservers (Google Cloud DNS). |

## 13. Implementation Reference

### 13.1 New Terraform Files

| File | Layer | Purpose |
|------|-------|---------|
| `init/dns.tf` | init/ | Cloud DNS managed zone for `scm-platform.org` |
| `iac/dns-domain.tf` | iac/ | Cloud Run domain mapping + CNAME record |

### 13.2 Modified Files

| File | Change |
|------|--------|
| `config.yaml` | Add `custom_domain` field under `gcp.resources.cloud_run`, add `dns.googleapis.com` to services |
| `iac/workload-mcp.tf` | Update `BASE_URL` env var to use custom domain when available |
| `iac/local.tf` | Add locals for custom domain parsing (base domain extraction, zone name derivation) |
| `init/local.tf` | Add locals for domain zone configuration |
| `init/services.tf` | Already handles services list; `dns.googleapis.com` will be picked up from config.yaml |
| `CLAUDE.md` | Update project structure and configuration reference |
| `.agent_docs/terraform.md` | Add DNS and domain mapping documentation |
| `.agent_docs/authentication.md` | Add OAuth2 callback URL update procedure |

### 13.3 config.yaml Target State

```yaml
gcp:
  services:
    - run.googleapis.com
    - firestore.googleapis.com
    - secretmanager.googleapis.com
    - cloudbuild.googleapis.com
    - artifactregistry.googleapis.com
    - cloudresourcemanager.googleapis.com
    - iam.googleapis.com
    - dns.googleapis.com                    # NEW
  resources:
    cloud_run:
      name: google-contacts-mcp
      region: europe-west1
      cpu: "1"
      memory: 256Mi
      min_instances: 0
      max_instances: 3
      allow_unauthenticated: true
      custom_domain: contacts.mcp.scm-platform.org  # NEW
```

### 13.4 Post-Deployment: OAuth2 Callback URL Update

After running `make deploy`, the operator must:

1. Go to [Google Cloud Console > APIs & Services > Credentials](https://console.cloud.google.com/apis/credentials)
2. Open the OAuth 2.0 Client ID used by the MCP server
3. Under "Authorized redirect URIs", **add**: `https://contacts.mcp.scm-platform.org/oauth/callback`
4. Optionally **remove** the old Cloud Run auto-generated URL callback (e.g., `https://google-contacts-mcp-XXXXXXX.europe-west1.run.app/oauth/callback`)
5. Save

**Failure to do this will cause `redirect_uri_mismatch` errors on all OAuth2 flows.**

### 13.5 Project Migration Procedure

When changing `project_id` in config.yaml:

1. Update `project_id` in `config.yaml`
2. Run `make init-plan` against the new project -- review new DNS zone, service accounts, state backend
3. Run `make init-deploy` -- creates all foundational resources in the new project
4. **Check nameservers**: compare `terraform output dns_zone_nameservers` (from init/) with current Squarespace delegation
   - If they match: no action needed at Squarespace
   - If they differ: update nameservers at Squarespace (expect up to 48h propagation)
5. Run `make plan` -- review Cloud Run, domain mapping, CNAME, all in new project
6. Run `make deploy` -- deploys everything to the new project
7. Update OAuth2 callback URL in Google Cloud Console if the credentials are project-scoped
8. Verify with `dig contacts.mcp.scm-platform.org` and test MCP connectivity
9. Clean up old project resources (optional, via `make undeploy` in old project context)
