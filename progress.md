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

## 2026-01-14 - US-00013 - google-contacts skill: Search and retrieve contacts

**Status:** Completed successfully (already implemented)

### What was implemented
Verified that the SKILL.md at `~/.claude/skills/google-contacts/` already contains comprehensive documentation for searching and retrieving contacts. The search and show workflows were fully documented as part of US-00010.

### Acceptance criteria verification

All criteria were already met in existing SKILL.md:

1. **Search workflow section** ✅
   - Section "3. Rechercher des contacts" (line 194)
   - Examples for name, company, and phone searches
   - Result behavior table for 0, 1, or multiple results

2. **Show command section** ✅
   - Section "4. Afficher les détails d'un contact" (line 238)
   - ID format options documented (full or ID only)
   - Complete example output with all fields

3. **Single result auto-show** ✅
   - "Comportement selon les résultats" table (lines 230-237)
   - Explicitly states: "1 | Affiche automatiquement les détails complets"

4. **Field extraction examples** ✅
   - "Workflow 3" (lines 313-322)
   - Shows extracting specific field (phone number) from search results
   - Complete workflow from user question to response

### Files changed
- **Modified:**
  - `stories.yaml` - Updated US-00013 `passes: false` to `passes: true`
  - `internal/cli/cli_test.go` - gofmt formatting fixes for struct alignment
  - `progress.md` - Added this entry

### Learnings

**Comprehensive skill documentation pattern:**
- When creating a skill (US-00010), documenting all workflows upfront is efficient
- Stories US-00011, US-00012, and US-00013 all verified already-implemented documentation
- This pattern reduces implementation time but requires careful planning upfront

**gofmt struct alignment:**
- Running `make check` includes `gofmt` which auto-aligns struct fields
- When a struct has fields of different lengths, gofmt aligns all colons
- Example: `name string` and `errContains string` get aligned to longest field name

**Verification vs implementation stories:**
- Some stories end up being verification checkpoints rather than new implementations
- This is valid when earlier stories naturally fulfill later requirements
- The verification step confirms completeness and catches any gaps

**SKILL.md as complete reference:**
- A well-structured SKILL.md should be self-contained
- Includes: workflows, command syntax, examples, error handling, authentication
- The document serves both as Claude's guide and as human-readable documentation

---

## 2026-01-14 - US-00014 - google-contacts: Delete contact command

**Status:** Completed successfully

### What was implemented
Implemented the 'delete' command to remove contacts from Google Contacts with safety features.

**Features:**
- Delete by contact ID (full resource name or just the ID)
- Displays contact summary before deletion
- Confirmation prompt by default (y/N)
- --force flag to skip confirmation for scripting
- Clear success message with contact name

### Files changed
- **Modified:**
  - `internal/contacts/service.go` - Added DeleteContact method
  - `internal/cli/cli.go` - Added deleteCmd, deleteForce flag, runDelete handler, displayDeleteSummary function
  - `CLAUDE.md` - Updated command structure, service methods, and added delete API pattern
  - `README.md` - Added delete command documentation with example output

### Learnings

**People API DeleteContact pattern:**
- Use `srv.People.DeleteContact(resourceName).Context(ctx).Do()`
- Returns `(*Empty, error)` - need to capture both values even though Empty is unused
- No response body on success (returns empty)
- Returns error if contact not found

**User confirmation in CLI:**
- Use `fmt.Scanln(&response)` for simple yes/no confirmation
- Default to "No" (y/N pattern) for destructive operations
- Normalize response with `strings.ToLower(strings.TrimSpace(response))`
- Accept both "y" and "yes" for flexibility

**--force flag pattern:**
- Common pattern for destructive CLI commands
- Use `BoolVarP(&deleteForce, "force", "f", false, "...")` for both long and short flags
- Best practice: always show summary before even with --force
- --force only skips confirmation, not the display

**Safety features for delete commands:**
- Fetch contact details BEFORE deletion (for confirmation display)
- Show minimal but identifying information (name, ID, phone, email, company)
- Use warning colors (yellow for "Contact to delete:")
- Explicit success message with contact name for audit trail

**Makefile cache behavior:**
- `make build` uses Go's build cache - won't rebuild unchanged code
- `make clean && make build` forces complete rebuild
- After code changes, ensure rebuild to test new binary
- The cache is usually correct but can cause confusion during development

---

## 2026-01-14 - US-00015 - google-contacts: Update contact command

**Status:** Completed successfully

### What was implemented
Implemented the 'update' command to modify existing contacts with selective field updates using the People API.

**Features:**
- All contact fields can be updated: firstname, lastname, phone, email, company, position, notes
- Only specified fields are modified; unspecified fields remain unchanged
- Before/after display highlights changed fields with colored arrows (→)
- Validates at least one field is specified before making API calls
- Uses pointer types in UpdateInput to distinguish "not provided" from "empty value"

