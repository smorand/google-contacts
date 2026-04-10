# Authentication

## OAuth2 Flow

1. Reads credentials from `~/.credentials/google_credentials.json`
2. Checks for existing token at `~/.credentials/google_token.json`
3. If no token, initiates OAuth2 flow with browser
4. Saves token for future use
5. Creates People API service with authenticated HTTP client

## Credential Sharing Strategy

The `pkg/auth/auth.go` package is **duplicated** from email-manager.

**Both applications use:**
- Same token file: `~/.credentials/google_token.json`
- Same credentials file: `~/.credentials/google_credentials.json`
- Same scopes for unified OAuth consent

**Why duplicate?**
- Simpler deployment
- Independent builds
- No versioning conflicts
- Isolated changes

## Unified OAuth2 Scopes

```go
// Gmail API (for email-manager)
gmail.GmailModifyScope
gmail.GmailSendScope
gmail.GmailLabelsScope

// People API (for google-contacts)
people.ContactsScope
people.ContactsOtherReadonlyScope
```

## Context Token Injection

For MCP server multi-user support:

```go
// Inject refresh token into context
ctx = auth.WithRefreshToken(ctx, refreshToken)

// People API will use this token instead of local file
srv, err := contacts.GetPeopleService(ctx)
```

## File Locations

| File | Path |
|------|------|
| Credentials | `~/.credentials/google_credentials.json` |
| Token | `~/.credentials/google_token.json` |

## Vault Credential Source

On VPS, Google OAuth credentials are loaded from HashiCorp Vault KV v2.

**API:** `GET {VAULT_ADDR}/v1/secret/data/{VAULT_SECRET_PATH}`
**Auth:** `X-Vault-Token` header
**Response:** `{ "data": { "data": { ...credentials JSON... } } }` (KV v2 double nesting)
**Timeout:** 5 seconds
**Default path:** `secret/credentials/google-credentials`

### Credential Loading Priority Chain

1. **Secret Manager** (if `SECRET_PROJECT` and `SECRET_NAME` are set)
2. **Vault** (if `VAULT_ADDR` and `VAULT_TOKEN` are set)
3. **Local file** (if `CREDENTIAL_FILE` is set)

Each source logs success/failure. If a source fails, execution falls through silently.
If all sources fail, an error listing all attempted sources is returned.

Both `OAuth2Server.LoadCredentials()` (oauth2.go) and `AuthHandler.loadOAuthCredentials()` (auth.go) implement this chain.

## MCP Server Authentication

See `.agent_docs/mcp-server.md` for:
- Static API key mode
- Firestore API keys mode
- OAuth endpoints for API key generation
