# CONFIGURE_YOUR_GCP_PROJECT_ID

Google Contacts MCP Server - Cloud Run deployment with OAuth API Key management

## Project Information

- **Environment:** prod
- **Cloud Provider:** GCP
- **Location:** europe-west1
- **Prefix:** scmgcontacts

## Infrastructure

This project uses Terraform for infrastructure as code (IaC) with the following structure:

```
├── iac.yaml              # Configuration file (single source of truth)
├── Makefile              # Build and deployment targets
├── init/                 # Initialization terraform (backend, state, service accounts)
└── iac/                  # Main infrastructure (application resources)
```

## Prerequisites

- Terraform >= 1.0
- Cloud provider CLI installed and configured:
  - GCP: `gcloud` CLI with authenticated account
  - AWS: `aws` CLI with configured credentials
  - Azure: `az` CLI with logged-in account
- `make` command available

## Getting Started

### First-Time Setup

1. **Review configuration**
   ```bash
   cat iac.yaml
   ```

2. **Initialize backend (first time only)**
   ```bash
   make init-plan     # Preview initialization changes
   make init-deploy   # Create state backend and service accounts
   ```

3. **Update backend configuration**
   After `init-deploy`, update `iac/provider.tf` with the backend configuration from the init outputs.

4. **Deploy infrastructure**
   ```bash
   make plan     # Preview infrastructure changes
   make deploy   # Deploy infrastructure
   ```

## Common Operations

### Planning Changes
Before deploying, always plan to see what will change:
```bash
make plan
```

### Deploying Changes
Apply terraform changes:
```bash
make deploy
```

### Destroying Infrastructure
To destroy all infrastructure (careful!):
```bash
make undeploy
```

### Viewing Outputs
After deployment, view terraform outputs:
```bash
cd iac && terraform output
```

## Makefile Targets

### Build and CLI Targets

| Target | Description |
|--------|-------------|
| `make build` | Build CLI binary for current platform |
| `make build-all` | Build for all platforms |
| `make test` | Run unit tests |
| `make install` | Install CLI to /usr/local/bin |

### Docker and Cloud Run Targets

| Target | Description |
|--------|-------------|
| `make docker-build` | Build container image locally |
| `make docker-push` | Push container to Artifact Registry |
| `make cloud-run-deploy` | Full deployment (build + push + deploy) |

### Terraform Targets

| Target | Description |
|--------|-------------|
| `make plan` | Plan main infrastructure changes |
| `make deploy` | Deploy main infrastructure |
| `make undeploy` | Destroy main infrastructure |
| `make init-plan` | Plan initialization (backend, state) |
| `make init-deploy` | Deploy initialization |
| `make init-destroy` | Destroy initialization (dangerous!) |

## Docker Deployment

The project includes a Dockerfile for containerized deployment:

```bash
# Build the container image
make docker-build

# Run locally
docker run -p 8080:8080 google-contacts-mcp:latest

# Deploy to Cloud Run (requires GCP credentials)
make cloud-run-deploy
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PORT` | Server listening port (default: 8080) |
| `FIRESTORE_PROJECT` | GCP project for API key validation |

## MCP Server

The project includes an MCP (Model Context Protocol) server that enables AI assistants to manage Google Contacts remotely.

### Starting the Server

```bash
# Start on default port (8080)
google-contacts mcp

# Start with API key authentication
google-contacts mcp --api-key "your-secret-key"

# Start with Firestore-based authentication (production)
google-contacts mcp \
  --firestore-project "my-gcp-project" \
  --secret-name "oauth-credentials" \
  --base-url "https://my-cloudrun-url.run.app"
```

### OAuth Authentication

When running with Firestore integration, the server exposes OAuth endpoints:

| Endpoint | Description |
|----------|-------------|
| `/auth` | Initiates OAuth flow - redirects to Google consent |
| `/auth/callback` | OAuth callback - exchanges code for tokens |
| `/health` | Health check endpoint |

**OAuth Flow:**
1. User visits `/auth` endpoint
2. Server generates CSRF protection state token
3. Redirects to Google OAuth consent page
4. Google redirects back to `/auth/callback` with authorization code
5. Server exchanges code for refresh token
6. API key is generated and stored in Firestore (upcoming feature)

### GCP Configuration

The Makefile reads configuration from `config.yaml`:
- Project ID from `gcp.project_id`
- Region from `gcp.resources.cloud_run.region`

Override with environment variables:
```bash
GCP_PROJECT=my-project GCP_REGION=us-central1 make cloud-run-deploy
```

## Configuration

All configuration is managed through `iac.yaml`:

```yaml
prefix: scmgcontacts
project_name: CONFIGURE_YOUR_GCP_PROJECT_ID
location: europe-west1
env: prod
```

See `iac.yaml` for the complete configuration including services, resources, and custom parameters.

## File Organization

### init/ Directory
Contains initialization terraform:
- State backend setup (GCS bucket, S3+DynamoDB, or Azure Storage)
- Service accounts / IAM roles (never use defaults)
- API / service enablement

### iac/ Directory
Contains main infrastructure terraform organized by feature:
- `provider.tf` - Provider configuration and backend
- `local.tf` - Configuration loader
- `workload-*.tf` - Application workloads (Cloud Run, ECS, Container Apps)
- `database-*.tf` - Database resources
- `storage-*.tf` - Storage resources

Each resource file follows the pattern:
1. Local definitions
2. Resource definitions
3. Permissions/IAM
4. Outputs (inline, not in separate output.tf)

## Adding New Resources

1. Create new file in `iac/` directory following naming convention:
   ```bash
   # Example: iac/workload-api.tf
   ```

2. Follow the standard pattern:
   ```hcl
   # Locals for this feature
   locals {
     api_name = "api"
   }

   # Resources
   resource "..." "..." { }

   # Permissions
   resource "..." "..." { }

   # Outputs
   output "..." { }
   ```

3. Plan and deploy:
   ```bash
   make plan
   make deploy
   ```

## Troubleshooting

### Backend not configured
If you see "backend not initialized":
1. Check that `init-deploy` was run successfully
2. Verify `iac/provider.tf` has backend configuration uncommented
3. Run `cd iac && terraform init`

### State locked
If state is locked:
- GCP: Check GCS bucket for lock files
- AWS: Check DynamoDB table for locks
- Azure: Wait for lock timeout or manually remove if necessary

### Permission errors
Ensure your cloud CLI is authenticated:
- GCP: `gcloud auth application-default login`
- AWS: `aws configure`
- Azure: `az login`

## Version Control

This project uses Git for version control. Important files committed:
- `iac.yaml` - Configuration
- `init/*.tf` - Initialization terraform
- `iac/*.tf` - Infrastructure terraform
- `Makefile` - Build targets
- `.gitignore` - Excludes sensitive and generated files

**Never commit:**
- `.terraform/` directory
- `*.tfstate` files
- `*.tfvars` files with secrets
- `.env` files

## Documentation

- `README.md` (this file) - Human-readable documentation
- `CLAUDE.md` - AI-optimized project knowledge for Claude Code
- `iac.yaml` - Configuration reference

## Support

For issues or questions about the Terraform setup, refer to:
1. This README
2. `CLAUDE.md` for AI-specific documentation
3. Terraform files comments for implementation details
