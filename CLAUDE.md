# Google Contacts Manager - AI Development Guide

## Project Overview

**Type**: CLI Application
**Language**: Go 1.25+
**Purpose**: Google Contacts management via People API v1
**Authentication**: OAuth2 with Google
**CLI Framework**: Cobra

## Project Structure

Following golang skill conventions:

```
google-contacts/
├── go.mod                    # Module at root
├── go.sum
├── Makefile                  # Build automation + Terraform + Docker targets
├── Dockerfile                # MCP server container image
├── README.md                 # User documentation
├── CLAUDE.md                 # AI development guide
├── config.yaml               # Terraform configuration
├── cmd/
│   └── google-contacts/
│       └── main.go           # Entry point (minimal)
├── internal/
│   ├── cli/
│   │   └── cli.go            # CLI commands and flags
│   ├── contacts/
│   │   └── service.go        # People API service wrapper
│   └── mcp/
│       └── server.go         # MCP server implementation
├── pkg/
│   └── auth/
│       └── auth.go           # OAuth2 authentication (duplicated from email-manager)
├── init/                     # Terraform initialization (state backend, service accounts)
│   ├── provider.tf
│   ├── local.tf
│   ├── state-backend.tf
│   ├── service-accounts.tf
│   └── services.tf
└── iac/                      # Terraform infrastructure (Cloud Run, Firestore, etc.)
    ├── provider.tf.template
    ├── local.tf
    └── *.tf                  # Resource files
```

## Architecture

### Core Packages

1. **cmd/google-contacts/main.go** - Minimal entry point, initializes CLI and executes
2. **internal/cli/cli.go** - Command definitions, flag setup, command handlers
3. **internal/contacts/service.go** - People API service wrapper with `GetPeopleService()` function
4. **internal/mcp/server.go** - MCP server implementation for remote AI access
5. **pkg/auth/auth.go** - OAuth2 authentication (identical to email-manager)

### Command Structure

```
google-contacts
├── create               # Create new contact
├── search               # Search contacts
├── show                 # Show contact details
├── update               # Update existing contact
├── delete               # Delete a contact
├── mcp                  # Start MCP server for remote access
└── version              # Print version
```

## Key Dependencies

- `github.com/spf13/cobra` - CLI framework
- `google.golang.org/api/people/v1` - People API client
- `golang.org/x/oauth2` - OAuth2 authentication
- `github.com/fatih/color` - Terminal colors
- `github.com/modelcontextprotocol/go-sdk/mcp` - MCP server SDK

## Authentication Flow

1. Reads credentials from `~/.credentials/google_credentials.json`
2. Checks for existing token at `~/.credentials/google_token.json`
3. If no token, initiates OAuth2 flow with browser
4. Saves token for future use
5. Creates People API service with authenticated HTTP client

## Credential Sharing Strategy

The `pkg/auth/auth.go` package is **duplicated** (not shared as a library) from email-manager. Both applications:

- Use the same token file: `~/.credentials/google_token.json`
- Use the same credentials file: `~/.credentials/google_credentials.json`
- Have the same scopes (Gmail + People API) for unified OAuth consent

**Why duplicate instead of share?**
- Simpler deployment (no external dependencies)
- Both apps can be built and run independently
- Avoids versioning conflicts between projects
- Changes to auth in one project don't break the other

### Unified OAuth2 Scopes

The auth package includes ALL scopes for both applications:

```go
// Gmail API scopes (for email-manager)
gmail.GmailModifyScope
gmail.GmailSendScope
gmail.GmailLabelsScope

// People API scopes (for google-contacts)
people.ContactsScope
people.ContactsOtherReadonlyScope
```

## Development Workflow

### Build and Test

```bash
make build      # Build binary for current platform
make build-all  # Build for all platforms
make test       # Run tests
make fmt        # Format code
make vet        # Run linter
make check      # All checks
```

### Install/Uninstall

```bash
make install    # Install to /usr/local/bin
make uninstall  # Remove from system
```

### Common Tasks

**Add new command**:
1. Create command variable in `internal/cli/cli.go`
2. Implement `RunE` function
3. Register in `Init()` function with `RootCmd.AddCommand()`

**Add service method**:
1. Create function in `internal/contacts/service.go`
2. Use `contacts.GetPeopleService(ctx)` to get authenticated service
3. Call People API methods via `srv.People.Get()`, `srv.People.SearchContacts()`, etc.

**Using the contacts service**:
```go
import "google-contacts/internal/contacts"

// Get authenticated service (triggers OAuth flow if no token)
srv, err := contacts.GetPeopleService(ctx)
if err != nil {
    return fmt.Errorf("failed to get service: %w", err)
}

// Test connection
if err := srv.TestConnection(ctx); err != nil {
    return fmt.Errorf("connection test failed: %w", err)
}

// Use People API methods
result, err := srv.People.Get("people/c123456789").PersonFields("names,phoneNumbers").Do()
```

