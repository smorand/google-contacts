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

## MCP Server Authentication

See `.agent_docs/mcp-server.md` for:
- Static API key mode
- Firestore API keys mode
- OAuth endpoints for API key generation
