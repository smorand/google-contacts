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

## 2026-01-14 - US-00005 - google-contacts: Create contact command

**Status:** Completed successfully

### What was implemented
Implemented the 'create' command to add new contacts to Google Contacts via People API.

**Features:**
- Required fields: firstname (-f), lastname (-l), phone (-p)
- Optional fields: company (-c), position (-r), email (-e), notes (-n)
- Colorized output with contact ID on success
- Proper validation of required fields before API call

### Files changed
- **Modified:**
  - `internal/cli/cli.go` - Added createCmd with flags and runCreate handler
  - `internal/contacts/service.go` - Added ContactInput, CreatedContact types and CreateContact() method
  - `go.mod` / `go.sum` - Added fatih/color dependency
  - `CLAUDE.md` - Updated with create command patterns and API reference
  - `README.md` - Added create command usage documentation

### Learnings

**People API CreateContact pattern:**
- Use `srv.People.CreateContact(person).PersonFields(...).Context(ctx).Do()`
- PersonFields is required to get data back in the response
- Returns `*people.Person` with ResourceName (e.g., "people/c123456789")

**Go People API types:**
- `people.Name` - Uses `GivenName` (not FirstName) and `FamilyName` (not LastName)
- `people.PhoneNumber` - Has `Value` for the number and `Type` for label (mobile, work, home)
- `people.Organization` - `Name` is company, `Title` is job position
- `people.Biography` - `Value` for text, `ContentType` for format ("TEXT_PLAIN")

**Cobra flags pattern:**
- Use `StringVarP` for flags with both long and short names
- Flags are stored in package-level variables for access in RunE function
- Manual validation preferred over `MarkFlagRequired` for clearer error messages

**Color output:**
- `github.com/fatih/color` provides cross-platform terminal colors
- Use `color.New(color.FgGreen).SprintFunc()` to create colored string formatters
- Colors are automatically disabled when output is piped

---

## 2026-01-14 - US-00006 - google-contacts: Search contacts command

**Status:** Completed successfully

### What was implemented
Implemented the 'search' command to find contacts by name, phone, email, or company using the People API searchContacts endpoint.

**Features:**
- Single query argument searches across names, phones, emails, and organizations
- Single result: displays full contact details with colorized output
- Multiple results: displays summary table using tabwriter for aligned columns
- No results: shows appropriate message
- Automatic cache warmup request before search (as recommended by Google)

### Files changed
- **Modified:**
  - `internal/cli/cli.go` - Added searchCmd with runSearch handler, displayContactDetails(), displayContactTable(), extractID(), truncate()
  - `internal/contacts/service.go` - Added SearchResult type, SearchContacts(), GetContact() methods
  - `CLAUDE.md` - Updated with search patterns and API reference
  - `README.md` - Added search command usage documentation

### Learnings

**People API SearchContacts pattern:**
- Use `srv.People.SearchContacts().Query(query).PageSize(30).ReadMask("...").Context(ctx).Do()`
- SearchContacts requires a warmup request with empty query before actual search
- The warmup updates the search cache for better results
- Maximum PageSize is 30 (values greater are capped)

**Search response structure:**
- Response contains `Results` array with `Person` objects
- Each Result has `Person` field that may be nil (need to check)
- Person contains ResourceName and requested fields

**ReadMask for search:**
- Use comma-separated field names: `names,phoneNumbers,emailAddresses,organizations,biographies`
- Only requested fields are returned
- Must match field names exactly (case-sensitive)

**CLI output formatting:**
- `text/tabwriter` provides aligned column output for tables
- Use `tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)` for tab-aligned output
- Remember to call `w.Flush()` (defer is common pattern)
- Truncate long strings with `...` suffix for clean display

**Resource name handling:**
- Resource names are in format `people/cXXXXXXXXX`
- Extract just the ID part (`cXXXXXXXXX`) for display
- GetContact accepts both full resource name and just the ID

---

## 2026-01-14 - US-00007 - google-contacts: Show contact details command

**Status:** Completed successfully

### What was implemented
Implemented the 'show' command to display full contact details by ID, including all phone numbers, all email addresses with their type labels, and metadata.

