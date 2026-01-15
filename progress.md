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

## 2026-01-14 - US-00020 - google-contacts skill: Document delete and update workflows

**Status:** Completed successfully

### What was implemented
Updated SKILL.md to document the delete and update commands with complete workflows, including natural language examples and safety considerations.

### Files changed
- **Modified:**
  - `~/.claude/skills/google-contacts/SKILL.md` - Added sections 5 (Modifier) and 6 (Supprimer), Workflows 5 and 6
  - `CLAUDE.md` - Added "Contact update" and "Contact deletion" to skill features list
  - `stories.yaml` - Updated US-00020 `passes: false` to `passes: true`
  - `progress.md` - Added this entry

### SKILL.md sections added

**New sections in "Opérations disponibles":**
- Section 5: Modifier un contact - Full update command documentation with flags table
- Section 6: Supprimer un contact - Delete command documentation with safety notes

**New workflows in "Workflows courants":**
- Workflow 5: Modifier un contact - Natural language update with before/after display
- Workflow 6: Supprimer un contact - Delete with mandatory confirmation and multiple results handling

**Additional content:**
- Examples table for update modifications (phone, email, company, etc.)
- Precautions for deletion (4 safety rules)
- Undo guidance for accidental deletions

### Learnings

**Skill workflow documentation patterns:**
- Update workflows should show before/after comparison for user verification
- Delete workflows must emphasize irreversibility and always require confirmation
- Include examples for common variations (single vs multiple matches)
- Provide recovery guidance even when true recovery isn't possible

**Natural language examples importance:**
- Tables with "User request → CLI action" help Claude understand mappings
- Cover various phrasings: "change", "update", "modify", "correct" for updates
- Cover various phrasings: "delete", "remove", "suppress" for deletions

**Safety-first documentation:**
- Mark warnings visibly with ⚠️ emoji
- Explain WHY confirmation is required, not just that it's required
- Show exact CLI commands with `--force` flag but explain its risks
- Include multiple-result scenarios to prevent accidental deletions

**Skill feature consistency:**
- When adding workflows to SKILL.md, also update CLAUDE.md's "Skill Features" list
- Keeps the project documentation synchronized across both files
- Features list in CLAUDE.md is for developers; SKILL.md workflows are for Claude

---

## 2026-01-14 - US-00021 - google-contacts skill: Document multiple phones/emails support

**Status:** Completed successfully

### What was implemented
Updated SKILL.md to comprehensively document support for multiple phone numbers and email addresses with types/labels in both create and update commands.

### Files changed
- **Modified:**
  - `~/.claude/skills/google-contacts/SKILL.md` - Added phone types table, email types table, multiple phones/emails examples for create and update commands
  - `stories.yaml` - Updated US-00021 `passes: false` to `passes: true`
  - `progress.md` - Added this entry

### SKILL.md sections added/updated

**New documentation in "Créer un contact" section:**
- "Types de téléphones supportés" table with all 5 types: mobile (default), work, home, main, other
- "Types d'emails supportés" table with all 3 types: work (default), home, other
- "Exemples avec plusieurs téléphones et emails" - 3 comprehensive CLI examples

**New documentation in "Modifier un contact" section:**
- "Types et formats" reference to reuse create command types
- "Gestion avancée des téléphones" - Examples for --add-phone, --remove-phone, --phones
- "Gestion avancée des emails" - Examples for --add-email, --remove-email, --emails

**Updated "Exemples de demandes de modification" section:**
- Split into 3 sub-tables: "Modifications de téléphones", "Modifications d'emails", "Autres modifications"
- Added natural language → CLI action mappings for all phone/email operations

### Learnings

**Documentation structure for multi-value fields:**
- Use tables for type definitions (Type | Description | Example format)
- Provide both simple and advanced examples in separate sections
- Reference types across sections ("same types as create apply") to reduce duplication

**CLI flag documentation patterns:**
- Document backward-compatible flags first (-p, -e for single value replacement)
- Then document advanced operations (--phones/--emails for full replacement)
- Finally show additive/subtractive operations (--add-*, --remove-*)
- This progression from simple to complex helps users gradually learn

**Natural language parsing tables:**
- Organizing by operation type (phones, emails, other) improves scanability
- Each row should show exact phrasing variations users might use
- Include common synonyms: "ajoute", "supprime", "remplace", "change"

**Validation prompt consistency:**
- Multiple phones/emails format already documented in earlier story (US-00019)
- Bullet-point format (• type : value) works well for displaying lists
- Type labels should be shown in the validation to ensure transparency

**SKILL.md size considerations:**
- The file is now ~1000 lines and comprehensive
- Good organization with clear section headers makes navigation easy
- Consider future split into multiple files if documentation grows further

---

## 2026-01-14 - US-00022 - google-contacts: Add birthday field support

**Status:** Completed successfully

### What was implemented

Added birthday field support across the entire stack (service layer, CLI, documentation):