### Files changed
- **Modified:**
  - `internal/contacts/service.go` - Added UpdateInput type and UpdateContact method
  - `internal/cli/cli.go` - Added updateCmd, update flags, runUpdate handler, displayUpdateSummary function
  - `CLAUDE.md` - Added update command to command structure, service methods, and API patterns
  - `README.md` - Added update command documentation with flags table and example output

### Learnings

**Pointer types for optional updates:**
- Use `*string` instead of `string` to distinguish "not provided" (nil) from "provided empty" ("")
- Cobra's `cmd.Flags().Changed("flagname")` checks if flag was explicitly set by user
- This pattern allows updating a field to empty string if desired

**People API UpdateContact pattern:**
```go
updated, err := srv.People.UpdateContact(resourceName, person).
    UpdatePersonFields("names,phoneNumbers").  // Comma-separated field names
    PersonFields("names,phoneNumbers").         // Fields to return in response
    Context(ctx).
    Do()
```

**UpdatePersonFields mask:**
- Specifies which fields to modify in the contact
- Only listed fields are updated; others are preserved
- Field names are the same as PersonFields: `names`, `phoneNumbers`, `emailAddresses`, `organizations`, `biographies`

**Fetch before update pattern:**
- Fetch current contact data before making changes
- Merge new values into existing Person object
- Build UpdatePersonFields mask based on which fields changed
- Display before/after comparison for user verification

**Before/after display pattern:**
- Compare before and after values for each field
- Show arrow (→) only when value changed: `old → new`
- Use color coding: yellow for old value, green for new value
- Skip arrow when value unchanged, just show current value

**Flag ordering in CLI:**
- Check for required inputs BEFORE making API calls to fail fast
- Better UX: "no fields specified" error appears immediately
- Avoids unnecessary network calls when command is incomplete

**String import in service.go:**
- Added `strings` import for `strings.Join(updateFields, ",")`
- Used to build the UpdatePersonFields comma-separated list

---

## 2026-01-14 - US-00016 - google-contacts: Multiple phones support in create/update

**Status:** Completed successfully

### What was implemented
Added support for multiple phone numbers with types (labels) in both create and update commands.

**Features:**
- Create command: `--phone` flag can be repeated multiple times
- Phone format: `type:number` or just `number` (defaults to mobile)
- Valid phone types: mobile (default), work, home, main, other
- Update command: new flags `--phones`, `--add-phone`, `--remove-phone`
  - `--phone`: backward compatible, replaces first phone only
  - `--phones`: replaces ALL phones (can be repeated)
  - `--add-phone`: adds phones without removing existing
  - `--remove-phone`: removes specific phones by value

### Files changed
- **Modified:**
  - `internal/contacts/service.go` - Changed ContactInput.Phone to ContactInput.Phones []PhoneEntry, extended UpdateInput with Phones, AddPhones, RemovePhones fields
  - `internal/cli/cli.go` - Added parsePhones function, updated create/update commands with new flags
  - `internal/cli/cli_test.go` - Added TestParsePhones with comprehensive test cases
  - `internal/contacts/service_test.go` - Updated ContactInput validation tests for multiple phones
  - `CLAUDE.md` - Updated ContactInput documentation, added phone types section, updated UpdateInput documentation
  - `README.md` - Updated create and update command documentation with multiple phone examples

### Learnings

**Cobra StringArray vs StringSlice:**
- `StringArrayVarP` collects multiple flag occurrences into a slice
- Each `-p value` adds one element to the slice
- Use for flags that should be repeated: `--phone "mobile:123" --phone "work:456"`

**Phone parsing with type prefix:**
- Format: `type:number` where type is optional (defaults to mobile)
- Use `strings.Index(s, ":")` to find separator position
- Extract type and value: `type = s[:idx]`, `value = s[idx+1:]`
- Validate type against allowed set: mobile, work, home, main, other

**Backward compatibility in APIs:**
- Keep old `--phone` flag working (replaces first phone)
- Add new `--phones` flag for full replacement
- Add `--add-phone` and `--remove-phone` for granular control
- Priority: if multiple phone flags provided, process in defined order

**UpdateInput pointer vs slice distinction:**
- Use `*string` for single value updates (nil = not provided)
- Use `[]PhoneEntry` directly (empty slice = nothing to add)
- Check `len(slice) > 0` to determine if update needed
- Process all phone operations in sequence: Phone → Phones → AddPhones → RemovePhones

**Phone removal by value:**
- Simple approach: compare phone values exactly
- Build new slice with non-matching entries
- Replace original slice with filtered result

**Test coverage for parsing:**
- Test all valid types (mobile, work, home, main, other)
- Test invalid type (fax) returns error
- Test empty phone value
- Test case insensitivity (WORK → work)
- Test mixed formats (with type and without)
- Test empty input

---

## 2026-01-14 - US-00018 - google-contacts skill: Name recognition improvements