**Features:**
- Accepts both full resource name (`people/c123456789`) and just the ID (`c123456789`)
- Displays first and last name breakdown under display name
- Shows all phone numbers with type labels (mobile, work, home, etc.)
- Shows all email addresses with type labels
- Displays company and position
- Handles multiline notes with proper indentation
- Shows last update timestamp from metadata

### Files changed
- **Modified:**
  - `internal/cli/cli.go` - Added showCmd, runShow handler, displayFullContactDetails(), formatTime()
  - `internal/contacts/service.go` - Added ContactDetails, PhoneEntry, EmailEntry types and GetContactDetails() method
  - `CLAUDE.md` - Updated with show command patterns and ContactDetails documentation
  - `README.md` - Added show command usage documentation with example output

### Learnings

**Multiple entries display pattern:**
- When displaying lists (phones, emails), check if single or multiple
- Single item: inline format `Phone: +33... (mobile)`
- Multiple items: list format with bullet points
- Use `•` (Unicode bullet) for clean list formatting

**ContactDetails vs SearchResult:**
- `SearchResult` - First phone/email only, used for search results and tables
- `ContactDetails` - All phones/emails with labels, used for detailed view
- Separate types allow different use cases without bloating the simpler struct

**People API Metadata:**
- `p.Metadata.Sources` contains array of source objects
- Filter by `source.Type == "CONTACT"` for user-created contacts
- `source.UpdateTime` is in ISO 8601 format: "2026-01-14T10:30:00.123456Z"
- CreatedTime is not directly available; UpdateTime reflects last modification

**Phone/Email type labels:**
- Google People API uses `Type` field for labels
- Common types: "mobile", "work", "home", "other"
- Empty type should default to "other" for display consistency

**ISO 8601 timestamp formatting:**
- Simple string slicing works well for display: `date[:10]` for date, `time[11:19]` for time
- Avoids complex time parsing when you just need human-readable output
- Format example: "2026-01-14 10:30:00"

**Cobra command args validation:**
- Use `Args: cobra.ExactArgs(1)` for required positional argument
- Provides automatic error message: "Error: accepts 1 arg(s), received 0"

---

## 2026-01-14 - US-00008 - google-contacts: Complete documentation

**Status:** Completed successfully

### What was implemented
Verified and finalized all project documentation. Both README.md and CLAUDE.md were already comprehensive from previous stories. Only minor update needed.

### Files changed
- **Modified:**
  - `CLAUDE.md` - Removed "(future)" comment from contacts package description as the People API service is now fully implemented

### Documentation verification
All acceptance criteria were already met:

**README.md:**
- ✅ Installation instructions (clone, make build, make install)
- ✅ Usage examples for create command with all flags
- ✅ Usage examples for search command with various query types
- ✅ Usage examples for show command with example output
- ✅ Credential setup (Google Cloud project, People API, OAuth credentials)
- ✅ Credential sharing explanation with email-manager

**CLAUDE.md:**
- ✅ Project structure with file tree
- ✅ Development workflow (build, test, fmt, check targets)
- ✅ Duplicated auth code rationale (simpler deployment, independent builds, no versioning conflicts)

### Learnings

**Documentation completeness in incremental development:**
- When implementing features incrementally (US-00005, US-00006, US-00007), documenting each feature as it's built results in naturally complete documentation
- By the time US-00008 was reached, documentation was already comprehensive
- The dedicated documentation story serves as a verification checkpoint rather than a major implementation effort

**Outdated comments:**
- Watch for comments like "(future)" that become stale as features are implemented
- Good practice to update comments as part of feature implementation, not just in documentation stories

---

## 2026-01-14 - US-00009 - google-contacts: Unit tests

**Status:** Completed successfully

### What was implemented
Implemented unit tests for core CLI and service functionality without requiring network calls.

**Test files created:**
- `internal/cli/cli_test.go` - Tests for CLI utility functions
- `internal/contacts/service_test.go` - Tests for service types and validation

### Files changed
- **Created:**
  - `internal/cli/cli_test.go` - CLI utility function tests
  - `internal/contacts/service_test.go` - Service type and validation tests
- **Modified:**
  - `CLAUDE.md` - Added comprehensive testing documentation with patterns and guidelines
  - `README.md` - Added testing section with commands and test file locations

### Test coverage