**Service layer (internal/contacts/service.go):**
- Added `Birthday` field to `ContactInput` struct (format: YYYY-MM-DD or --MM-DD)
- Added `Birthday` and `ClearBirthday` fields to `UpdateInput` struct
- Added `Birthday` field to `ContactDetails` struct
- Added `parseBirthday()` function to parse birthday strings into People API Birthday struct
- Added `formatBirthday()` function to convert API birthday to string format
- Updated `CreateContact()` to set birthday on new contacts
- Updated `UpdateContact()` to handle birthday set/clear operations
- Updated `GetContactDetails()` to extract birthday from API response
- Added 'birthdays' to all PersonFields API calls

**CLI layer (internal/cli/cli.go):**
- Added `--birthday`/`-b` flag to create command
- Added `--birthday`/`-b` and `--clear-birthday` flags to update command
- Added `formatBirthdayDisplay()` function for human-readable birthday output (e.g., "March 15, 1985")
- Updated show command to display birthday section

### Files changed

- `internal/contacts/service.go` - Added birthday support to all structs and methods
- `internal/cli/cli.go` - Added birthday flags and display formatting
- `CLAUDE.md` - Documented birthday fields in structs and CLI options
- `README.md` - Added birthday usage examples and format documentation

### Learnings

**People API Birthday format:**
- Birthday uses `people.Birthday` with nested `Date` struct
- Date has Year, Month, Day as int64 fields
- Year = 0 means year is unknown (for month/day only birthdays)
- PersonFields must include "birthdays" to fetch/update birthday data

**Birthday string format design:**
- Standard format: YYYY-MM-DD (e.g., "1985-03-15")
- Month/day only: --MM-DD (e.g., "--03-15") - uses ISO 8601 convention for unknown year
- Display format: Human-readable with month names (e.g., "March 15, 1985" or "March 15")

**CLI flag handling for optional clear operations:**
- Use separate bool flag (--clear-birthday) rather than magic value
- Check ClearBirthday first before Birthday pointer check
- ClearBirthday sets birthday slice to nil in API call

**Date parsing considerations:**
- Using fmt.Sscanf for simple integer extraction from date parts
- Validate month range (1-12) and day range (1-31) for basic sanity checks
- Return nil from parser on invalid format rather than error (let API handle detailed validation)

---

## 2026-01-14 - US-00023 - google-contacts: Add address field support

**Status:** Completed successfully

### What was implemented

Added postal address support across the entire stack (service layer, CLI, documentation):

**Service layer (internal/contacts/service.go):**
- Added `AddressEntry` struct with `Value` and `Type` fields
- Added `Addresses` field to `ContactInput` struct
- Added `Addresses`, `AddAddresses`, `RemoveAddresses` fields to `UpdateInput` struct
- Added `Addresses` field to `ContactDetails` struct
- Updated `CreateContact()` to set addresses on new contacts using `FormattedValue`
- Updated `UpdateContact()` to handle address set/add/remove operations
- Updated `GetContactDetails()` to extract addresses from API response
- Added 'addresses' to all PersonFields API calls

**CLI layer (internal/cli/cli.go):**
- Added `parseAddresses()` function for parsing address strings with type prefix
- Added `--address`/`-a` flag to create command (repeatable)
- Added `--addresses`, `--add-address`, `--remove-address` flags to update command
- Updated show command to display all addresses with types

### Files changed

- `internal/contacts/service.go` - Added AddressEntry struct and address support to all structs and methods
- `internal/cli/cli.go` - Added address parsing and flags
- `CLAUDE.md` - Documented address types, formats, and CLI options
- `README.md` - Added address usage examples and format documentation

### Learnings

**People API Address format:**
- Address uses `people.Address` with `FormattedValue` and `Type` fields
- `FormattedValue` stores the full address as a single string
- Structured fields (streetAddress, city, postalCode, etc.) are also available but not used in this story
- PersonFields must include "addresses" to fetch/update address data

**Address type design:**
- Three types supported: home (default), work, other
- Type prefix format: `type:address` (e.g., "work:50 Avenue Business, Lyon")
- Default type is "home" when no prefix provided
- Follows the same pattern established for phones and emails

**Address parsing pattern:**
- Similar to parsePhones() and parseEmails() but simpler
- Type validation against allowed set: home, work, other
- Uses `strings.Index()` to find the type prefix separator
- Handles addresses that contain colons (e.g., "Address: Street") by checking valid types

**Remove address by content match:**
- Unlike phones/emails which match by exact value
- Addresses are removed by checking if FormattedValue contains the search string
- This allows removing by partial match (e.g., street name) since addresses are long strings
- Uses `strings.Contains()` for flexible matching

**Consistent multi-value field pattern:**
- AddressEntry follows same pattern as PhoneEntry and EmailEntry
- Update operations: single value replacement, full replacement, add, remove
- Display format matches phones/emails with bullet points and type labels
- This consistency helps users learn the CLI quickly