## File Locations

- **Credentials**: `~/.credentials/google_credentials.json`
- **Token**: `~/.credentials/google_token.json`
- **Binary**: `bin/google-contacts-<os>-<arch>` (after build)
- **Installed**: `/usr/local/bin/google-contacts` (after install)
- **Skill**: `~/.claude/skills/google-contacts/` (for Claude integration)

## Claude Skill Integration

The project includes a Claude skill at `~/.claude/skills/google-contacts/` that enables natural language interaction with Google Contacts.

### Skill Structure

```
~/.claude/skills/google-contacts/
├── SKILL.md           # Skill definition and usage documentation
└── scripts/
    └── google-contacts  # Symlink to installed binary
```

### Skill Features

- **Natural language contact creation**: "Create a contact for John Doe, phone +33612345678"
- **Screenshot-based contact creation**: "Create contact from this screenshot: ~/Downloads/card.png"
- **Contact search**: "Find contacts at Acme Corp"
- **Contact details**: "Show me John's contact details"
- **Contact update**: "Change Jean's phone number to 0698765432"
- **Contact deletion**: "Delete the contact for Jean Dupont"
- **Mandatory validation**: All contact creation requires user confirmation before execution
- **Name recognition**: Intelligent parsing of first/last names with confidence levels
- **Multiple phones/emails support**: "Add his work phone: 0123456789"
- **Birthday support**: "Jean's birthday is March 15, 1985"
- **Address support**: "Add his address: 10 Rue Example, 75001 Paris" with French auto-detection

### Updating the Skill

When adding new CLI commands or modifying existing ones:
1. Update the CLI code in `internal/cli/cli.go`
2. Update the skill documentation in `~/.claude/skills/google-contacts/SKILL.md`
3. Ensure the symlink points to the current binary

### Symlink Management

The skill uses a symlink to the built binary:
```bash
# Check current symlink
ls -la ~/.claude/skills/google-contacts/scripts/google-contacts

# Update symlink after build (if needed)
ln -sf $(pwd)/bin/google-contacts-linux-amd64 ~/.claude/skills/google-contacts/scripts/google-contacts
```

## Testing

### Test Structure

```
internal/
├── cli/
│   └── cli_test.go         # CLI utility function tests
└── contacts/
    └── service_test.go     # Service type and validation tests
```

### Running Tests

```bash
make test       # Run all tests with verbose output
go test ./...   # Alternative: run with go test directly
```

### Test Patterns

Tests use standard library testing with table-driven tests (`if` + `t.Errorf` pattern):

```go
func TestExtractID(t *testing.T) {
    tests := []struct {
        name         string
        resourceName string
        expected     string
    }{
        {"full resource name", "people/c123", "c123"},
        {"ID only", "c123", "c123"},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            result := extractID(tc.resourceName)
            if result != tc.expected {
                t.Errorf("extractID(%q) = %q, want %q", tc.resourceName, result, tc.expected)
            }
        })
    }
}
```

### What's Tested

- **CLI utilities** (`internal/cli/cli_test.go`):
  - `extractID()` - Resource name to ID extraction
  - `truncate()` - String truncation for table display
  - `formatTime()` - ISO 8601 timestamp formatting
  - `parsePhones()` - Phone string parsing with type:number format
  - `parseEmails()` - Email string parsing with type:email format
  - Field validation logic for create command

- **Service types** (`internal/contacts/service_test.go`):
  - `extractID()` - Resource name parsing
  - `ContactInput` validation with multiple phones and emails
  - `SearchResult` struct field access
  - `ContactDetails` with phone/email entries
  - Resource name normalization
  - `ParseAddress()` - Structured address parsing (French and generic formats)
  - `isPostalCode()` - Postal code detection helper

### Test Guidelines

- No network calls in unit tests (external API is mocked/avoided)
- Test pure functions and validation logic
- Use table-driven tests for multiple cases
- Test edge cases (empty strings, boundary conditions)

## Notes for AI

- This is a CLI tool, avoid suggesting web/API frameworks
- OAuth2 flow requires user browser interaction
- People API has rate limits - consider batch operations
- Token refresh is handled automatically by oauth2 library
- Always use proper error wrapping with `%w` format
- Follow Go coding standards defined in golang skill
- pkg/auth is duplicated from email-manager, keep them in sync manually

## MCP Server

The project includes an MCP (Model Context Protocol) server that enables AI assistants to manage contacts remotely over HTTP.

### Server Architecture

```
internal/mcp/
└── server.go           # MCP server setup and HTTP handler
```

### Starting the Server

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

### Authentication

The MCP server supports three authentication modes:

#### 1. No Authentication (Default)

When neither `--api-key` nor `--firestore-project` is provided, the server allows unauthenticated access. This is suitable for local development where the CLI uses the local OAuth token file (`~/.credentials/google_token.json`).

#### 2. Static API Key (`--api-key`)

For simple authentication, provide a static API key via the command line:

```bash
google-contacts mcp --api-key "my-secret-key-123"
```

Clients must include the API key in the `Authorization` header:

```bash
curl -X POST http://localhost:8080/ \
  -H "Authorization: Bearer my-secret-key-123" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ping"}}'
```

With static API key authentication, the server uses the local OAuth token file for People API access.

#### 3. Firestore API Keys (`--firestore-project`)

For production deployments with multiple users, use Firestore-based API key validation:

```bash
google-contacts mcp --firestore-project "my-gcp-project"
```

**Firestore Collection Structure:**

Collection: `api_keys`
Document ID: The API key itself (e.g., `user-abc-key-123`)

```go
// APIKeyDocument structure in Firestore
type APIKeyDocument struct {
    RefreshToken string `firestore:"refresh_token"`    // Required: OAuth refresh token
    UserEmail    string `firestore:"user_email"`       // Optional: user identifier
    CreatedAt    string `firestore:"created_at"`       // Optional: creation timestamp
    Description  string `firestore:"description"`      // Optional: key description
}
```

**Example Firestore document:**

```json
{
  "refresh_token": "1//0gxxxxxx-xxxxxxxx",
  "user_email": "user@example.com",
  "created_at": "2026-01-15T10:00:00Z",
  "description": "MCP access for user's AI assistant"
}
```

When a request arrives with a valid API key from Firestore, the server injects the associated `refresh_token` into the context. This token is then used for People API authentication, enabling per-user contact management.

**Authentication Flow:**

1. Client sends request with `Authorization: Bearer <api_key>`
2. Server looks up `api_keys/<api_key>` document in Firestore
3. If found, extracts `refresh_token` from document
4. Injects token into context via `auth.WithRefreshToken(ctx, token)`
5. People API service uses this token for authenticated requests

#### Context Token Injection

The `pkg/auth` package supports token injection via context:

```go
// Inject a refresh token into context
ctx = auth.WithRefreshToken(ctx, refreshToken)

// The People API service will use this token instead of the local file
srv, err := contacts.GetPeopleService(ctx)
```

This allows the MCP server to handle requests from multiple users, each with their own OAuth credentials stored in Firestore.

### Available Tools

All contact management tools are implemented:

| Tool | Description |
|------|-------------|
| **ping** | Test connectivity with the server |
| **contacts_create** | Create a new contact (requires firstName, lastName, phones) |
| **contacts_search** | Search contacts by name, phone, email, or company |
| **contacts_show** | Get full details of a contact by ID |
| **contacts_update** | Update an existing contact (only specified fields are modified) |
| **contacts_delete** | Delete a contact by ID |

**Tool Input/Output Types:**

See `internal/mcp/server.go` for complete type definitions including:
- `CreateInput` / `CreateOutput` - Contact creation with phones, emails, addresses
- `SearchInput` / `SearchOutput` - Search query and results
- `ShowInput` / `ShowOutput` - Contact ID and full details
- `UpdateInput` / `UpdateOutput` - Partial updates with add/remove operations
- `DeleteInput` / `DeleteOutput` - Contact ID and confirmation

### MCP Protocol

The server uses the official MCP Go SDK with Streamable HTTP transport:
- Protocol version: 2024-11-05
- Session-based communication (Mcp-Session-Id header)
- SSE (Server-Sent Events) for streaming responses

### Adding New Tools

To add a new MCP tool:

```go
// In internal/mcp/server.go, inside RegisterTools()

// Define input/output types
type MyInput struct {
    Field string `json:"field" jsonschema:"field description"`
}

type MyOutput struct {
    Result string `json:"result" jsonschema:"result description"`
}

// Register the tool
mcp.AddTool(s.mcpServer, &mcp.Tool{
    Name:        "my_tool",
    Description: "Tool description for AI assistants",
}, func(ctx context.Context, req *mcp.CallToolRequest, input MyInput) (
    *mcp.CallToolResult,
    MyOutput,
    error,
) {
    // Tool implementation
    return nil, MyOutput{Result: input.Field}, nil
})
```

### Testing MCP Server

```bash
# Initialize session
curl -sD - -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0"},"capabilities":{}}}'

# Extract Mcp-Session-Id from response headers, then:

# Send initialized notification
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: SESSION_ID" \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}'

# List tools
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'

# Call ping tool
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ping","arguments":{}}}'

# Search contacts
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"contacts_search","arguments":{"query":"John"}}}'

# Create a contact
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"contacts_create","arguments":{"firstName":"John","lastName":"Doe","phones":[{"value":"+33612345678","type":"mobile"}]}}}'

# Show contact details
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"contacts_show","arguments":{"contactId":"c123456789"}}}'
```

## People API Reference

### Service Wrapper

