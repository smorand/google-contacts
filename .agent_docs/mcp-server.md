# MCP Server Documentation

The project includes an MCP (Model Context Protocol) server that enables AI assistants to manage contacts remotely over HTTP.

## Server Architecture

```
internal/mcp/
├── server.go           # MCP server setup and HTTP handler
├── auth.go             # OAuth2 authentication handlers
├── templates/
│   └── success.html    # OAuth success page template (embedded)
├── server_test.go      # Server tests
└── auth_test.go        # Auth handler tests
```

## Starting the Server

```bash
# Start on default port (8080)
google-contacts mcp

# Start on custom port
google-contacts mcp --port 3000

# Start with static API key authentication (for local development)
google-contacts mcp --api-key "your-secret-key"

# Start with Firestore-based API key validation (for production)
google-contacts mcp --firestore-project "my-gcp-project"

# Bind to all interfaces (for remote access)
google-contacts mcp --host 0.0.0.0 --port 8080
```

## Authentication Modes

### 1. No Authentication (Default)

When neither `--api-key` nor `--firestore-project` is provided, the server allows unauthenticated access. Suitable for local development using local OAuth token file.

### 2. Static API Key (`--api-key`)

```bash
google-contacts mcp --api-key "my-secret-key-123"
```

Clients must include: `Authorization: Bearer my-secret-key-123`

### 3. Firestore API Keys (`--firestore-project`)

For production with multiple users:

```bash
google-contacts mcp --firestore-project "my-gcp-project"
```

**Firestore Collection:** `api_keys`
**Document ID:** The API key itself (UUID v4)

```go
type APIKeyDocument struct {
    RefreshToken string `firestore:"refresh_token"`      // Required
    AccessToken  string `firestore:"access_token"`       // Optional
    TokenExpiry  string `firestore:"token_expiry"`       // Optional
    UserEmail    string `firestore:"user_email"`         // Optional
    CreatedAt    string `firestore:"created_at"`         // Optional
    LastUsed     string `firestore:"last_used"`          // Optional
    Description  string `firestore:"description"`        // Optional
}
```

**Authentication Flow:**
1. Client sends `Authorization: Bearer <api_key>`
2. Server looks up `api_keys/<api_key>` in Firestore
3. Extracts `refresh_token` and injects into context
4. People API uses this token for requests

## OAuth Endpoints

When running with `--firestore-project`:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/auth` | GET | Start OAuth flow |
| `/auth/callback` | GET | OAuth callback |
| `/auth/success` | GET | Success page with API key |
| `/health` | GET | Health check |

**CLI Flags:**

| Flag | Description |
|------|-------------|
| `--firestore-project` | GCP project for Firestore |
| `--base-url` | Base URL for OAuth callbacks |
| `--secret-name` | Secret Manager secret name |
| `--credential-file` | Local credential file path |

## Available Tools

| Tool | Description |
|------|-------------|
| `ping` | Test connectivity |
| `contacts_create` | Create contact (firstName, lastName, phones required) |
| `contacts_search` | Search by name, phone, email, company |
| `contacts_show` | Get full contact details by ID |
| `contacts_update` | Update contact (only specified fields) |
| `contacts_delete` | Delete contact by ID |

## Data Validation Rules

### Last Name (UPPERCASE)

**All last names are automatically converted to UPPERCASE** when creating or updating contacts.

- Input: `"Doe"` → Stored as: `"DOE"`
- Input: `"Van Der Berg"` → Stored as: `"VAN DER BERG"`

This ensures consistent formatting across all contacts.

### Phone Numbers (International Format Required)

**Phone numbers MUST be in international format**, starting with `+`.

**Valid formats:**
- `+33612345678` (France)
- `+1-555-123-4567` (USA)
- `+44 20 7123 4567` (UK)

**Invalid formats (will be rejected):**
- `0612345678` (missing `+`)
- `06 12 34 56 78` (missing `+`)

**Error message:** `phone number 'XXX' must be in international format (starting with +, e.g. +33612345678)`

## MCP Protocol

- Protocol version: 2024-11-05
- Session-based communication (Mcp-Session-Id header)
- SSE for streaming responses
- Uses official MCP Go SDK with Streamable HTTP transport

## Adding New Tools

```go
// In internal/mcp/server.go, inside RegisterTools()

type MyInput struct {
    Field string `json:"field" jsonschema:"description"`
}

type MyOutput struct {
    Result string `json:"result" jsonschema:"description"`
}

mcp.AddTool(s.mcpServer, &mcp.Tool{
    Name:        "my_tool",
    Description: "Tool description",
}, func(ctx context.Context, req *mcp.CallToolRequest, input MyInput) (
    *mcp.CallToolResult, MyOutput, error,
) {
    return nil, MyOutput{Result: input.Field}, nil
})
```

## Testing MCP Server

```bash
# Initialize session
curl -sD - -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0"},"capabilities":{}}}'

# Extract Mcp-Session-Id, then:

# List tools
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'

# Call tool
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"contacts_search","arguments":{"query":"John"}}}'
```
