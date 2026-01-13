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
├── Makefile                  # Build automation
├── README.md                 # User documentation
├── CLAUDE.md                 # AI development guide
├── cmd/
│   └── google-contacts/
│       └── main.go           # Entry point (minimal)
├── internal/
│   ├── cli/
│   │   └── cli.go            # CLI commands and flags
│   └── contacts/             # (future) People API service
│       └── service.go
└── pkg/
    └── auth/
        └── auth.go           # OAuth2 authentication (duplicated from email-manager)
```

## Architecture

### Core Packages

1. **cmd/google-contacts/main.go** - Minimal entry point, initializes CLI and executes
2. **internal/cli/cli.go** - Command definitions, flag setup, command handlers
3. **internal/contacts/service.go** - People API service wrapper with `GetPeopleService()` function
4. **pkg/auth/auth.go** - OAuth2 authentication (identical to email-manager)

### Command Structure

```
google-contacts
├── create               # Create new contact (implemented)
├── search               # Search contacts (planned)
├── show                 # Show contact details (planned)
└── version              # Print version
```

## Key Dependencies

- `github.com/spf13/cobra` - CLI framework
- `google.golang.org/api/people/v1` - People API client
- `golang.org/x/oauth2` - OAuth2 authentication
- `github.com/fatih/color` - Terminal colors

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

## Testing

Recommended test structure:

```
internal/
├── cli/
│   └── cli_test.go
└── contacts/
    └── service_test.go
pkg/
└── auth/
    └── auth_test.go
```

## Notes for AI

- This is a CLI tool, avoid suggesting web/API frameworks
- OAuth2 flow requires user browser interaction
- People API has rate limits - consider batch operations
- Token refresh is handled automatically by oauth2 library
- Always use proper error wrapping with `%w` format
- Follow Go coding standards defined in golang skill
- pkg/auth is duplicated from email-manager, keep them in sync manually

## People API Reference

### Service Wrapper

The `internal/contacts/service.go` provides:
- `GetPeopleService(ctx)` - Returns authenticated `*Service` wrapper
- `(*Service).TestConnection(ctx)` - Verifies API connectivity
- `(*Service).CreateContact(ctx, input)` - Creates a new contact

The `Service` struct embeds `*people.Service` for full People API access.

### ContactInput and CreatedContact

```go
// Input for creating a contact
type ContactInput struct {
    FirstName string  // Required
    LastName  string  // Required
    Phone     string  // Required
    Email     string  // Optional
    Company   string  // Optional
    Position  string  // Optional
    Notes     string  // Optional
}

// Result of contact creation
type CreatedContact struct {
    ResourceName string  // e.g., "people/c123456789"
    DisplayName  string  // e.g., "John Doe"
}
```

### Common PersonFields

When calling People API methods, use `PersonFields()` to specify which fields to return:
- `names` - First name, last name
- `phoneNumbers` - Phone numbers with labels
- `emailAddresses` - Email addresses with labels
- `organizations` - Company, job title
- `biographies` - Notes/bio text
- `metadata` - Creation/update times, sources

Example: `PersonFields("names,phoneNumbers,emailAddresses,organizations")`

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