The `internal/contacts/service.go` provides:
- `GetPeopleService(ctx)` - Returns authenticated `*Service` wrapper
- `(*Service).TestConnection(ctx)` - Verifies API connectivity
- `(*Service).CreateContact(ctx, input)` - Creates a new contact
- `(*Service).SearchContacts(ctx, query)` - Searches contacts by name, phone, email, company
- `(*Service).GetContact(ctx, resourceName)` - Retrieves a single contact by ID (basic info)
- `(*Service).GetContactDetails(ctx, resourceName)` - Retrieves full contact details with all phones, emails, and metadata
- `(*Service).UpdateContact(ctx, resourceName, input)` - Updates an existing contact (only specified fields)
- `(*Service).DeleteContact(ctx, resourceName)` - Deletes a contact by ID

The `Service` struct embeds `*people.Service` for full People API access.

### ContactInput and CreatedContact

```go
// Input for creating a contact (supports multiple phones, emails, and addresses)
type ContactInput struct {
    FirstName string          // Required
    LastName  string          // Required
    Phones    []PhoneEntry    // Required (at least one)
    Emails    []EmailEntry    // Optional (multiple emails with types)
    Addresses []AddressEntry  // Optional (multiple addresses with types)
    Company   string          // Optional
    Position  string          // Optional
    Notes     string          // Optional
    Birthday  string          // Optional (YYYY-MM-DD or --MM-DD)
}

// PhoneEntry represents a phone with type label
type PhoneEntry struct {
    Value string  // e.g., "+33612345678"
    Type  string  // mobile, work, home, main, other (default: mobile)
}

// EmailEntry represents an email with type label
type EmailEntry struct {
    Value string  // e.g., "john@acme.com"
    Type  string  // work, home, other (default: work)
}

// AddressEntry represents a postal address with type label
type AddressEntry struct {
    Value string  // e.g., "10 Rue Example, 75001 Paris, France"
    Type  string  // home, work, other (default: home)
}

// Result of contact creation
type CreatedContact struct {
    ResourceName string  // e.g., "people/c123456789"
    DisplayName  string  // e.g., "John Doe"
}
```

### Phone Types

Valid phone types for create/update commands:
- `mobile` - Mobile phone (default if not specified)
- `work` - Work phone
- `home` - Home phone
- `main` - Main phone
- `other` - Other phone

Phone format in CLI: `type:number` or just `number` (defaults to mobile)
- Simple: `+33612345678` → mobile
- Typed: `work:+33123456789` → work

### Email Types

Valid email types for create/update commands:
- `work` - Work email (default if not specified)
- `home` - Personal/home email
- `other` - Other email

Email format in CLI: `type:email` or just `email` (defaults to work)
- Simple: `john@acme.com` → work
- Typed: `home:john@gmail.com` → home

### Address Types

Valid address types for create/update commands:
- `home` - Home address (default if not specified)
- `work` - Work address
- `other` - Other address

Address format in CLI: `type:address` or just `address` (defaults to home)
- Simple: `10 Rue Example, 75001 Paris, France` → home
- Typed: `work:50 Avenue Business, Lyon, 69001` → work

### Structured Address Parsing

The service automatically parses addresses into structured fields for better Google Contacts integration.

**StructuredAddress type:**
```go
type StructuredAddress struct {
    FormattedValue string // Full address as a single string
    StreetAddress  string // Street name and number
    City           string // City name
    PostalCode     string // Postal/ZIP code
    Region         string // State/Province (optional)
    Country        string // Country name
    CountryCode    string // ISO 3166-1 alpha-2 code (optional)
}
```

**Supported address formats:**

1. **French format (auto-detected)**: Addresses with 5-digit postal codes are recognized as French
   - `10 Rue Example, 75001 Paris` → street, postal, city, country=France
   - `10 Rue Example, Paris 75001` → street, city, postal, country=France
   - `10 Rue Example, 75001 Paris, France` → street, postal, city, country

2. **Generic comma-separated**: For non-French addresses
   - `123 Main St, New York, USA` → street, city, country
   - `123 Main St, London, SW1A 1AA, UK` → street, city, postal, country

3. **Structured syntax**: Explicit field specification with semicolons
   - `street=10 Rue Test;city=Paris;postal=75001;country=France`
   - Supported keys: `street`, `city`, `postal`, `region`, `country`, `countrycode`

**Usage in code:**
```go
// Parse an address into structured fields
structured := contacts.ParseAddress("10 Rue Example, 75001 Paris")
// structured.StreetAddress = "10 Rue Example"
// structured.PostalCode = "75001"
// structured.City = "Paris"
// structured.Country = "France"
// structured.CountryCode = "FR"
```

**Note**: The parsing is automatic in CreateContact and UpdateContact. The CLI address input is transparently parsed and stored with structured fields in Google Contacts.

### Phone Number Normalization

All phone numbers are automatically normalized to international format when creating or updating contacts.

**Normalization rules:**
- Phone starting with `0` (French local format): prepends `+33` and removes leading `0`
- Phone starting with `00` (international prefix): replaces `00` with `+`
- Phone already starting with `+`: keeps as-is
- Removes spaces, dashes, dots, and parentheses for consistency

