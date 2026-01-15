# CONFIGURE_YOUR_GCP_PROJECT_ID - AI Documentation

This file provides AI-optimized knowledge about the project's infrastructure and Terraform setup.

## Project Context

**Project:** CONFIGURE_YOUR_GCP_PROJECT_ID
**Description:** Google Contacts MCP Server - Cloud Run deployment with OAuth API Key management
**Environment:** prod
**Cloud Provider:** GCP
**Location:** europe-west1
**Prefix:** scmgcontacts

## Infrastructure Architecture

### Terraform Structure

This project follows a two-folder Terraform pattern:

1. **init/** - Initialization layer (run once)
   - Creates state backend (GCS bucket for state storage)
   - Creates custom service accounts/IAM roles (never use defaults)
   - Enables required cloud services
   - Uses local state (no backend)

2. **iac/** - Infrastructure layer (main application)
   - Uses remote state from init backend
   - Contains all application resources
   - Organized by feature, not resource type

### File Organization Rules

**Standard files in both init/ and iac/:**
- `provider.tf` - Provider configuration and credentials
- `local.tf` - Loads `iac.yaml` and defines locals

**Resource files (iac/ only):**
- Named by feature: `workload-api.tf`, `database-postgres.tf`
- Structure: locals → resources → permissions → outputs
- NO separate `output.tf` file - outputs inline in each file

### Configuration (iac.yaml)

Single source of truth for all terraform variables:

```yaml
prefix: scmgcontacts
project_name: CONFIGURE_YOUR_GCP_PROJECT_ID
location: europe-west1
env: prod
services: [...]      # APIs/services to enable
resources: {...}     # Default resource specs
parameters: {...}    # Custom env vars for workloads
```

Accessed in terraform via:
```hcl
locals {
  config = yamldecode(file("${path.root}/../iac.yaml"))
  prefix = local.config.prefix
}
```

## Deployment Workflow

### Initial Setup (First Time)
```bash
make init-plan       # Preview backend creation
make init-deploy     # Create backend and service accounts
# Update iac/provider.tf with backend config from output
make plan           # Preview infrastructure
make deploy         # Deploy infrastructure
```

### Regular Updates
```bash
make plan           # Preview changes
make deploy         # Apply changes
```

### Makefile Targets
- `plan` / `deploy` / `undeploy` - Main infrastructure (iac/)
- `init-plan` / `init-deploy` / `init-destroy` - Initialization (init/)

## Cloud Provider Specifics

### GCP

State backend: GCS bucket
Service accounts: Custom (never default)
Location ID: Calculated from europe-west1

## Git Workflow

**CRITICAL:** This project MUST be under Git version control.

### Committed Files
- `iac.yaml` - Configuration
- `init/*.tf`, `iac/*.tf` - Terraform files
- `Makefile` - Build targets
- `.gitignore` - Exclusions
- `README.md`, `CLAUDE.md` - Documentation

### Never Committed (in .gitignore)
- `.terraform/` - Provider plugins
- `*.tfstate` - State files (use remote backend)
- `*.tfvars` - May contain secrets
- `.env` - Environment variables

### Commit Guidelines
- **Always commit after making changes**
- Commit message format: `<type>: <description>`
  - `feat:` - New infrastructure
  - `fix:` - Bug fixes
  - `chore:` - Config updates
  - `docs:` - Documentation
- Example: `feat: add Cloud Run API service`
- Example: `chore: update resource limits in iac.yaml`

## Best Practices for AI Agents

### When Modifying Infrastructure

1. **Read before write**
   - Read existing terraform files to understand patterns
   - Check `iac.yaml` for configuration
   - Review `local.tf` to see available variables

2. **Follow conventions**
   - Use feature-based file naming: `workload-*.tf`, `database-*.tf`
   - Follow structure: locals → resources → permissions → outputs
   - Put outputs inline, not in separate file

3. **Validate configuration**
   ```bash
   # In skill directory
   scripts/run.sh parse_config iac.yaml --validate-only
   ```

4. **Plan before deploy**
   ```bash
   make plan  # Always review changes
   ```

5. **Commit changes**
   ```bash
   git add <changed-files>
   git commit -m "feat: add new resource"
   ```

6. **Update documentation**
   - Update this CLAUDE.md if architecture changes
   - Update README.md if user instructions change

### When Adding Resources

1. Create new file: `iac/<feature>.tf`
2. Follow pattern from `example-workload.tf`
3. Add locals for feature-specific variables
4. Use `local.config.*` for iac.yaml values
5. Use custom service accounts/roles (never defaults)
6. Add inline outputs at end of file
7. Commit: `git commit -m "feat: add <feature>"`

### When Updating Config

1. Edit `iac.yaml`
2. Validate: `scripts/run.sh parse_config`
3. Review what will change: `make plan`
4. Apply: `make deploy`
5. Commit: `git commit -m "chore: update config"`

## Common Patterns

### Loading Config Values
```hcl
# In local.tf
locals {
  config = yamldecode(file("${path.root}/../iac.yaml"))
  prefix = local.config.prefix
  env = local.config.env
  # ... extract other values
}
```

### Injecting Parameters as Env Vars
```hcl
# In workload resource
dynamic "env" {
  for_each = local.parameters
  content {
    name  = upper(replace(env.key, "-", "_"))
    value = tostring(env.value)
  }
}
```

### Naming Convention
```
{prefix}-{resource}-{env}
Example: mycompany-api-dev
```

### Service Accounts (GCP)
```hcl
# NEVER use default - always create custom
resource "google_service_account" "my_sa" {
  account_id = "${local.prefix}-myservice-${local.env}"
}
```

### IAM Roles (AWS)
```hcl
# NEVER use default - always create custom
resource "aws_iam_role" "my_role" {
  name = "${local.prefix}-myservice-${local.env}"
}
```

## Troubleshooting

### State Issues
- Init failed: Check cloud CLI authentication
- State locked: Wait or manually unlock in cloud console
- Backend not configured: Run `init-deploy` first

### Permission Issues
- Check service account/role exists in init/
- Verify IAM bindings in resource file
- Ensure cloud CLI is authenticated

### Configuration Issues
- Run: `scripts/run.sh parse_config iac.yaml`
- Check all required fields present
- Verify services list is correct for provider

## Important Notes

1. **Two-phase deployment:** init creates foundation, iac creates application
2. **Feature-based organization:** Group by feature, not resource type
3. **Config-driven:** All variables in iac.yaml, not hardcoded
4. **Custom identities:** Never use default service accounts/roles
5. **Git required:** Always under version control, commit all changes
6. **Documentation required:** Keep README.md and CLAUDE.md up to date

## Working With This Project

When asked to:
- **Add infrastructure:** Create new .tf file in iac/, follow patterns, commit
- **Update config:** Edit iac.yaml, validate, plan, deploy, commit
- **Deploy changes:** Always plan first, then deploy
- **Document changes:** Update README.md (users) and CLAUDE.md (AI)
- **Initialize new project:** Run terraform skill's init_project.py

Always remember:
- Read existing files first
- Follow established patterns
- Validate before applying
- **Commit after changes**
- Update documentation