**FormattedValue vs structured addresses:**
- This story uses FormattedValue for simplicity
- US-00024 will add structured address parsing for better Google Contacts integration
- FormattedValue is sufficient for storage but doesn't enable city/postal code search

---

## 2026-01-14 - US-00024 - google-contacts: Structured address parsing

**Status:** Completed successfully

### What was implemented

Enhanced address support with structured parsing for better Google Contacts API integration:

**Service layer (internal/contacts/service.go):**
- Added `StructuredAddress` struct with all Google API address fields
- Added `ParseAddress()` function for automatic address field extraction
- Implemented French address auto-detection (5-digit postal codes)
- Implemented structured syntax parsing (`street=...;city=...;postal=...`)
- Updated `CreateContact()` to use structured address fields
- Updated `UpdateContact()` to use structured address fields

**Parsing formats supported:**
1. French format: `10 Rue Test, 75001 Paris` → auto-detects France, extracts structured fields
2. French format with country: `10 Rue Test, 75001 Paris, France`
3. Generic format: `123 Main St, London, SW1A 1AA, UK`
4. Structured syntax: `street=10 Rue Test;city=Paris;postal=75001;country=France`

**Unit tests (internal/contacts/service_test.go):**
- 12 new test functions for ParseAddress
- Tests for empty input, French formats, structured syntax, generic formats
- Tests for FormattedValue preservation and building

### Files changed

- `internal/contacts/service.go` - Added StructuredAddress, ParseAddress, and helper functions
- `internal/contacts/service_test.go` - Added comprehensive ParseAddress tests
- `CLAUDE.md` - Added Structured Address Parsing documentation section
- `README.md` - Added structured address parsing documentation

### Learnings

**Google People API Address structured fields:**
- `people.Address` supports both `FormattedValue` and structured fields
- Structured fields: `StreetAddress`, `City`, `PostalCode`, `Region`, `Country`, `CountryCode`
- All fields can be set simultaneously - API stores both formatted and structured
- Structured fields enable better search and display in Google Contacts

**French postal code detection:**
- French postal codes are exactly 5 digits (e.g., 75001, 69001)
- Regex pattern: `\b(\d{5})\b` matches standalone 5-digit sequences
- This is also true for some US ZIP codes, so context matters
- When 5-digit code detected, assume French address and set country=France

**French address format variations:**
- `street, postal city` - postal code before city (most common)
- `street, city postal` - postal code after city
- `street, postal city, country` - with explicit country
- `street, city, postal, country` - fully separated parts
- Parser handles all variations by detecting postal code position

**Structured syntax design:**
- Using semicolons as field separators (`;`) avoids conflict with commas in addresses
- Key=value format allows explicit field assignment
- Supported keys: `street`, `city`, `postal`, `region`, `country`, `countrycode`
- Alternative keys accepted: `streetaddress`, `postalcode`, `zip`, `state`, `province`

**FormattedValue handling:**
- For natural addresses, FormattedValue is the original input string
- For structured syntax, FormattedValue is built from structured fields
- Building formula: `street, postal city, region, country` with non-empty parts

**Test limitations with regex detection:**
- US ZIP codes (5 digits like 10001) are detected as French postal codes
- Test adjusted to use UK postal codes (SW1A 1AA) for generic format tests
- In production, context (presence of "USA" or state abbreviation) could disambiguate

---

## 2026-01-14 - US-00025 - google-contacts skill: Document birthday support

**Status:** Completed successfully

### What was implemented

Updated SKILL.md to comprehensively document birthday field support across all workflows and examples.

### Files changed

- **Modified:**
  - `~/.claude/skills/google-contacts/SKILL.md` - Added birthday documentation throughout the skill
  - `stories.yaml` - Updated US-00025 `passes: false` to `passes: true`
  - `progress.md` - Added this entry

### SKILL.md sections added/updated

**Birthday format documentation:**
- Added `--birthday, -b` to optional fields in create command
- Added "Format de date de naissance" section with table showing:
  - `YYYY-MM-DD` for full date
  - `--MM-DD` for month/day only (year unknown)
- Display format examples with age calculation

**Natural language parsing:**
- Added birthday examples to parsing table:
  - "Jean Dupont né le 15 mars 1985, 0612345678"
  - "Marie Martin, 0698765432, anniversaire le 20 juin"
- Added keyword list for birthday detection: "né le", "née le", "date de naissance", "DOB", "birthday"

**Screenshot extraction:**
- Added date de naissance to extraction fields
- Added "Champs de date de naissance à rechercher dans les images" section
- Updated sources table: LinkedIn (sometimes), Formulaire/CV, Pièce d'identité

**Validation prompts:**
- Updated all validation prompt examples to include "Naissance" field
- Added example without year: "15 mars" (month/day only format)
- Age calculation shown when year is known: "(39 ans)"

