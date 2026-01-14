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
│   └── contacts/
│       └── service.go        # People API service wrapper
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
├── create               # Create new contact
├── search               # Search contacts
├── show                 # Show contact details
├── update               # Update existing contact
├── delete               # Delete a contact
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
  - Field validation logic for create command

- **Service types** (`internal/contacts/service_test.go`):
  - `extractID()` - Resource name parsing
  - `ContactInput` validation with multiple phones
  - `SearchResult` struct field access
  - `ContactDetails` with phone/email entries
  - Resource name normalization

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
// Input for creating a contact (supports multiple phones)
type ContactInput struct {
    FirstName string        // Required
    LastName  string        // Required
    Phones    []PhoneEntry  // Required (at least one)
    Email     string        // Optional
    Company   string        // Optional
    Position  string        // Optional
    Notes     string        // Optional
}

// PhoneEntry represents a phone with type label
type PhoneEntry struct {
    Value string  // e.g., "+33612345678"
    Type  string  // mobile, work, home, main, other (default: mobile)
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
// GetContactDetails returns all phones, emails with labels, and metadata
details, err := srv.GetContactDetails(ctx, "c123456789")

// ContactDetails contains complete contact information
type ContactDetails struct {
    ResourceName string
    FirstName    string
    LastName     string
    DisplayName  string
    Phones       []PhoneEntry  // All phones with labels
    Emails       []EmailEntry  // All emails with labels
    Company      string
    Position     string
    Notes        string
    CreatedAt    string
    UpdatedAt    string
}

// PhoneEntry and EmailEntry include type labels
type PhoneEntry struct {
    Value string  // e.g., "+33612345678"
    Type  string  // e.g., "mobile", "work", "home"
}
type EmailEntry struct {
    Value string  // e.g., "john@acme.com"
    Type  string  // e.g., "work", "home"
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
    FirstName    *string       // Optional - only update if non-nil
    LastName     *string       // Optional
    Phone        *string       // Optional - replaces first phone (backward compat)
    Phones       []PhoneEntry  // Optional - replaces ALL phones
    AddPhones    []PhoneEntry  // Optional - add phones without removing existing
    RemovePhones []string      // Optional - remove phones by value
    Email        *string       // Optional - replaces first email
    Company      *string       // Optional
    Position     *string       // Optional
    Notes        *string       // Optional
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

**People API metadata:**
- Metadata contains source information including creation/update times
- Access via `p.Metadata.Sources` array
- Filter by `source.Type == "CONTACT"` for user contacts
- `source.UpdateTime` contains the last modification timestamp in ISO 8601 format
