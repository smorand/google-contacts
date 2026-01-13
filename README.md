# Google Contacts Manager

A command-line tool for managing Google Contacts using Google People API v1.

## Features

- Create new contacts with name, phone, email, company, and notes
- Search contacts by name, email, phone, or company
- View detailed contact information
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
# Create with required fields only
google-contacts create -f John -l Doe -p +33612345678

# Create with all fields
google-contacts create \
  --firstname John \
  --lastname Doe \
  --phone +33612345678 \
  --company "Acme Inc" \
  --position "CTO" \
  --email john@acme.com \
  --notes "Met at conference"
```

**Flags:**
| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--firstname` | `-f` | Yes | First name |
| `--lastname` | `-l` | Yes | Last name |
| `--phone` | `-p` | Yes | Phone number |
| `--company` | `-c` | No | Company name |
| `--position` | `-r` | No | Role/position at company |
| `--email` | `-e` | No | Email address |
| `--notes` | `-n` | No | Notes about the contact |

### Search Contacts (Coming Soon)

```bash
google-contacts search "John"
```

### Show Contact Details (Coming Soon)

```bash
google-contacts show c123456789
```

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

## License

MIT