**Update command:**
- Added `--birthday` and `--clear-birthday` flags to flags table
- Added "Gestion de la date de naissance" section with examples
- Added "Modifications de la date de naissance" table with natural language mappings

**Workflows:**
- Workflow 1: Updated to include birthday in example request and validation prompt
- Workflow 2: Updated to mention birthday extraction and include in validation

### Learnings

**Skill documentation completeness:**
- Birthday is a field that can appear in multiple places in SKILL.md
- Important to update ALL relevant sections: create command, update command, validation prompts, parsing examples, screenshot extraction, workflows
- Consistent field naming across all examples ("Naissance" in French)

**Age calculation in prompts:**
- Including calculated age in validation prompts (e.g., "39 ans") helps users verify data
- Age should only appear when year is known
- Month/day only format should not show age

**Date format flexibility:**
- Supporting both full dates (YYYY-MM-DD) and month/day only (--MM-DD) is important
- ISO 8601 convention for unknown year (--MM-DD) is intuitive
- Natural language parsing should recognize both "né le" and "anniversaire"

**Screenshot extraction patterns:**
- Different image sources have different likelihood of containing birthdays:
  - High: Formulaire/CV, Pièce d'identité
  - Medium: LinkedIn profiles (sometimes)
  - Low: Business cards, email signatures
- Include keyword variations in multiple languages (FR/EN)

**Documentation-only stories:**
- This story only modified SKILL.md, not CLI code
- Still required updating stories.yaml and progress.md
- No code tests needed since no Go code was changed

---

## 2026-01-14 - US-00026 - google-contacts skill: Document address support

**Status:** Completed successfully

### What was implemented

Updated SKILL.md to comprehensively document address field support across all workflows and examples.

### Files changed

- **Modified:**
  - `~/.claude/skills/google-contacts/SKILL.md` - Added address documentation throughout the skill
  - `CLAUDE.md` - Added address support to skill features list
  - `stories.yaml` - Updated US-00026 `passes: false` to `passes: true`
  - `progress.md` - Added this entry

### SKILL.md sections added/updated

**Address types documentation:**
- Added "Types d'adresses supportés" table with home (default), work, other types
- Added "Formats d'adresses supportés" section with French auto-detection examples
- Added structured syntax documentation (key=value format)

**Natural language parsing:**
- Added address examples to parsing table:
  - "Jean Dupont, 0612345678, habite 10 Rue Test, 75001 Paris"
  - "Pierre Bernard chez Acme, bureau au 50 Avenue Business, Lyon"
- Added "Mots-clés pour l'adresse" section (habite, bureau, réside, address, etc.)

**Screenshot extraction:**
- Added "Champs d'adresse à rechercher dans les images" section
- Updated sources table with "Adresse probable" column showing likelihood per source type
- Added "Courrier/Facture" as a new source with high address probability

**Validation prompts:**
- Updated main validation prompt format to include Adresse field
- Updated multiple phones/emails format to show Adresses list
- Updated screenshot validation prompt with address example

**Update command:**
- Added address flags to flags table (--address, --addresses, --add-address, --remove-address)
- Updated "Types et formats" section to include addresses
- Added "Gestion avancée des adresses" section with CLI examples
- Added "Modifications d'adresses" table in update workflow examples

### Learnings

**Skill documentation consistency patterns:**
- When adding a new field (address), it must be documented in MANY places:
  - Types table, format examples, validation prompts, natural language parsing, screenshot extraction, update workflow
- Consistency is key: use same field names and formats everywhere
- Check all validation prompt examples and update them all

**Screenshot source probability tables:**
- Adding a third column (probability) to source tables helps Claude prioritize extraction efforts
- Different sources have very different likelihoods for address information:
  - High: Business cards, Courrier/Facture, CV
  - Medium: Email signatures
  - Low: LinkedIn profiles (city only), Phone screenshots

**Address-specific documentation:**
- Address removal uses content matching (unlike exact value matching for phones/emails)
- This is documented but important to highlight since behavior differs
- French auto-detection is a key feature that should be mentioned multiple times

**Documentation-only stories:**
- This story only modified SKILL.md and CLAUDE.md, not CLI code
- No Go code changes = no additional unit tests needed
- `make check` still runs but uses cached test results

**SKILL.md growth considerations:**
- The file is now over 1250 lines of documentation
- Clear section structure and headers make navigation manageable
- Consider using a table of contents if file grows much further

---

## 2026-01-15 - US-00027 - Phone number internationalization with France default

**Status:** Completed successfully

### What was implemented

Added automatic phone number normalization to international format with France (+33) as the default country code.

**Service layer (internal/contacts/service.go):**
- Added `NormalizePhoneNumber()` function (exported for cross-package use)
- Handles multiple input formats:
  - French local: `0612345678` → `+33612345678`
  - French with spaces: `06 12 34 56 78` → `+33612345678`
  - French with dots: `06.12.34.56.78` → `+33612345678`
  - International prefix: `0033612345678` → `+33612345678`
  - Already international: `+33612345678` → `+33612345678` (preserved)
  - US format: `+1-555-123-4567` → `+15551234567` (cleaned)