**Examples:**
| Input | Output |
|-------|--------|
| `0612345678` | `+33612345678` |
| `06 12 34 56 78` | `+33612345678` |
| `06.12.34.56.78` | `+33612345678` |
| `+33612345678` | `+33612345678` |
| `+1-555-123-4567` | `+15551234567` |
| `0033612345678` | `+33612345678` |

**Usage in code:**
```go
// Normalize a phone number
normalized := contacts.NormalizePhoneNumber("06 12 34 56 78")
// normalized = "+33612345678"
```

**Note**: Normalization is automatic in CreateContact and UpdateContact. Phone numbers provided via CLI or API are transparently normalized before being stored in Google Contacts. This ensures consistent phone number format and enables better search.

### Common PersonFields

When calling People API methods, use `PersonFields()` to specify which fields to return:
- `names` - First name, last name
- `phoneNumbers` - Phone numbers with labels
- `emailAddresses` - Email addresses with labels
- `addresses` - Postal addresses with labels
- `organizations` - Company, job title
- `biographies` - Notes/bio text
- `birthdays` - Birthday dates
- `metadata` - Creation/update times, sources

Example: `PersonFields("names,phoneNumbers,emailAddresses,addresses,organizations")`

### People API Patterns

**Creating contacts:**
```go
person := &people.Person{
    Names: []*people.Name{{
        GivenName:  "John",
        FamilyName: "Doe",
    }},
    PhoneNumbers: []*people.PhoneNumber{{
        Value: "+33612345678",
        Type:  "mobile",  // mobile, work, home, etc.
    }},
}

created, err := srv.People.CreateContact(person).
    PersonFields("names,phoneNumbers").
    Context(ctx).
    Do()
```

**Organization fields:**
- `Name` - Company name
- `Title` - Job title/position

**Biography (notes):**
- `Value` - The note text
- `ContentType` - "TEXT_PLAIN" or "TEXT_HTML"

**Searching contacts:**
```go
// SearchContacts searches by name, phone, email, company
results, err := srv.SearchContacts(ctx, "John")

// SearchResult contains contact summary
type SearchResult struct {
    ResourceName string  // e.g., "people/c123456789"
    DisplayName  string  // e.g., "John Doe"
    Phone        string  // First phone number
    Email        string  // First email address
    Company      string  // Company name
    Position     string  // Job title
    Notes        string  // Biography text
}
```

**Search API warmup:**
- People API recommends sending a warmup request with empty query before actual search
- This updates the search cache for better results
- The `SearchContacts` method handles this automatically

**GetContact for single contact:**
```go
// Get by full resource name or just the ID
contact, err := srv.GetContact(ctx, "people/c123456789")
contact, err := srv.GetContact(ctx, "c123456789")  // ID only also works
```

**GetContactDetails for full information:**
```go
// GetContactDetails returns all phones, emails, addresses with labels, and metadata
details, err := srv.GetContactDetails(ctx, "c123456789")

// ContactDetails contains complete contact information
type ContactDetails struct {
    ResourceName string
    FirstName    string
    LastName     string
    DisplayName  string
    Phones       []PhoneEntry    // All phones with labels
    Emails       []EmailEntry    // All emails with labels
    Addresses    []AddressEntry  // All addresses with labels
    Company      string
    Position     string
    Notes        string
    Birthday     string          // Format: YYYY-MM-DD or --MM-DD (if year unknown)
    CreatedAt    string
    UpdatedAt    string
}

// PhoneEntry, EmailEntry, and AddressEntry include type labels
type PhoneEntry struct {
    Value string  // e.g., "+33612345678"
    Type  string  // e.g., "mobile", "work", "home"
}
type EmailEntry struct {
    Value string  // e.g., "john@acme.com"
    Type  string  // e.g., "work", "home"
}
type AddressEntry struct {
    Value string  // e.g., "10 Rue Example, 75001 Paris"
    Type  string  // e.g., "home", "work"
}
```

**Deleting contacts:**
```go
// DeleteContact deletes by resource name or ID
err := srv.DeleteContact(ctx, "c123456789")
// or
err := srv.DeleteContact(ctx, "people/c123456789")
```

**Delete API behavior:**
- Returns empty response on success
- Returns error if contact not found
- Deletion is permanent (no undo)
- Best practice: fetch contact details first to display confirmation