**CLI tests (`internal/cli/cli_test.go`):**
- `TestExtractID` - Resource name to ID extraction
- `TestTruncate` - String truncation for table display
- `TestFormatTime` - ISO 8601 timestamp formatting
- `TestValidateRequiredFlags` - Create command field validation

**Service tests (`internal/contacts/service_test.go`):**
- `TestExtractID` - Resource name parsing
- `TestContactInput_Validation` - ContactInput struct validation
- `TestSearchResult_Fields` - SearchResult struct field access
- `TestContactDetails_PhoneEntries` - Multiple phone/email entries
- `TestPhoneEntry_EmptyType` - Default type handling
- `TestEmailEntry_EmptyType` - Default type handling
- `TestCreatedContact_Fields` - CreatedContact struct
- `TestResourceNameNormalization` - Resource name prefix logic

### Learnings

**Table-driven tests pattern:**
- Use `[]struct{ name string; input; expected }` for comprehensive test cases
- `t.Run(tc.name, func(t *testing.T) {...})` provides clear test output
- Makes adding new test cases trivial - just add to the slice

**Testing without API calls:**
- Extract pure functions (like `extractID`, `truncate`, `formatTime`) for easy testing
- Create validation helper functions in test files to test business logic
- Test struct field access to ensure type definitions are correct

**Boundary condition testing:**
- Test edge cases: empty strings, exact length matches, boundary values
- The `extractID` function uses `len(resourceName) > 7` not `>= 7`
- "people/" (exactly 7 chars) doesn't get stripped - test exposed this behavior

**Duplicate functions:**
- Both `cli.go` and `service.go` have `extractID()` functions
- Tests exist for both to ensure consistent behavior
- Consider refactoring to a shared utility if project grows

**Test file placement:**
- Go convention: `*_test.go` in same package as code being tested
- Tests have access to unexported functions in the same package
- `make test` runs `go test -v ./...` to find all test files

---

## 2026-01-14 - US-00010 - google-contacts: Create skill in ~/.claude/skills

**Status:** Completed successfully

### What was implemented
Verified and documented the Claude skill integration for google-contacts. The skill directory at `~/.claude/skills/google-contacts/` was already created with complete SKILL.md and symlink to the binary. Added documentation about the skill in both CLAUDE.md and README.md.

### Files changed
- **Modified:**
  - `CLAUDE.md` - Added Claude Skill Integration section with skill structure, features, and symlink management
  - `README.md` - Added Claude Skill Integration section with installation and usage examples

### Skill structure verified
```
~/.claude/skills/google-contacts/
├── SKILL.md           # Comprehensive skill definition (381 lines)
└── scripts/
    └── google-contacts  # Symlink to bin/google-contacts-linux-amd64
```

### SKILL.md contents verified
- Proper frontmatter with `name` and `description` fields
- Trigger phrases documented (create contact, search contact, show details, screenshot)
- All CLI commands documented: create, search, show
- Natural language parsing examples
- Screenshot workflow with confirmation step
- Error handling patterns
- Authentication notes

### Learnings

**Claude skill structure:**
- Skills are located in `~/.claude/skills/<skill-name>/`
- SKILL.md uses YAML frontmatter with `name` and `description` fields
- The `description` field controls when Claude loads the skill (trigger phrases)
- Scripts can use symlinks to binaries for easier updates

**Skill discovery:**
- Claude automatically discovers skills in `~/.claude/skills/`
- Skills appear in the Skill tool's "Available skills" list
- The description in frontmatter is used for skill matching

**Symlink management:**
- Symlinks allow updating the binary without modifying skill configuration
- Use absolute paths for symlinks: `ln -sf /absolute/path/to/binary scripts/binary`
- After `make build`, the symlink continues to point to the correct binary

**Documentation duplication:**
- Skill documentation (SKILL.md) is separate from project documentation (README.md, CLAUDE.md)
- Both should document the same commands but for different audiences:
  - SKILL.md: For Claude AI to understand how to use the tool
  - README.md: For human users to understand how to use the CLI directly
  - CLAUDE.md: For AI developers working on the codebase

**Bilingual skills:**
- The SKILL.md is written in both French and English
- This supports multilingual user interactions
- Example triggers in both languages help Claude recognize requests

---