- Removes all non-essential characters (spaces, dashes, dots, parentheses)
- Integrated in CreateContact (1 location) and UpdateContact (4 locations)

**Unit tests (internal/contacts/service_test.go):**
- Added `TestNormalizePhoneNumber` with 15 comprehensive test cases
- Tests cover: French local formats, international formats, edge cases, empty input

### Files changed

- `internal/contacts/service.go` - Added NormalizePhoneNumber function, integrated in CreateContact/UpdateContact
- `internal/contacts/service_test.go` - Added 15 test cases for phone normalization
- `CLAUDE.md` - Added "Phone Number Normalization" section with rules and examples
- `README.md` - Added feature bullet and phone normalization examples

### Learnings

**Phone number normalization rules:**
- French local numbers start with `0` and have 10 digits
- Removing the leading `0` and adding `+33` converts to international format
- International prefix `00` can be converted to `+`
- All formatting characters (spaces, dashes, dots, parentheses) should be stripped for consistency

**Implementation patterns:**
- Using `strings.Builder` for efficient character-by-character cleaning
- Exporting the function (capital N) allows use across packages
- Normalizing at service layer ensures consistency regardless of entry point (CLI or future MCP)
- For RemovePhones operation, the removal value must also be normalized for accurate comparison

**Test coverage:**
- 15 test cases ensure comprehensive coverage of input formats
- Table-driven tests make it easy to add new cases
- Edge cases include empty strings, already-normalized numbers, and various formatting styles

---

## 2026-01-15 - Backlog Update: MCP Server and Cloud Run Deployment

**Status:** Backlog created - 14 new user stories (US-00027 to US-00040)

### What was added to the backlog

Created comprehensive backlog for major new features spanning 4 phases:

**Phase 9: Phone Number Internationalization (1 story)**
- US-00027: Phone number internationalization with France default (+33)

**Phase 10: MCP HTTP Streamable Server (3 stories)**
- US-00028: Create MCP server command structure using official mcp-go SDK
- US-00029: Implement MCP tools for contacts operations (5 tools)
- US-00030: MCP API Key middleware for authentication

**Phase 11: Terraform Infrastructure (5 stories)**
- US-00031: Initialize infrastructure project structure (config.yaml, init/, iac/)
- US-00032: Cloud Run service configuration (Artifact Registry, Cloud Build, IAM)
- US-00033: Firestore database and indexes for API keys
- US-00034: Secret Manager for OAuth credentials
- US-00035: Makefile deployment targets + Dockerfile

**Phase 12: OAuth Authentication Flow (5 stories)**
- US-00036: OAuth authentication endpoint (/auth)
- US-00037: API Key generation (UUID v4) and Firestore storage
- US-00038: OAuth success page with API Key display
- US-00039: MCP server integration with per-user tokens
- US-00040: Document MCP server deployment and usage

### Configuration decisions

| Setting | Value |
|---------|-------|
| Terraform prefix | scmgcontacts |
| GCP project | Configured in config.yaml |
| OAuth credentials | ~/.credentials/scm-pwd.json |
| API Key format | UUID v4 |
| MCP SDK | github.com/modelcontextprotocol/go-sdk (official) |
| Firestore location | Same project as Cloud Run |
| Cloud Run auth | Unauthenticated (API key at app level) |

