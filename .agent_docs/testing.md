# Testing Guide

## Test Structure

```
internal/
├── cli/
│   └── cli_test.go         # CLI utility tests
└── contacts/
    └── service_test.go     # Service type tests
```

## Running Tests

```bash
make test       # All tests with verbose output
go test ./...   # Alternative
```

## Test Patterns

Use table-driven tests with `if` + `t.Errorf`:

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

## What's Tested

### CLI Utilities (`internal/cli/cli_test.go`)

- `extractID()` - Resource name to ID extraction
- `truncate()` - String truncation for tables
- `formatTime()` - ISO 8601 timestamp formatting
- `parsePhones()` - Phone parsing with type:number format
- `parseEmails()` - Email parsing with type:email format
- Field validation for create command

### Service Types (`internal/contacts/service_test.go`)

- `extractID()` - Resource name parsing
- `ContactInput` validation with multiple phones/emails
- `SearchResult` struct access
- `ContactDetails` with entries
- Resource name normalization
- `ParseAddress()` - French and generic formats
- `isPostalCode()` - Postal code detection

## Guidelines

- No network calls in unit tests
- Test pure functions and validation logic
- Use table-driven tests for multiple cases
- Test edge cases (empty strings, boundaries)
