# Google Contacts MCP Server

A Go CLI tool and MCP (Model Context Protocol) server for managing Google Contacts via the People API v1. Enables AI assistants like Claude to interact with your Google Contacts remotely.

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

The project includes an MCP (Model Context Protocol) server that enables AI assistants to manage Google Contacts remotely over HTTP.

### Features

- **Full contact management**: Create, search, show, update, and delete contacts
- **Multi-user support**: Each API key maps to a different Google account
- **OAuth authentication flow**: Generate API keys via browser-based OAuth
- **Firestore-based key storage**: Production-ready persistent API key management
- **Cloud Run deployment**: Scalable, serverless deployment

### Quick Start (Local Development)

```bash
# Build the binary
make build

# Start MCP server on default port (8080)
./bin/google-contacts-linux-amd64 mcp

# Start with static API key (simple authentication)
./bin/google-contacts-linux-amd64 mcp --api-key "your-secret-key"
```

### Production Deployment

```bash
# 1. Deploy infrastructure (Terraform)
make plan     # Preview changes
make deploy   # Deploy Cloud Run, Firestore, Secret Manager

# 2. Upload OAuth credentials to Secret Manager
gcloud secrets versions add scm-pwd-oauth-creds \
  --data-file=$HOME/.credentials/scm-pwd.json \
  --project=scmgcontacts-mcp-prd

# 3. Build and deploy container
make cloud-run-deploy
```

### Starting the Server

```bash
# Local development (no auth)
google-contacts mcp

# Local development with static API key
google-contacts mcp --api-key "your-secret-key"

# Production with Firestore-based authentication
google-contacts mcp \
  --firestore-project "my-gcp-project" \
  --secret-name "oauth-credentials" \
  --base-url "https://my-cloudrun-url.run.app"

# Custom port and host
google-contacts mcp --host 0.0.0.0 --port 9090
```

### MCP Client Configuration

To use the MCP server with an AI assistant, add this to your MCP client configuration:

```json
{
  "mcpServers": {
    "google-contacts": {
      "url": "https://your-cloudrun-url.run.app",
      "transport": "streamable-http",
      "headers": {
        "Authorization": "Bearer <your-api-key>"
      }
    }
  }
}
```

For Claude Desktop, add this to your `claude_desktop_config.json`.

### Getting an API Key

1. Visit `https://your-cloudrun-url.run.app/auth` in your browser
2. Complete the Google OAuth consent flow
3. The success page displays your API key with a copy button
4. Store the API key securely - it provides access to your Google Contacts

### OAuth Authentication Endpoints

When running with Firestore integration (`--firestore-project`):

| Endpoint | Description |
|----------|-------------|
| `GET /auth` | Initiates OAuth flow - redirects to Google consent |
| `GET /auth/callback` | OAuth callback - exchanges code for tokens |
| `GET /auth/success` | Success page - displays generated API key |
| `GET /health` | Health check endpoint (for Cloud Run) |

**OAuth Flow:**
1. User visits `/auth` endpoint
2. Server generates CSRF protection state token
3. Redirects to Google OAuth consent page
4. User authorizes the application
5. Google redirects back to `/auth/callback` with authorization code
6. Server exchanges code for refresh token
7. Server generates UUID API key and stores with refresh token in Firestore
8. Server redirects to `/auth/success` page displaying the API key
9. User copies API key for use in MCP client configuration

### Available MCP Tools

| Tool | Description |
|------|-------------|
| `ping` | Test server connectivity |
| `contacts_create` | Create a new contact (firstName, lastName, phones required) |
| `contacts_search` | Search contacts by name, phone, email, or company |
| `contacts_show` | Get full details of a contact by ID |
| `contacts_update` | Update an existing contact (partial updates) |
| `contacts_delete` | Delete a contact by ID |

### Self-Hosting Guide

1. **Clone the repository**
   ```bash
   git clone https://github.com/smorand/google-contacts
   cd google-contacts
   ```

2. **Create a GCP project** (or use existing)
   ```bash
   gcloud projects create my-contacts-mcp --name="Contacts MCP"
   gcloud config set project my-contacts-mcp
   ```

3. **Update config.yaml** with your project ID
   ```yaml
   gcp:
     project_id: my-contacts-mcp
   ```

4. **Deploy infrastructure**
   ```bash
   make init-deploy  # Create state bucket, service accounts
   make deploy       # Deploy Cloud Run, Firestore, etc.
   ```

5. **Create OAuth credentials** in Google Cloud Console
   - Go to APIs & Services > Credentials
   - Create OAuth client ID (Web application)
   - Add authorized redirect URI: `https://YOUR-CLOUD-RUN-URL/auth/callback`
   - Download JSON credentials

6. **Upload credentials to Secret Manager**
   ```bash
   gcloud secrets versions add scm-pwd-oauth-creds \
     --data-file=path/to/credentials.json
   ```

7. **Deploy the container**
   ```bash
   make cloud-run-deploy
   ```

8. **Test the deployment**
   - Visit `https://YOUR-CLOUD-RUN-URL/auth` to get an API key
   - Configure your MCP client with the API key

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