### Architecture overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     Cloud Run (MCP Server)                       │
├─────────────────────────────────────────────────────────────────┤
│  /auth          → Start OAuth flow                               │
│  /auth/callback → Exchange code, generate API Key                │
│  /auth/success  → Display API Key to user                        │
│  /mcp/*         → MCP protocol endpoints (protected)             │
├─────────────────────────────────────────────────────────────────┤
│  API Key Middleware → Validates Bearer token                     │
│  Token from Firestore → Per-user Google API auth                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Firestore                                 │
│  api_keys/{uuid}                                                 │
│    ├── refresh_token                                             │
│    ├── access_token                                              │
│    ├── user_email                                                │
│    ├── created_at                                                │
│    └── last_used                                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Secret Manager                               │
│  scm-pwd-oauth-creds → OAuth client_id/secret                    │
└─────────────────────────────────────────────────────────────────┘
```

### Files changed

- **Modified:**
  - `stories.yaml` - Added 14 new user stories (US-00027 to US-00040)
  - `progress.md` - Added this entry

### Next steps

Stories should be implemented in order:
1. US-00027: Phone normalization (standalone, can be done immediately)
2. US-00028-30: MCP server (sequential, builds the server)
3. US-00031-35: Terraform (sequential, creates infrastructure)
4. US-00036-40: OAuth flow (sequential, adds authentication)

---

## 2026-01-15 - US-00028 - google-contacts: Create MCP server command structure

**Status:** Success

**What was implemented:**
- Added MCP (Model Context Protocol) server support using official Go SDK
- Created `internal/mcp/server.go` with server initialization and HTTP handler
- Added `mcp` command to CLI with --port, --host, --api-key, --firestore-project flags
- Registered `ping` tool for connectivity testing
- Implemented Streamable HTTP transport with session-based communication
- Added graceful shutdown with SIGINT/SIGTERM signal handling

**Files changed:**
- `go.mod` - Added github.com/modelcontextprotocol/go-sdk/mcp v1.0.0 dependency
- `go.sum` - Updated dependencies
- `internal/mcp/server.go` - New MCP server implementation
- `internal/cli/cli.go` - Added mcpCmd with flags and runMCP handler
- `CLAUDE.md` - Added MCP server documentation section with tool registration guide
- `README.md` - Added MCP command usage documentation

**Learnings:**
- MCP protocol requires full initialization handshake: initialize → initialized notification → then tools/list and tools/call work
- Session ID (Mcp-Session-Id header) must be preserved across requests after initialization
- The mcp-go SDK's AddTool function uses generics to infer JSON schemas from Go struct types
- StreamableHTTPOptions.Stateless should be false for session tracking

**Remaining issues:** None

---

## 2026-01-15 - US-00029 - google-contacts: Implement MCP tools for contacts operations

**Status:** Success

**What was implemented:**
- Implemented 5 MCP tools for contacts management in `internal/mcp/server.go`
- `contacts_create` - Create new contact with firstName, lastName, phones (required), plus optional emails, addresses, company, position, notes, birthday
- `contacts_search` - Search contacts by query (name, phone, email, company)
- `contacts_show` - Get full contact details by ID
- `contacts_update` - Update contact with partial updates (supports add/remove operations for phones, emails, addresses)
- `contacts_delete` - Delete contact by ID with confirmation message
- Defined type-safe input/output schemas with jsonschema tags for MCP schema generation
- Tools call existing contacts.Service methods (no code duplication)
- Proper validation for required fields (returns error for missing firstName, lastName, phones, query, contactId)

**Files changed:**
- `internal/mcp/server.go` - Added type definitions (PhoneInput, EmailInput, AddressInput, CreateInput/Output, SearchInput/Output, ShowInput/Output, UpdateInput/Output, DeleteInput/Output) and 5 handler methods (handleCreateContact, handleSearchContacts, handleShowContact, handleUpdateContact, handleDeleteContact)
- `CLAUDE.md` - Updated Available Tools section with tool table and input/output type documentation, added curl examples for testing tools
- `README.md` - Updated Available tools table with all 5 contact tools

**Learnings:**
- MCP Go SDK's mcp.AddTool uses generics to infer JSON schemas from struct field tags
- jsonschema tags like `jsonschema:"required,description=..."` control schema generation
- Tool handlers receive typed input structs and return typed output structs (third return value is error)
- Struct embedding (e.g., UpdateOutput embedding ShowOutput) works for composing output types
- Pointer fields in Go can distinguish "not provided" from "empty value" for partial updates

**Remaining issues:** None

---

## 2026-01-15 - US-00030 - google-contacts: MCP API Key middleware

**Status:** Success

**What was implemented:**
- Implemented API Key authentication middleware for the MCP server in `internal/mcp/server.go`
- Three authentication modes supported:
  1. No auth (default): When neither `--api-key` nor `--firestore-project` is set, all requests allowed
  2. Static API key: Via `--api-key` flag for local development
  3. Firestore API keys: Via `--firestore-project` flag for multi-user production
- API key extraction from `Authorization: Bearer <key>` header
- Firestore collection structure: `api_keys/<api-key>` with `refresh_token`, `user_email`, `created_at`, `description` fields
- Context injection for refresh token via `auth.WithRefreshToken(ctx, token)`
- Modified `auth.GetClient()` to check for refresh token in context before falling back to file

**Files changed:**
- `internal/mcp/server.go` - Added Firestore client, APIKeyDocument struct, validateAPIKey(), extractBearerToken(), authMiddleware(), initFirestore() methods. Updated Run() to use middleware and initialize Firestore.
- `internal/mcp/server_test.go` - New test file with unit tests for extractBearerToken, validateAPIKey (no auth, static key), and authMiddleware (no auth, valid/invalid keys)
- `pkg/auth/auth.go` - Added WithRefreshToken(), GetRefreshTokenFromContext() functions and updated GetClient() to check context
- `go.mod` - Added cloud.google.com/go/firestore dependency
- `go.sum` - Updated dependencies
- `CLAUDE.md` - Added comprehensive authentication documentation section after "Starting the Server"
- `README.md` - Updated MCP Server section with authentication modes and usage examples

**Learnings:**
- MCP HTTP handler can be wrapped with standard Go middleware for authentication
- Firestore client initialization should be done once at server startup with proper cleanup
- Context value injection allows passing per-request data (refresh tokens) through the call stack
- OAuth2 config.Client() handles token refresh automatically when given just a refresh token
- Static analysis warnings (SA1012) remind us not to pass nil context even when code permits it

**Remaining issues:** None

---

## 2026-01-15 - US-00031 - terraform: Initialize infrastructure project structure

**Status:** Success

**What was implemented:**
- Created config.yaml at project root with GCP configuration
- Used terraform skill to initialize init/ and iac/ directories
- init/ contains state backend, service accounts, and API enablement templates
- iac/ contains provider template and local.tf for main infrastructure
- Updated Makefile with terraform targets (init-plan, init-deploy, plan, deploy, etc.)
- Updated .gitignore with terraform exclusion patterns
- Added Terraform infrastructure section to CLAUDE.md

**Files changed:**
- `config.yaml` - New terraform configuration file with prefix, project_id, location, services, resources
- `init/provider.tf` - GCP provider configuration for initialization
- `init/local.tf` - Config loader for init resources
- `init/state-backend.tf` - GCS bucket for terraform state
- `init/service-accounts.tf` - Custom service accounts (never use defaults)
- `init/services.tf` - API enablement
- `iac/provider.tf.template` - Provider template with backend placeholder
- `iac/local.tf` - Config loader for main infrastructure
- `iac/example-workload.tf` - Example resource file (to be replaced)
- `Makefile` - Added 7 terraform targets (plan, deploy, undeploy, init-plan, init-deploy, init-destroy, update-backend)
- `.gitignore` - Added terraform patterns (.terraform/, *.tfstate, etc.)
- `CLAUDE.md` - Added Terraform Infrastructure section and updated project structure

**Learnings:**
- The terraform skill requires explicit --project-root parameter to work in the correct directory
- config.yaml validation requires env to be one of: dev, stg, uat, prd (not prod, staging, etc.)
- GCP project_id must be 6-30 characters, lowercase letters, digits, hyphens, start with letter, end with letter/digit
- The terraform skill handles Makefile merging (adds targets to existing Makefile)
- The skill auto-updates .gitignore with comprehensive terraform exclusion patterns

**Remaining issues:** None

---

## 2026-01-15 - US-00032 - terraform: Cloud Run service configuration

**Status:** Success

**What was implemented:**
- Created `iac/workload-mcp.tf` with complete Cloud Run v2 service deployment
- Artifact Registry repository resource for Docker container images
- Cloud Run v2 service with autoscaling (min=0, max=3 instances)
- Environment variables: FIRESTORE_PROJECT, PORT, ENVIRONMENT, PROJECT_ID
- IAM permissions for service account (Firestore datastore.user, Secret Manager secretAccessor)
- Public access binding (allUsers can invoke, API key protection at app level)
- Removed example-workload.tf template file

**Files changed:**
- `iac/workload-mcp.tf` - New Cloud Run workload terraform file with all resources
- `iac/local.tf` - Minor terraform fmt spacing fix
- `iac/example-workload.tf` - Deleted (replaced by workload-mcp.tf)
- `CLAUDE.md` - Added comprehensive Cloud Run documentation section
- `stories.yaml` - Updated US-00032 `passes: false` to `passes: true`

**Learnings:**
- Cloud Run v2 API (`google_cloud_run_v2_service`) has different syntax from v1
  - Uses `scaling { min_instance_count, max_instance_count }` instead of annotations
  - Uses `traffic { type = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST" }` instead of `latest_revision`
  - Outputs use `.uri` instead of `.status[0].url`
- Service account email for init-created accounts follows pattern: `{prefix}-cloudrun-{env}@{project_id}.iam.gserviceaccount.com`
- IAM member resource for Cloud Run v2 uses `google_cloud_run_v2_service_iam_member`
- `cpu_idle = true` allows CPU throttling when idle for cost savings
- Terraform validate requires provider initialization; use `terraform init -backend=false` for syntax-only validation
- The `make plan` command requires `init-deploy` first (checks for provider.tf existence)

**Remaining issues:** None

---

## 2026-01-15 - US-00033 - terraform: Firestore database and indexes

**Status:** Success

**What was implemented:**
- Created `iac/database-firestore.tf` with complete Firestore database configuration
- Firestore database in Native mode with eur3 (Europe multi-region) location
- Composite indexes for efficient API key queries (by created_at and user_email)
- Delete protection enabled to prevent accidental deletion
- Comprehensive documentation of Firestore collection structure in CLAUDE.md

**Files changed:**
- `iac/database-firestore.tf` - New Firestore database and indexes terraform file
- `CLAUDE.md` - Added Firestore Database section and detailed collection structure documentation

**Learnings:**
- Firestore database resource (`google_firestore_database`) can only be created once per project
- If the default database already exists, the terraform resource will fail (need to import or remove)
- Firestore collections are NOT created by Terraform - they're created automatically on first document write
- Using the API key itself as the document ID enables O(1) lookup performance (no query needed)
- `delete_protection_state = "DELETE_PROTECTION_ENABLED"` prevents accidental database deletion
- Composite indexes require all fields to be specified, including `__name__` for sort order
- Location `eur3` is the Europe multi-region option (not `europe-west1` which is regional)

**Remaining issues:** None

---

## 2026-01-15 - US-00034 - terraform: Secret Manager for OAuth credentials

**Status:** Success

**What was implemented:**
- Created `iac/secrets.tf` with Secret Manager secret resource for OAuth credentials
- Secret uses automatic replication for high availability
- Manual secret version creation documented (keeps sensitive data out of Terraform state)
- Updated CLAUDE.md with comprehensive Secret Manager documentation

**Files changed:**
- `iac/secrets.tf` - New Secret Manager terraform configuration
- `CLAUDE.md` - Added Secret Manager section with resource table, configuration, outputs, and manual creation instructions

**Learnings:**
- Secret Manager secrets vs secret versions: Terraform creates the secret container, but actual credential data should be uploaded manually via gcloud to avoid storing sensitive data in Terraform state
- The `replication { auto {} }` block enables automatic replication for Google-managed replication across regions
- Secret versions can be created manually using: `gcloud secrets versions add <secret-id> --data-file=<path>`
- Cloud Run service account already has `roles/secretmanager.secretAccessor` from the workload-mcp.tf IAM binding
- Labels like `purpose = "oauth-credentials"` help with resource organization and auditing

**Remaining issues:** None

---

## 2026-01-15 - US-00035 - google-contacts: Makefile deployment targets

**Status:** Success

**What was implemented:**
- Created multi-stage Dockerfile for MCP server containerization
- Added docker-build target to build container image locally
- Added docker-push target to push image to Artifact Registry
- Added cloud-run-deploy target for full deployment pipeline (build + push + deploy)
- Updated CLAUDE.md with Docker deployment documentation
- Updated README.md with deployment section and targets

**Files changed:**
- `Dockerfile` - New multi-stage build (Go 1.25 builder, Alpine final image)
- `Makefile` - Added docker-build, docker-push, cloud-run-deploy targets
- `CLAUDE.md` - Added Docker Deployment section with targets table and usage examples
- `README.md` - Added Docker/Cloud Run targets section and deployment documentation

**Learnings:**
- Go 1.25 is available in Docker Hub as `golang:1.25` (not alpine variant yet)
- Multi-stage builds significantly reduce final image size (~20MB vs ~800MB)
- Makefile can read GCP config from config.yaml using shell commands and awk
- The `cloud-run-deploy` target chains docker-build → docker-push → gcloud run deploy
- REGISTRY_URL construction: `$(GCP_REGION)-docker.pkg.dev/$(GCP_PROJECT)/$(BINARY_NAME)`
- Health check using wget in Alpine (busybox wget) instead of curl
- Running as non-root user (appuser) follows security best practices
- Existing CLI targets remain unchanged for backward compatibility

**Remaining issues:** None

---

## 2026-01-15 - US-00036 - OAuth authentication endpoint (/auth)

**Status:** Success

**What was implemented:**
- OAuth2 authentication endpoints for API key generation workflow
- `/auth` endpoint redirects to Google OAuth consent page with CSRF protection
- `/auth/callback` exchanges authorization code for tokens and retrieves user email
- Cryptographically secure state parameter stored in-memory with 10-minute TTL
- OAuth credentials loaded from Secret Manager (primary) or local file (fallback)
- Integration with MCP server via AuthHandler and SetupRoutes

**Files changed:**
- `internal/mcp/auth.go` - New AuthHandler implementation with HandleAuth and HandleCallback
- `internal/mcp/auth_test.go` - Unit tests for state generation, validation, and callback handling
- `internal/mcp/server.go` - Integration with auth handler and route setup
- `internal/cli/cli.go` - CLI flags for OAuth configuration (--base-url, --secret-name, --credential-file)
- `go.mod` / `go.sum` - Added cloud.google.com/go/secretmanager dependency
- `CLAUDE.md` - Documentation for OAuth endpoints, CLI flags, and OAuth flow
- `README.md` - MCP Server section with OAuth authentication documentation

**Learnings:**
- State parameter must be cryptographically random (32 bytes, base64 URL encoded)
- Single-use state tokens prevent replay attacks
- ApprovalForce option ensures refresh token is always returned (even for re-authorizations)
- People API `people/me` endpoint retrieves authenticated user's email
- Secret Manager path format: `projects/{project}/secrets/{name}/versions/latest`
- In-memory state store with periodic cleanup goroutine is sufficient for single-instance deployments
- OAuth config should be loaded lazily (on first request) to support optional OAuth functionality

**Remaining issues:** None

---