**Updating contacts:**
```go
// UpdateInput uses pointers to distinguish "not provided" from "empty value"
type UpdateInput struct {
    FirstName       *string         // Optional - only update if non-nil
    LastName        *string         // Optional
    Phone           *string         // Optional - replaces first phone (backward compat)
    Phones          []PhoneEntry    // Optional - replaces ALL phones
    AddPhones       []PhoneEntry    // Optional - add phones without removing existing
    RemovePhones    []string        // Optional - remove phones by value
    Email           *string         // Optional - replaces first email (backward compat)
    Emails          []EmailEntry    // Optional - replaces ALL emails
    AddEmails       []EmailEntry    // Optional - add emails without removing existing
    RemoveEmails    []string        // Optional - remove emails by value
    Addresses       []AddressEntry  // Optional - replaces ALL addresses
    AddAddresses    []AddressEntry  // Optional - add addresses without removing existing
    RemoveAddresses []string        // Optional - remove addresses by street content match
    Company         *string         // Optional
    Position        *string         // Optional
    Notes           *string         // Optional
    Birthday        *string         // Optional - sets birthday (YYYY-MM-DD or --MM-DD)
    ClearBirthday   bool            // Optional - set to true to remove birthday
}

// UpdateContact merges changes with existing contact
details, err := srv.UpdateContact(ctx, "c123456789", contacts.UpdateInput{
    FirstName: &newFirstName,  // Only this field will be updated
})

// Add a phone without removing existing
details, err := srv.UpdateContact(ctx, "c123456789", contacts.UpdateInput{
    AddPhones: []contacts.PhoneEntry{{Value: "+33123456789", Type: "work"}},
})

// Remove a specific phone by value
details, err := srv.UpdateContact(ctx, "c123456789", contacts.UpdateInput{
    RemovePhones: []string{"+33612345678"},
})

// Add an email without removing existing
details, err := srv.UpdateContact(ctx, "c123456789", contacts.UpdateInput{
    AddEmails: []contacts.EmailEntry{{Value: "john@gmail.com", Type: "home"}},
})

// Remove a specific email by value
details, err := srv.UpdateContact(ctx, "c123456789", contacts.UpdateInput{
    RemoveEmails: []string{"old@acme.com"},
})

// Add an address without removing existing
details, err := srv.UpdateContact(ctx, "c123456789", contacts.UpdateInput{
    AddAddresses: []contacts.AddressEntry{{Value: "50 Avenue Business, Lyon", Type: "work"}},
})

// Remove an address by street content match
details, err := srv.UpdateContact(ctx, "c123456789", contacts.UpdateInput{
    RemoveAddresses: []string{"Avenue Business"},
})
```

**Update API pattern:**
- Uses `People.UpdateContact(resourceName, person).UpdatePersonFields(...)`
- `UpdatePersonFields` specifies which fields to modify (comma-separated)
- Fetches current contact first to preserve unchanged fields
- Returns updated `ContactDetails` for display
- Only fields with non-nil values in `UpdateInput` are modified

**Phone update options (in priority order):**
1. `--phone` (or `-p`): Replaces first phone only (backward compatible)
2. `--phones`: Replaces ALL phones with new ones
3. `--add-phone`: Adds phone(s) without removing existing
4. `--remove-phone`: Removes specific phone(s) by value

**Email update options (in priority order):**
1. `--email` (or `-e`): Replaces first email only (backward compatible)
2. `--emails`: Replaces ALL emails with new ones
3. `--add-email`: Adds email(s) without removing existing
4. `--remove-email`: Removes specific email(s) by value

**Address update options:**
1. `--addresses`: Replaces ALL addresses with new ones
2. `--add-address`: Adds address(es) without removing existing
3. `--remove-address`: Removes addresses by street content match

**Birthday update options:**
- `--birthday` (or `-b`): Sets birthday (format: YYYY-MM-DD or --MM-DD)
- `--clear-birthday`: Removes birthday from contact

**Birthday formats:**
- Full date: `YYYY-MM-DD` (e.g., "1985-03-15")
- Month/day only: `--MM-DD` (e.g., "--03-15" when year is unknown)

**People API metadata:**
- Metadata contains source information including creation/update times
- Access via `p.Metadata.Sources` array
- Filter by `source.Type == "CONTACT"` for user contacts
- `source.UpdateTime` contains the last modification timestamp in ISO 8601 format

## Terraform Infrastructure

The project includes Terraform infrastructure for deploying the MCP server to Google Cloud Platform.

### Infrastructure Structure

```
google-contacts/
├── config.yaml               # Single source of terraform configuration
├── init/                     # Initialization terraform (run once)
│   ├── provider.tf           # GCP provider config
│   ├── local.tf              # Loads config.yaml
│   ├── state-backend.tf      # Creates GCS bucket for state
│   ├── service-accounts.tf   # Creates custom service accounts
│   └── services.tf           # Enables required APIs
└── iac/                      # Main infrastructure
    ├── provider.tf.template  # Provider template (before init-deploy)
    ├── provider.tf           # Generated (after init-deploy)
    ├── local.tf              # Loads config.yaml
    └── *.tf                  # Resource files (added in later stories)
```

### Configuration (config.yaml)

Single source of truth for all terraform variables:

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
    # ... more services
  resources:
    cloud_run:
      name: google-contacts-mcp
      region: europe-west1
      cpu: "1"
      memory: 256Mi
```

### Deployment Workflow

**First Time Setup:**
```bash
# 1. Review init resources (state bucket, service accounts)
make init-plan