## 2026-01-14 - US-00012 - google-contacts skill: Create contact from screenshot

**Status:** Completed successfully (already implemented)

### What was implemented
Verified that the SKILL.md at `~/.claude/skills/google-contacts/` already contains comprehensive documentation for creating contacts from screenshots. The screenshot workflow was fully documented as part of US-00010.

### Acceptance criteria verification

All criteria were already met in existing SKILL.md:

1. **Screenshot workflow section** ✅
   - Section "2. Créer un contact depuis une capture d'écran" (line 116)
   - Complete workflow with 5 steps (lines 120-177)

2. **Read tool usage for images** ✅
   - Étape 2 shows how to use Read tool (lines 131-137)
   - Explains Claude's multimodal image analysis capability

3. **Confirmation step documented** ✅
   - Étape 4 marked as "IMPORTANT" (lines 149-164)
   - Example confirmation dialog provided
   - Clear statement: "Toujours confirmer les données extraites avant de créer le contact"

4. **~/Downloads/ path handling** ✅
   - "Chemins d'images courants" section (lines 189-193)
   - Explicitly lists `~/Downloads/` as common path

### Additional documentation already present
- Common image sources table (business cards, email signatures, LinkedIn profiles)
- Workflow 2 in the "Workflows courants" section showing complete screenshot-to-contact flow
- Trigger phrases including "Add contact from this screenshot"

### Files changed
- **Modified:**
  - `stories.yaml` - Updated US-00012 `passes: false` to `passes: true`
  - `progress.md` - Added this entry

### Learnings

**Comprehensive skill documentation from the start:**
- US-00010 created a very thorough SKILL.md that anticipated all subsequent stories
- Screenshot workflow was documented with all required elements upfront
- This pattern is efficient but requires careful story sequencing awareness

**Confirmation step importance:**
- The skill documentation correctly marks the confirmation step as "IMPORTANT"
- This is critical for screenshot-based contact creation where OCR/extraction may have errors
- User verification prevents creating contacts with incorrect information

**Image source diversity:**
- The documentation covers multiple image sources (business cards, email signatures, LinkedIn, phone screenshots)
- Each source has different typical information available
- This helps Claude extract the right fields based on the image context

**Path handling patterns:**
- `~/Downloads/` is the most common location for screenshots
- `/tmp/` for temporary files
- Absolute paths for user-specified locations
- These patterns help Claude locate image files from user requests

---

## 2026-01-14 - US-00011 - google-contacts skill: Create contact from basic information

**Status:** Completed successfully (already implemented)

### What was implemented
Verified that the SKILL.md at `~/.claude/skills/google-contacts/` already contains comprehensive documentation for creating contacts from basic information. The skill was fully implemented as part of US-00010.

### Acceptance criteria verification

All criteria were already met in existing SKILL.md:

1. **Create contact workflow** ✅
   - Section "1. Créer un contact" (line 52)
   - "Workflow 1: Créer un contact depuis une demande texte" (line 283)

2. **Natural language parsing examples** ✅
   - "Workflow de parsing naturel" section (line 86)
   - Parsing examples table with input/extraction columns (lines 108-114)

3. **Required fields documented** ✅
   - "Champs requis" section documents: firstname, lastname, phone (lines 68-71)

4. **Optional fields documented** ✅
   - "Champs optionnels" section documents: email, company, position, notes (lines 73-77)

### Files changed
- **Modified:**
  - `stories.yaml` - Updated US-00011 `passes: false` to `passes: true`
  - `progress.md` - Added this entry

### Learnings

**Story verification vs implementation:**
- Stories may already be implemented as part of earlier work
- US-00011's requirements were naturally fulfilled when creating the comprehensive SKILL.md in US-00010
- The verification step confirms completeness without requiring new code

**Skill documentation scope:**
- A well-structured SKILL.md should cover both basic and advanced workflows
- Including natural language parsing examples in the initial skill creation prevents fragmented documentation
- Bilingual examples (French/English) were included from the start to support multilingual interactions

**Incremental story completion:**
- When creating skills, it's efficient to document all use cases upfront
- Subsequent stories (US-00011, US-00012, US-00013) serve as verification checkpoints
- This pattern ensures nothing is missed while allowing focused review of each capability

---
