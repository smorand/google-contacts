# Terraform Infrastructure

## Structure

```
google-contacts/
├── config.yaml      # Single source of configuration
├── init/            # One-time setup (state backend, service accounts)
│   ├── provider.tf
│   ├── local.tf
│   ├── state-backend.tf
│   ├── service-accounts.tf
│   └── services.tf
└── iac/             # Main infrastructure
    ├── provider.tf.template
    ├── provider.tf  # Generated after init-deploy
    ├── local.tf
    ├── docker.tf    # Docker build via kreuzwerker/docker
    ├── workload-mcp.tf
    ├── database-firestore.tf
    └── secrets.tf
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `init-plan` | Plan initialization resources |
| `init-deploy` | Deploy initialization |
| `init-destroy` | Destroy initialization (DANGEROUS!) |
| `plan` | Plan main infrastructure |
| `deploy` | Deploy (builds Docker + Cloud Run) |
| `undeploy` | Destroy main infrastructure |
| `update-backend` | Regenerate iac/provider.tf |

## Deployment Workflow

**First Time:**
```bash
make init-plan    # Review init resources
make init-deploy  # Create state backend
make plan         # Review main infra
make deploy       # Deploy everything
```

**Regular Updates:**
```bash
make plan    # Review changes
make deploy  # Apply
```

## Docker Build (iac/docker.tf)

Uses [kreuzwerker/docker](https://registry.terraform.io/providers/kreuzwerker/docker/latest) provider.

**Build triggers:**
- `Dockerfile`
- `go.mod`
- `go.sum`
- `cmd/google-contacts/main.go`

**Requirements:**
- Local Docker daemon running
- `gcloud auth configure-docker europe-west1-docker.pkg.dev`

## Cloud Run (iac/workload-mcp.tf)

**Resources:**
- `google_artifact_registry_repository.mcp` - Docker repo
- `google_cloud_run_v2_service.mcp` - MCP service
- IAM bindings for Firestore and Secret Manager

**Environment Variables:**
- `FIRESTORE_PROJECT` - GCP project
- `PORT` - 8080
- `ENVIRONMENT` - prd/dev

**Service Account Permissions:**
- `roles/datastore.user`
- `roles/secretmanager.secretAccessor`

## Firestore (iac/database-firestore.tf)

**Resources:**
- `google_firestore_database.main` - Native mode in eur3
- Indexes for `api_keys` collection

**Collection:** `api_keys`
- Document ID = API key (UUID v4)
- Fields: refresh_token, access_token, user_email, created_at, last_used

## Secret Manager (iac/secrets.tf)

Stores OAuth credentials. Secret version created MANUALLY:

```bash
gcloud secrets versions add scm-pwd-oauth-creds \
  --data-file=$HOME/.credentials/scm-pwd.json \
  --project=scmgcontacts-mcp-prd
```

## Configuration (config.yaml)

```yaml
prefix: scmgcontacts
project_name: google-contacts-mcp
env: prd

gcp:
  project_id: scmgcontacts-mcp-prd
  location: europe-west1
  services:
    - run.googleapis.com
    - firestore.googleapis.com
    - secretmanager.googleapis.com
  resources:
    cloud_run:
      name: google-contacts-mcp
      region: europe-west1
      cpu: "1"
      memory: 256Mi
      min_instances: 0
      max_instances: 3
    artifact_registry:
      name: google-contacts
      format: DOCKER
    firestore:
      database_id: "(default)"
      location_id: eur3

secrets:
  oauth_credentials: scm-pwd-oauth-creds
```

## File Organization Rules

- **init/**: One-time setup (state backend, service accounts, API enablement)
- **iac/**: Application infrastructure
- Resource files named by feature: `workload-mcp.tf`, `database-firestore.tf`
- Structure per file: locals → resources → permissions → outputs
- NO separate `output.tf` - outputs inline

## Notes for AI

- Always run `make plan` before `make deploy`
- Never commit `.terraform/` or `*.tfstate`
- Use `config.yaml` as single source of truth
- Firestore collections created automatically on first write
- API keys use document ID for O(1) lookup
