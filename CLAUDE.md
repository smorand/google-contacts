# Google Contacts Manager

## Overview

| Attribute | Value |
|-----------|-------|
| Type | CLI Application + MCP Server |
| Language | Go 1.25+ |
| Purpose | Google Contacts management via People API v1 |
| Auth | OAuth2 with Google |
| CLI | Cobra |

## Project Structure

```
google-contacts/
├── cmd/google-contacts/main.go     # Entry point
├── internal/
│   ├── cli/cli.go                  # CLI commands
│   ├── contacts/service.go         # People API wrapper
│   └── mcp/
│       ├── server.go               # MCP server & tools
│       ├── auth.go                 # API key validation (Firestore)
│       ├── oauth2.go               # OAuth2 authorization server
│       └── templates/success.html  # OAuth success page
├── pkg/auth/auth.go                # OAuth2 (duplicated from email-manager)
├── init/                           # Terraform init (state backend)
├── iac/                            # Terraform infrastructure
├── config.yaml                     # Terraform configuration
└── Dockerfile                      # MCP server container
```

## Commands

```bash
# Build & Test
make build          # Build binary
make test           # Run tests
make check          # All checks (fmt, vet, test)

# Install
make install        # Install to /usr/local/bin
make uninstall      # Remove

# Infrastructure
make plan           # Preview Terraform changes
make deploy         # Deploy (Docker + Cloud Run)
```

## CLI Usage

```bash
google-contacts create -f John -l Doe -p +33612345678
google-contacts search "John"
google-contacts show c123456789
google-contacts update c123456789 --add-phone work:+33198765432
google-contacts delete c123456789
google-contacts mcp --port 8080
```

## File Locations

| File | Path |
|------|------|
| Credentials | `~/.credentials/google_credentials.json` |
| Token | `~/.credentials/google_token.json` |
| Skill | `~/.claude/skills/google-contacts/` |

## Conventions

- Error wrapping: `fmt.Errorf("context: %w", err)`
- **Last name**: Stored as provided (case preserved)
- **Phone numbers**: International format required (`+XX...`); auto-converted if possible (e.g., adds +33 for French numbers)
- Phone format: `type:number` (e.g., `work:+33123456789`)
- Email format: `type:email` (e.g., `home:john@gmail.com`)
- Address format: `type:address` (e.g., `work:50 Avenue Business, Lyon`)
- Birthday format: `YYYY-MM-DD` or `--MM-DD` (year unknown)
- pkg/auth is duplicated from email-manager, keep in sync manually

## Notes for AI

- CLI tool, avoid web/API frameworks suggestions
- OAuth2 requires browser interaction
- People API has rate limits
- Token refresh handled by oauth2 library
- Follow Go coding standards from golang skill

## Documentation Index

| Topic | File |
|-------|------|
| MCP Server | `.agent_docs/mcp-server.md` |
| People API Reference | `.agent_docs/people-api.md` |
| Terraform Infrastructure | `.agent_docs/terraform.md` |
| Testing Guide | `.agent_docs/testing.md` |
| Authentication | `.agent_docs/authentication.md` |