**Status:** Completed successfully

### What was implemented
Added a comprehensive "Reconnaissance des noms" (Name Recognition) section to SKILL.md documenting heuristics for identifying first names vs last names with confidence levels.

### Files changed
- **Modified:**
  - `~/.claude/skills/google-contacts/SKILL.md` - Added Name Recognition section (lines 335-432)
  - `stories.yaml` - Updated US-00018 `passes: false` to `passes: true`
  - `progress.md` - Added this entry

### Documentation added

**Name recognition rules:**
1. **ALL CAPS rule (HIGH confidence >90%)** - Word in ALL CAPS is likely the family name
   - Example: "Sebastien LAURENT" → firstname=Sebastien, lastname=Laurent
2. **French order (MEDIUM confidence 60-90%)** - Default firstname-lastname order
   - Example: "Jean Dupont" → firstname=Jean, lastname=Dupont
3. **Asian/ambiguous names (LOW confidence <60%)** - Multiple possibilities
   - Example: "Takeshi Yamamoto" → could be either order
4. **Multiple words rule** - Compound first names are more common

**Confidence levels and actions:**
- HIGH: Present interpretation + ask confirmation
- MEDIUM: Present with explanation + ask confirmation
- LOW: Present options + ask user to choose

### Learnings

**Name recognition heuristics:**
- ALL CAPS is a strong indicator of family name in European contexts (particularly French)
- This convention is common in formal documents, business cards, and administrative systems
- The heuristic works because ALL CAPS is intentionally used to distinguish the family name

**Cultural context matters:**
- French names typically follow firstname-lastname order
- Asian names often follow lastname-firstname order
- Without cultural context, confidence should be LOW

**Probabilistic reasoning in skills:**
- Skills should express uncertainty when it exists
- Confidence levels help users understand the reasoning
- Even with HIGH confidence, validation should still be required

**SKILL.md structure:**
- Adding cross-references between sections (e.g., "see Validation section") helps maintain consistency
- The section references a "Validation obligatoire avant création" section that will be added in US-00019

---

## 2026-01-14 - US-00019 - google-contacts skill: Mandatory validation before contact creation

**Status:** Completed successfully

### What was implemented
Added comprehensive mandatory validation workflow documentation to SKILL.md to ensure all contact creation is validated by the user before execution.

### Files changed
- **Modified:**
  - `~/.claude/skills/google-contacts/SKILL.md` - Added "Validation obligatoire avant création" section (~200 lines)
  - `CLAUDE.md` - Added skill features (validation, name recognition)
  - `internal/cli/cli.go` - gofmt formatting fixes
  - `internal/contacts/service.go` - gofmt formatting fixes
  - `stories.yaml` - Updated US-00019 `passes: false` to `passes: true`
  - `progress.md` - Added this entry

### SKILL.md sections added/updated

**New section "Validation obligatoire avant création":**
- Principle: ALL contact creation requires user validation (NO EXCEPTIONS)
- Why validation is mandatory (5 reasons)
- Standard validation prompt format with field alignment
- Multi-phone/email format example
- Accepted user responses table (confirm/cancel/modify)
- Modification flow with re-validation loop
- Integrated workflow examples (text + screenshot)
- Special cases: low confidence names, deduced fields
- "What NOT to do" checklist (5 anti-patterns)

**Updated workflows to include validation:**
- "Workflow de parsing naturel" (lines 86-118) - Added step 2 validation prompt
- "Étape 4" screenshot workflow (lines 161-179) - Renamed to "Validation OBLIGATOIRE"
- "Workflow 1" text creation (lines 283-311) - Added complete validation flow
- "Workflow 2" screenshot creation (lines 313-340) - Added complete validation flow

### Learnings

**Skill documentation for safety-critical workflows:**
- Making validation MANDATORY (vs recommended) requires explicit language: "OBLIGATOIRE", "SANS EXCEPTION"
- Anti-patterns (what NOT to do) are as important as positive examples
- Every workflow path must include the validation step - no shortcuts
- Cross-references between sections help maintain consistency

**Validation prompt design:**
- Field alignment with consistent spacing improves readability
- Source attribution ("Extrait de l'image...") provides context
- Clear options (oui/non/modifier) reduce ambiguity
- Include phone/email types in the prompt for full transparency

**User response handling:**
- Accept multiple synonyms for each action (oui/yes/ok/parfait)
- Modification triggers re-validation loop - don't skip
- Distinguish between user uncertainty ("je pense que oui") and clear confirmation

**SKILL.md organization:**
- Large skill documents benefit from clear section headers and sub-headers
- Reference earlier sections to avoid duplication
- Use tables for quick reference (response types, confidence levels)
- Include both positive examples and anti-patterns

**Documentation-only stories:**
- Some stories only modify skill documentation, not CLI code
- These still require updating CLAUDE.md to keep features list current
- gofmt may still run during `make check` and format code

---
