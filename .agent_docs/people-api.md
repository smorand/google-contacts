# People API Reference

## Service Wrapper

The `internal/contacts/service.go` provides:

| Method | Description |
|--------|-------------|
| `GetPeopleService(ctx)` | Returns authenticated `*Service` wrapper |
| `TestConnection(ctx)` | Verifies API connectivity |
| `CreateContact(ctx, input)` | Creates a new contact |
| `SearchContacts(ctx, query)` | Searches by name, phone, email, company |
| `GetContact(ctx, resourceName)` | Retrieves basic contact info |
| `GetContactDetails(ctx, resourceName)` | Retrieves full contact details |
| `UpdateContact(ctx, resourceName, input)` | Updates existing contact |
| `DeleteContact(ctx, resourceName)` | Deletes a contact |

## Types

### ContactInput

```go
type ContactInput struct {
    FirstName string          // Required
    LastName  string          // Required - stored in UPPERCASE
    Phones    []PhoneEntry    // Required (at least one) - international format (+XX...)
    Emails    []EmailEntry    // Optional
    Addresses []AddressEntry  // Optional
    Company   string          // Optional
    Position  string          // Optional
    Notes     string          // Optional
    Birthday  string          // Optional (YYYY-MM-DD or --MM-DD)
}
```

**Data Rules:**
- **Last name**: Automatically converted to UPPERCASE (`Doe` → `DOE`)
- **Phone numbers**: Must be in international format starting with `+`

### Entry Types

```go
type PhoneEntry struct {
    Value string  // e.g., "+33612345678"
    Type  string  // mobile, work, home, main, other
}

type EmailEntry struct {
    Value string  // e.g., "john@acme.com"
    Type  string  // work, home, other
}

type AddressEntry struct {
    Value string  // e.g., "10 Rue Example, 75001 Paris"
    Type  string  // home, work, other
}
```

### UpdateInput

Uses pointers to distinguish "not provided" from "empty value":

```go
type UpdateInput struct {
    FirstName       *string
    LastName        *string
    Phone           *string         // Replaces first phone (backward compat)
    Phones          []PhoneEntry    // Replaces ALL phones
    AddPhones       []PhoneEntry    // Add without removing
    RemovePhones    []string        // Remove by value
    Email           *string         // Replaces first email
    Emails          []EmailEntry    // Replaces ALL emails
    AddEmails       []EmailEntry    // Add without removing
    RemoveEmails    []string        // Remove by value
    Addresses       []AddressEntry  // Replaces ALL addresses
    AddAddresses    []AddressEntry  // Add without removing
    RemoveAddresses []string        // Remove by street match
    Company         *string
    Position        *string
    Notes           *string
    Birthday        *string         // YYYY-MM-DD or --MM-DD
    ClearBirthday   bool            // Remove birthday
}
```

## CLI Formats

**Phone:** `type:number` or `number` (defaults to mobile)
- `+33612345678` → mobile
- `work:+33123456789` → work

**Email:** `type:email` or `email` (defaults to work)
- `john@acme.com` → work
- `home:john@gmail.com` → home

**Address:** `type:address` or `address` (defaults to home)
- `10 Rue Example, 75001 Paris` → home
- `work:50 Avenue Business, Lyon` → work

## Phone Normalization

Automatic normalization to international format:

| Input | Output |
|-------|--------|
| `0612345678` | `+33612345678` |
| `06 12 34 56 78` | `+33612345678` |
| `+33612345678` | `+33612345678` |
| `0033612345678` | `+33612345678` |

## Address Parsing

Automatic structured parsing:

**French format (auto-detected):**
- `10 Rue Example, 75001 Paris` → street, postal, city, country=France

**Generic format:**
- `123 Main St, New York, USA` → street, city, country

**Structured syntax:**
- `street=10 Rue Test;city=Paris;postal=75001;country=France`

## PersonFields

Common fields for API calls:

```go
PersonFields("names,phoneNumbers,emailAddresses,addresses,organizations,biographies,birthdays,metadata")
```

## API Patterns

**Creating:**
```go
person := &people.Person{
    Names: []*people.Name{{GivenName: "John", FamilyName: "Doe"}},
    PhoneNumbers: []*people.PhoneNumber{{Value: "+33612345678", Type: "mobile"}},
}
created, err := srv.People.CreateContact(person).PersonFields("names,phoneNumbers").Context(ctx).Do()
```

**Searching:**
```go
results, err := srv.SearchContacts(ctx, "John")
// Returns []SearchResult with ResourceName, DisplayName, Phone, Email, Company, Position, Notes
```

**Updating:**
```go
details, err := srv.UpdateContact(ctx, "c123456789", contacts.UpdateInput{
    AddPhones: []contacts.PhoneEntry{{Value: "+33123456789", Type: "work"}},
})
```

**Deleting:**
```go
err := srv.DeleteContact(ctx, "c123456789")
// Permanent deletion, no undo
```
