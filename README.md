# Google Contacts Manager

A command-line tool for managing Google Contacts using Google People API v1.

## Features

- Create new contacts with name, phone, email, address, company, birthday, and notes
- Search contacts by name, email, phone, or company
- View detailed contact information including addresses and birthday
- Update existing contacts (modify only specified fields)
- Delete contacts with confirmation prompt
- Supports multiple phones, emails, and addresses with type labels
- Supports birthday with or without year (YYYY-MM-DD or --MM-DD)
- Shares credentials with email-manager for unified OAuth consent

## Prerequisites

- Go 1.25 or later
- Google Cloud project with People API enabled
- OAuth2 credentials (`~/.credentials/google_credentials.json`)

## Installation

```bash
# Clone the repository
git clone https://github.com/smorand/google-contacts.git
cd google-contacts

# Build
make build

# Install (optional)
make install
```

## Configuration

### Google Cloud Setup

1. Create a Google Cloud project at https://console.cloud.google.com/
2. Enable the People API
3. Create OAuth2 credentials (Desktop application)
4. Download credentials and save as `~/.credentials/google_credentials.json`

### Credential Sharing

This application shares credentials with [email-manager](https://github.com/smorand/email-manager):

- **Credentials file**: `~/.credentials/google_credentials.json`
- **Token file**: `~/.credentials/google_token.json`

Both applications use unified OAuth scopes, so you only need to authorize once for both Gmail and Contacts access.

### First-time Authentication

On first use, the application will:
1. Open your browser for Google OAuth consent
2. Request permissions for Gmail and Contacts APIs
3. Save the token to `~/.credentials/google_token.json`

If you've already authorized email-manager, the existing token will be used automatically (no re-authorization needed).

## Usage

```bash
# Show help
google-contacts --help

# Show version
google-contacts --version
```

### Create a Contact

Create a new contact with required and optional fields:

```bash
# Create with single phone (defaults to mobile)
google-contacts create -f John -l Doe -p +33612345678

# Create with typed phone
google-contacts create -f John -l Doe -p "work:+33123456789"

# Create with multiple phones
google-contacts create -f John -l Doe \
  -p "mobile:+33612345678" \
  -p "work:+33123456789"

# Create with single email (defaults to work)
google-contacts create -f John -l Doe -p +33612345678 -e john@acme.com

# Create with multiple emails
google-contacts create -f John -l Doe -p +33612345678 \
  -e "work:john@acme.com" \
  -e "home:john@gmail.com"

# Create with birthday (full date)
google-contacts create -f John -l Doe -p +33612345678 -b 1985-03-15

# Create with birthday (month/day only, when year is unknown)
google-contacts create -f John -l Doe -p +33612345678 -b "--03-15"

# Create with address
google-contacts create -f John -l Doe -p +33612345678 \
  -a "10 Rue Example, 75001 Paris, France"

# Create with typed address
google-contacts create -f John -l Doe -p +33612345678 \
  -a "work:50 Avenue Business, Lyon, 69001"

# Create with all fields
google-contacts create \
  --firstname John \
  --lastname Doe \
  --phone "mobile:+33612345678" \
  --phone "work:+33123456789" \
  --email "work:john@acme.com" \
  --email "home:john@gmail.com" \
  --address "home:10 Rue Example, Paris" \
  --address "work:50 Avenue Business, Lyon" \
  --company "Acme Inc" \
  --position "CTO" \
  --birthday 1985-03-15 \
  --notes "Met at conference"
```

**Flags:**
| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--firstname` | `-f` | Yes | First name |
| `--lastname` | `-l` | Yes | Last name |
| `--phone` | `-p` | Yes | Phone number (can be repeated) |
| `--email` | `-e` | No | Email address (can be repeated) |
| `--address` | `-a` | No | Postal address (can be repeated) |
| `--company` | `-c` | No | Company name |
| `--position` | `-r` | No | Role/position at company |
| `--birthday` | `-b` | No | Birthday (YYYY-MM-DD or --MM-DD) |
| `--notes` | `-n` | No | Notes about the contact |

**Phone format:**
- Simple: `+33612345678` (defaults to "mobile" type)
- With type: `mobile:+33612345678`, `work:+33123456789`
- Multiple: Use `-p` flag multiple times

**Phone types:** `mobile` (default), `work`, `home`, `main`, `other`

**Email format:**
- Simple: `john@acme.com` (defaults to "work" type)
- With type: `work:john@acme.com`, `home:john@gmail.com`
- Multiple: Use `-e` flag multiple times

**Email types:** `work` (default), `home`, `other`

**Address format:**
- Simple: `10 Rue Example, 75001 Paris` (defaults to "home" type)
- With type: `work:50 Avenue Business, Lyon`
- Multiple: Use `-a` flag multiple times

**Address types:** `home` (default), `work`, `other`

**Structured address parsing:**
Addresses are automatically parsed into structured fields (street, city, postal code, country) for better Google Contacts integration:
- French addresses (5-digit postal codes) are auto-detected: `10 Rue Test, 75001 Paris` → France assumed
- Generic format: `123 Main St, New York, 10001, USA`
- Structured syntax (advanced): `street=10 Rue Test;city=Paris;postal=75001;country=France`

**Birthday format:**
- Full date: `YYYY-MM-DD` (e.g., `1985-03-15`)
- Month/day only: `--MM-DD` (e.g., `--03-15` when year is unknown)

### Search Contacts

Search for contacts by name, phone, email, or company:

```bash
# Search by name
google-contacts search "John"

# Search by partial name
google-contacts search "Joh"

# Search by company
google-contacts search "Acme"

# Search by phone (partial match)
google-contacts search "0612"
```

**Output behavior:**
- **Single result**: Shows full contact details (name, phone, email, company, position, notes)
- **Multiple results**: Shows a summary table with ID, name, phone, company, and email
- **No results**: Displays a message indicating no matches found

**Example output (multiple results):**
```
Found 2 contacts:

ID               Name                  Phone            Company          Email
---------------  --------------------  ---------------  ---------------  -------------------------
c123456789       John Doe              +33612345678     Acme Inc         john@acme.com
c987654321       John Smith            +33698765432     Tech Corp        john@tech.com
```

### Show Contact Details

Display full information for a specific contact:

```bash
# Show by contact ID
google-contacts show c123456789

# Show by full resource name
google-contacts show people/c123456789
```

**Output includes:**
- Full name (first and last)
- All phone numbers with labels (mobile, work, home, etc.)
- All email addresses with labels
- All postal addresses with labels
- Company and position
- Birthday (formatted as "March 15, 1985" or "March 15" if no year)
- Notes
- Google Contact ID
- Last update time

**Example output:**
```
Contact Details
────────────────────────────────────────

  Name: John Doe
    First: John
    Last: Doe
  ID: c123456789

  Phones:
    • +33612345678 (mobile)
    • +33145678901 (work)
  Email: john@acme.com (work)

  Addresses:
    • 10 Rue Example, 75001 Paris (home)
    • 50 Avenue Business, Lyon, 69001 (work)

  Company: Acme Inc
  Position: CTO

  Birthday: March 15, 1985

  Notes:
    Met at conference 2025
    Follow up about partnership

  Updated: 2026-01-14 10:30:00
```

### Update a Contact

Update an existing contact. Only the specified fields will be modified:

```bash
# Update only first name
google-contacts update c123456789 --firstname "Jane"

# Update primary phone (backward compatible, replaces first phone)
google-contacts update c123456789 -p "+33698765432"

# Replace ALL phones with new ones
google-contacts update c123456789 \
  --phones "mobile:+33612345678" \
  --phones "work:+33123456789"

# Add a work phone without removing existing phones
google-contacts update c123456789 --add-phone "work:+33123456789"

# Remove a specific phone by value
google-contacts update c123456789 --remove-phone "+33612345678"

# Update primary email (backward compatible, replaces first email)
google-contacts update c123456789 -e "newemail@acme.com"

# Replace ALL emails with new ones
google-contacts update c123456789 \
  --emails "work:john@acme.com" \
  --emails "home:john@gmail.com"

# Add a personal email without removing existing
google-contacts update c123456789 --add-email "home:john@gmail.com"

# Remove a specific email by value
google-contacts update c123456789 --remove-email "old@acme.com"

# Replace ALL addresses with new ones
google-contacts update c123456789 \
  --addresses "home:10 Rue Example, Paris" \
  --addresses "work:50 Avenue Business, Lyon"

# Add a work address without removing existing
google-contacts update c123456789 --add-address "work:50 Avenue Business, Lyon"

# Remove an address by street content match
google-contacts update c123456789 --remove-address "Avenue Business"

# Update company information
google-contacts update c123456789 --company "New Corp" --position "CEO"

# Set birthday
google-contacts update c123456789 --birthday 1985-03-15

# Set birthday (month/day only)
google-contacts update c123456789 --birthday "--03-15"

# Remove birthday
google-contacts update c123456789 --clear-birthday
```

**Basic Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--firstname` | `-f` | Update first name |
| `--lastname` | `-l` | Update last name |
| `--company` | `-c` | Update company name |
| `--position` | `-r` | Update role/position |
| `--notes` | `-n` | Update notes |

**Phone Management Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--phone` | `-p` | Update primary phone (replaces first phone only) |
| `--phones` | | Replace ALL phones (can be repeated) |
| `--add-phone` | | Add phone without removing existing (can be repeated) |
| `--remove-phone` | | Remove specific phone by value (can be repeated) |

**Email Management Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--email` | `-e` | Update primary email (replaces first email only) |
| `--emails` | | Replace ALL emails (can be repeated) |
| `--add-email` | | Add email without removing existing (can be repeated) |
| `--remove-email` | | Remove specific email by value (can be repeated) |

**Address Management Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--addresses` | | Replace ALL addresses (can be repeated) |
| `--add-address` | | Add address without removing existing (can be repeated) |
| `--remove-address` | | Remove address by street content match (can be repeated) |

**Birthday Management Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--birthday` | `-b` | Set birthday (YYYY-MM-DD or --MM-DD) |
| `--clear-birthday` | | Remove birthday from contact |

**Behavior:**
- Only specified fields are updated; unspecified fields remain unchanged
- At least one field must be specified
- Displays before/after summary showing changes
- Phone and email operations can be combined (e.g., add one and remove another)

**Example output:**
```
Contact updated successfully!

  Name: Jane Smith
  ID: c123456789
  Phone: +33612345678 → +33698765432
  Email: john@acme.com
  Company: Acme Inc → New Corp
  Position: CTO → CEO
```

### Delete a Contact

Delete a contact by its ID:

```bash
# Delete with confirmation prompt
google-contacts delete c123456789

# Delete without confirmation (use with caution)
google-contacts delete c123456789 --force
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Skip confirmation prompt |

**Safety features:**
- By default, displays contact summary before deletion
- Prompts for confirmation (y/N)
- Use `--force` to skip confirmation (useful for scripts)

**Example output:**
```
Contact to delete:

  Name: John Doe
  ID: c123456789
  Phone: +33612345678
  Email: john@acme.com
  Company: Acme Inc

Are you sure you want to delete this contact? (y/N): y

✓ Contact 'John Doe' has been deleted.
```

## Claude Skill Integration

This project includes a Claude skill for natural language interaction with Google Contacts.

### Installation

After building, install the skill symlink:

```bash
mkdir -p ~/.claude/skills/google-contacts/scripts
ln -sf $(pwd)/bin/google-contacts-linux-amd64 ~/.claude/skills/google-contacts/scripts/google-contacts
```

The skill documentation is maintained in `~/.claude/skills/google-contacts/SKILL.md`.

### Usage with Claude

Once installed, Claude can use natural language to manage contacts:

- **Create contacts**: "Create a contact for John Doe, phone +33612345678, at Acme Corp"
- **From screenshots**: "Create contact from this screenshot: ~/Downloads/business_card.png"
- **Search**: "Find contacts at L'Oreal" or "What's John's phone number?"
- **View details**: "Show me the details of contact c123456789"

## Project Structure

```
google-contacts/
├── go.mod                    # Module at root
├── go.sum
├── Makefile                  # Build automation
├── README.md                 # This file
├── CLAUDE.md                 # AI development guide
├── cmd/
│   └── google-contacts/
│       └── main.go           # Entry point
├── internal/
│   ├── cli/
│   │   └── cli.go            # CLI commands
│   └── contacts/
│       └── service.go        # People API service wrapper
└── pkg/
    └── auth/
        └── auth.go           # OAuth2 authentication
```

## Development

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Run all checks
make check
```

### Testing

Unit tests cover CLI utilities and service type validation:

```bash
# Run all tests
make test

# Run tests directly with go test
go test ./...
```

Tests are located at:
- `internal/cli/cli_test.go` - CLI utility functions
- `internal/contacts/service_test.go` - Service types and validation

## License

MIT