# 2. Deploy initialization (creates GCS bucket for state)
make init-deploy

# 3. Review main infrastructure
make plan

# 4. Deploy main infrastructure
make deploy
```

**Regular Updates:**
```bash
make plan    # Review changes
make deploy  # Apply changes
```

### Makefile Targets

**Terraform Targets:**
| Target | Description |
|--------|-------------|
| `init-plan` | Plan initialization resources |
| `init-deploy` | Deploy initialization (state backend, service accounts) |
| `init-destroy` | Destroy initialization (DANGEROUS!) |
| `plan` | Plan main infrastructure changes |
| `deploy` | Deploy main infrastructure |
| `undeploy` | Destroy main infrastructure |
| `update-backend` | Regenerate iac/provider.tf from template |

**Docker/Cloud Run Deployment Targets:**
| Target | Description |
|--------|-------------|
| `docker-build` | Build container image locally |
| `docker-push` | Push container to Artifact Registry |
| `cloud-run-deploy` | Full deployment (build + push + deploy) |

### Docker Deployment

The project includes a Dockerfile for containerized deployment of the MCP server.

**Dockerfile:**
- Multi-stage build using Go 1.25
- Final image based on Alpine Linux (~20MB)
- Runs as non-root user for security
- Exposes port 8080
- Health check endpoint at `/health` (wget-based)

**Environment Variables (for Cloud Run):**
| Variable | Description |
|----------|-------------|
| `PORT` | Server listening port (default: 8080) |
| `FIRESTORE_PROJECT` | GCP project for API key validation |

**Building Locally:**
```bash
# Build the image
make docker-build

# Run locally (no auth)
docker run -p 8080:8080 google-contacts-mcp:latest

# Run with static API key
docker run -p 8080:8080 -e API_KEY=my-secret google-contacts-mcp:latest --api-key "$API_KEY"
```

**Deploying to Cloud Run:**
```bash
# Full deployment (builds, pushes, and deploys)
make cloud-run-deploy

# Or step by step:
make docker-build    # Build image
make docker-push     # Push to Artifact Registry
# Then use gcloud run deploy manually
```

**Configuration:**
The Makefile reads GCP settings from `config.yaml`:
- `GCP_PROJECT`: From `gcp.project_id`
- `GCP_REGION`: From `gcp.resources.cloud_run.region`

Override with environment variables if needed:
```bash
GCP_PROJECT=my-project GCP_REGION=us-central1 make cloud-run-deploy
```

### File Organization Rules

- **init/ folder**: One-time setup (state backend, service accounts, API enablement)
- **iac/ folder**: Application infrastructure (Cloud Run, Firestore, etc.)
- Resource files named by feature: `workload-mcp.tf`, `database-firestore.tf`
- Structure per file: locals → resources → permissions → outputs
- NO separate `output.tf` - outputs are inline in each resource file

### Cloud Run Service (iac/workload-mcp.tf)

The MCP server is deployed as a Cloud Run service with the following resources:

**Resources:**
| Resource | Type | Description |
|----------|------|-------------|
| `google_artifact_registry_repository.mcp` | Artifact Registry | Docker repository for container images |
| `google_cloud_run_v2_service.mcp` | Cloud Run v2 | MCP server service with autoscaling |
| `google_project_iam_member.mcp_firestore` | IAM | Firestore access for API key storage |
| `google_project_iam_member.mcp_secretmanager` | IAM | Secret Manager access for OAuth credentials |
| `google_cloud_run_v2_service_iam_member.mcp_public` | IAM | Public access (API key protection at app level) |

**Configuration (from config.yaml):**
```yaml
gcp:
  resources:
    cloud_run:
      name: google-contacts-mcp    # Service name
      region: europe-west1          # Deployment region
      cpu: "1"                      # CPU allocation
      memory: 256Mi                 # Memory limit
      min_instances: 0              # Scale to zero when idle
      max_instances: 3              # Maximum scaling
      allow_unauthenticated: true   # Public access (API key at app level)
    artifact_registry:
      name: google-contacts         # Repository name
      format: DOCKER                # Container format
```

**Environment Variables:**
| Variable | Value | Description |
|----------|-------|-------------|
| `FIRESTORE_PROJECT` | `${project_id}` | GCP project for Firestore API key storage |
| `PORT` | `8080` | Server listening port |
| `ENVIRONMENT` | `${env}` | Environment name (prd, dev, etc.) |
| `PROJECT_ID` | `${project_id}` | GCP project ID |

**Service Account Permissions:**
- `roles/datastore.user` - Read/write access to Firestore (API keys collection)
- `roles/secretmanager.secretAccessor` - Read access to OAuth credentials secret

**Outputs:**
- `mcp_url` - Cloud Run service URL (https://google-contacts-mcp-xxx.run.app)
- `mcp_service_account` - Service account email
- `artifact_registry_url` - Docker registry URL for pushing images

### Firestore Database (iac/database-firestore.tf)

The Firestore database stores API keys for MCP server authentication.

**Resources:**
| Resource | Type | Description |
|----------|------|-------------|
| `google_firestore_database.main` | Firestore Database | Native mode database in eur3 |
| `google_firestore_index.api_keys_created_at` | Firestore Index | Index by creation date (descending) |
| `google_firestore_index.api_keys_user_email` | Firestore Index | Index by user email + creation date |

**Configuration (from config.yaml):**
```yaml
gcp:
  resources:
    firestore:
      database_id: "(default)"    # Database name
      location_id: eur3           # Europe multi-region
