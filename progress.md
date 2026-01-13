# Google Contacts Project - Progress Log

## 2026-01-13 - US-00001 - email-manager: Restructure to golang skill layout

**Status:** Completed successfully

### What was implemented
Restructured the email-manager project from a flat `src/` directory to follow golang skill conventions:
- Moved `go.mod` and `go.sum` from `src/` to project root
- Created `cmd/email-manager/main.go` as minimal entry point (only wiring)
- Created `internal/cli/cli.go` with all CLI command implementations
- Created `internal/gmail/service.go` with Gmail API service and helper functions
- Created `pkg/auth/auth.go` with OAuth2 authentication logic
- Removed old `src/` directory

### Files changed
- **Created:**
  - `go.mod` (at root, moved from src/)
  - `go.sum` (at root, moved from src/)
  - `cmd/email-manager/main.go` - Minimal entry point
  - `internal/cli/cli.go` - CLI commands (refactored from src/cli.go)
  - `internal/gmail/service.go` - Gmail service helpers (extracted)
  - `pkg/auth/auth.go` - OAuth2 auth (refactored from src/auth.go)
- **Modified:**
  - `.gitignore` - Updated to handle new structure (prefix `/` for binary)
  - `CLAUDE.md` - Updated with new project structure
  - `README.md` - Updated with new project structure
- **Deleted:**
  - `src/main.go`
  - `src/cli.go`
  - `src/auth.go`
  - `src/go.mod`
  - `src/go.sum`

### Learnings

**Makefile auto-detection:**
- The existing Makefile already supports auto-detecting project structure (src/ vs cmd/)
- It checks for `src/` first, then `cmd/`, so the old src/ must be removed before testing

**Gitignore patterns:**
- `email-manager` in .gitignore matches both the binary AND `cmd/email-manager/` directory
- Use `/email-manager` to only match the binary at root level
- Added `bin/` to ignore build output directory

**Package organization:**
- `pkg/auth/` contains shared OAuth2 code designed for duplication (not as a library)
- `internal/cli/` for CLI-specific code
- `internal/gmail/` for Gmail-specific service code
- Import the gmail API as `gmailapi` to avoid conflict with internal package name

**Token file naming:**
- Original used `token_gmail.json`
- Changed to `google_token.json` for unified credential sharing with google-contacts

---

## 2026-01-13 - US-00002 - email-manager: Add People API scopes for unified credentials

**Status:** Completed successfully

### What was implemented
Extended email-manager's OAuth2 configuration to include Google People API scopes, enabling a single OAuth consent for both email-manager and google-contacts applications.

### Files changed
- **Modified:**
  - `pkg/auth/auth.go` - Added People API import and scopes
  - `CLAUDE.md` - Added documentation for unified scopes
  - `README.md` - Added credential sharing section

### Changes details
Added to `pkg/auth/auth.go`:
```go
import (
    gmail "google.golang.org/api/gmail/v1"
    people "google.golang.org/api/people/v1"
)

var Scopes = []string{
    // Gmail API scopes (for email-manager)
    gmail.GmailModifyScope,
    gmail.GmailSendScope,
    gmail.GmailLabelsScope,
    // People API scopes (for google-contacts)
    people.ContactsScope,
    people.ContactsOtherReadonlyScope,
}
```

### Learnings

**Package aliasing for v1 conflicts:**
- Both `gmail/v1` and `people/v1` packages have default import name `v1`
- Must use explicit aliases: `gmail "google.golang.org/api/gmail/v1"` and `people "google.golang.org/api/people/v1"`

**People API scopes:**
- `people.ContactsScope` = "https://www.googleapis.com/auth/contacts" (read/write)
- `people.ContactsOtherReadonlyScope` = "https://www.googleapis.com/auth/contacts.other.readonly"

**Scope change implications:**
- Adding new scopes requires re-authorization
- Users must delete `~/.credentials/google_token.json` to force re-auth
- The OAuth consent screen will then request permissions for ALL scopes (Gmail + Contacts)

---

## 2026-01-13 - US-00003 - google-contacts: Project initialization with duplicated auth

**Status:** Completed successfully

### What was implemented
Initialized the google-contacts project following golang skill conventions with duplicated auth code from email-manager.

**Project structure created:**
```
google-contacts/
├── go.mod                    # Module at root
├── go.sum
├── Makefile                  # Copied from email-manager (generic)
├── README.md                 # Project documentation
├── CLAUDE.md                 # AI development guide
├── .gitignore                # Go patterns
├── cmd/
│   └── google-contacts/
│       └── main.go           # Minimal entry point
├── internal/
│   └── cli/
│       └── cli.go            # Root command + version
└── pkg/
    └── auth/
        └── auth.go           # OAuth2 (duplicated from email-manager)
```

### Files created
- `go.mod` - Module definition with dependencies
- `go.sum` - Dependency checksums (auto-generated)
- `Makefile` - Build automation (copied from email-manager)
- `README.md` - Project description and usage
- `CLAUDE.md` - AI development guide
- `.gitignore` - Git ignore patterns
- `cmd/google-contacts/main.go` - Entry point (18 lines)
- `internal/cli/cli.go` - CLI with version command
- `pkg/auth/auth.go` - OAuth2 authentication (identical to email-manager)

### Learnings

**Makefile reusability:**
- The email-manager Makefile is completely generic
- Uses `$(shell basename $$(pwd))` for binary name
- Auto-detects project structure (src/ vs cmd/)
- Can be copied directly to new Go projects

**go mod tidy behavior:**
- Running `go mod download && go mod tidy` populates go.sum
- The Makefile handles this automatically during build
- Direct dependencies in go.mod, indirect ones added automatically

**Version command patterns:**
- Cobra supports `--version` flag on root command via `RootCmd.Version`
- Custom version template: `RootCmd.SetVersionTemplate("...")`
- Also added explicit `version` subcommand for consistency

**Auth duplication strategy:**
- pkg/auth/auth.go is byte-for-byte identical to email-manager
- `diff` command returns no output (files match)
- Both apps use same token file path for credential sharing

---

## 2026-01-13 - US-00004 - google-contacts: People API service initialization

**Status:** Completed successfully

### What was implemented
Implemented Google People API service wrapper in `internal/contacts/service.go`:
- `Service` struct that embeds `*people.Service` for full People API access
- `GetPeopleService(ctx)` function that returns an authenticated service
- `TestConnection(ctx)` method to verify API connectivity by fetching the user profile

### Files changed
- **Created:**
  - `internal/contacts/service.go` - People API service wrapper
- **Modified:**
  - `CLAUDE.md` - Added People API reference section with usage examples
  - `README.md` - Added first-time authentication section

### Learnings

**Service wrapper pattern:**
- Embedding `*people.Service` in a custom `Service` struct provides:
  - Full access to all People API methods via the embedded struct
  - Ability to add custom helper methods (like `TestConnection`)
  - Clean API surface for consumers

**People API client creation:**
- Use `people.NewService(ctx, option.WithHTTPClient(client))` to create the service
- The `option.WithHTTPClient()` is from `google.golang.org/api/option` package
- Import alias: `people "google.golang.org/api/people/v1"`

**PersonFields parameter:**
- People API methods require specifying which fields to return via `PersonFields()`
- Common fields: `names`, `phoneNumbers`, `emailAddresses`, `organizations`, `biographies`, `metadata`
- Fields are comma-separated: `PersonFields("names,phoneNumbers")`

**Testing connection:**
- Fetching `people/me` with minimal fields is an efficient way to verify API connectivity
- This confirms authentication works without returning large amounts of data

---