```

**Outputs:**
- `firestore_database_name` - Database name
- `firestore_location` - Database location

### Firestore Collection Structure

The `api_keys` collection stores API key documents for MCP authentication. This collection is NOT created by Terraform - Firestore creates collections automatically on first document write.

**Collection:** `api_keys`
**Document ID:** The API key itself (UUID v4, e.g., `550e8400-e29b-41d4-a716-446655440000`)

**Document Fields:**
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `refresh_token` | string | OAuth refresh token for Google API access | Yes |
| `user_email` | string | Email address from OAuth flow | No |
| `created_at` | string | ISO 8601 timestamp when key was created | No |
| `description` | string | Optional description for the API key | No |

**Example Document:**
```json
{
  "refresh_token": "1//0gxxxxxx-xxxxxxxx",
  "user_email": "user@example.com",
  "created_at": "2026-01-15T10:00:00Z",
  "description": "MCP access for Claude AI assistant"
}
```

**Indexes:**
1. **api_keys_created_at**: For listing keys by creation date (admin purposes)
   - `created_at` DESCENDING + `__name__` DESCENDING
2. **api_keys_user_email**: For finding keys by user
   - `user_email` ASCENDING + `created_at` DESCENDING + `__name__` DESCENDING

**Go Type Definition (from internal/mcp/server.go):**
```go
type APIKeyDocument struct {
    RefreshToken string `firestore:"refresh_token"`
    UserEmail    string `firestore:"user_email"`
    CreatedAt    string `firestore:"created_at"`
    Description  string `firestore:"description"`
}
```

**Usage Pattern:**
1. User completes OAuth flow via `/auth` endpoint
2. Server generates UUID v4 API key
3. Server stores refresh token in Firestore with the API key as document ID
4. User receives API key to use in `Authorization: Bearer <key>` header
5. On each request, server looks up API key in Firestore
6. If found, uses stored refresh_token for Google API authentication

### Secret Manager (iac/secrets.tf)

Secret Manager stores OAuth credentials for the MCP server.

**Resources:**
| Resource | Type | Description |
|----------|------|-------------|
| `google_secret_manager_secret.oauth_credentials` | Secret Manager Secret | Stores OAuth client credentials |

**Configuration (from config.yaml):**
```yaml
secrets:
  oauth_credentials: scm-pwd-oauth-creds
```

**Outputs:**
- `oauth_secret_name` - Secret Manager secret name
- `oauth_secret_id` - Secret Manager secret resource ID

**Important:** The secret version (actual credentials) must be created MANUALLY after Terraform creates the secret. This keeps sensitive data out of Terraform state.

**Manual Secret Creation:**
```bash
# After terraform creates the secret, add the credentials:
gcloud secrets versions add scm-pwd-oauth-creds \
  --data-file=$HOME/.credentials/scm-pwd.json \
  --project=scmgcontacts-mcp-prd

# Verify the secret version:
gcloud secrets versions list scm-pwd-oauth-creds \
  --project=scmgcontacts-mcp-prd

# Check the secret exists:
gcloud secrets describe scm-pwd-oauth-creds \
  --project=scmgcontacts-mcp-prd
```

**Expected Secret Format:**
The secret should contain JSON with OAuth client credentials:
```json
{
  "installed": {
    "client_id": "xxx.apps.googleusercontent.com",
    "client_secret": "yyy",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token",
    ...
  }
}
```

**Security Notes:**
- Secret version data is NOT stored in Terraform state (manual upload)
- Cloud Run service account has `roles/secretmanager.secretAccessor` only (read-only)
- Secrets can be rotated by adding new versions

### Adding New Resources

1. Create a new `.tf` file in `iac/` named by feature
2. Follow the pattern: locals → resources → permissions → outputs
3. Reference config.yaml values via `local.config.*`
4. Run `make plan` to preview changes
5. Run `make deploy` to apply

### Notes for AI

- Always run `make plan` before `make deploy`
- Never commit `.terraform/` or `*.tfstate` files
- The `init/` folder creates infrastructure that other resources depend on
- After `make init-deploy`, the backend config is automatically copied to `iac/provider.tf`
- Use `config.yaml` as the single source of truth for configuration
- Firestore collections are created automatically on first write (not by Terraform)
- API keys use document ID as the key itself for O(1) lookup performance
