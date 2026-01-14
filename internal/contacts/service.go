// Package contacts provides the Google People API service wrapper.
package contacts

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/api/option"
	people "google.golang.org/api/people/v1"

	"google-contacts/pkg/auth"
)

// Service wraps the Google People API service with helper methods.
type Service struct {
	*people.Service
}

// GetPeopleService returns an authenticated Google People API service.
// It uses the shared OAuth2 token from pkg/auth.
// If no token exists, it will trigger the OAuth2 browser flow.
func GetPeopleService(ctx context.Context) (*Service, error) {
	client, err := auth.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated client: %w", err)
	}

	srv, err := people.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create People API service: %w", err)
	}

	return &Service{Service: srv}, nil
}

// TestConnection verifies the People API connection by fetching the authenticated user's profile.
// Returns nil if the connection is successful, error otherwise.
func (s *Service) TestConnection(ctx context.Context) error {
	// Fetch a minimal profile to verify connectivity
	_, err := s.People.Get("people/me").
		PersonFields("names").
		Do()
	if err != nil {
		return fmt.Errorf("failed to connect to People API: %w", err)
	}
	return nil
}

// ContactInput contains the data for creating a new contact.
type ContactInput struct {
	FirstName string
	LastName  string
	Phones    []PhoneEntry   // Multiple phones with types
	Emails    []EmailEntry   // Multiple emails with types
	Addresses []AddressEntry // Multiple addresses with types
	Company   string
	Position  string
	Notes     string
	Birthday  string // Format: YYYY-MM-DD or --MM-DD (month/day only)
}

// CreatedContact contains the result of a contact creation.
type CreatedContact struct {
	ResourceName string
	DisplayName  string
}

// SearchResult contains the data for a contact search result.
type SearchResult struct {
	ResourceName string
	DisplayName  string
	Phone        string
	Email        string
	Company      string
	Position     string
	Notes        string
}

// PhoneEntry represents a phone number with its label.
type PhoneEntry struct {
	Value string
	Type  string // mobile, work, home, etc.
}

// EmailEntry represents an email address with its label.
type EmailEntry struct {
	Value string
	Type  string // work, home, etc.
}

// AddressEntry represents a postal address with its label.
type AddressEntry struct {
	Value string // Formatted address string
	Type  string // home, work, other
}

// StructuredAddress represents a parsed postal address with structured fields.
type StructuredAddress struct {
	FormattedValue string // Full address as a single string
	StreetAddress  string // Street name and number
	City           string // City name
	PostalCode     string // Postal/ZIP code
	Region         string // State/Province (optional)
	Country        string // Country name
	CountryCode    string // ISO 3166-1 alpha-2 country code (optional)
}

// ContactDetails contains full information for a single contact.
type ContactDetails struct {
	ResourceName string
	FirstName    string
	LastName     string
	DisplayName  string
	Phones       []PhoneEntry
	Emails       []EmailEntry
	Addresses    []AddressEntry
	Company      string
	Position     string
	Notes        string
	Birthday     string // Format: YYYY-MM-DD or --MM-DD (if year unknown)
	CreatedAt    string
	UpdatedAt    string
}

// extractID extracts the contact ID from a resource name (e.g., "people/c123" -> "c123")
func extractID(resourceName string) string {
	if len(resourceName) > 7 && resourceName[:7] == "people/" {
		return resourceName[7:]
	}
	return resourceName
}

// parseBirthday parses a birthday string into a People API Birthday struct.
// Accepts formats:
// - YYYY-MM-DD: Full date (e.g., "1985-03-15")
// - --MM-DD: Month and day only, year unknown (e.g., "--03-15")
// Returns nil if the input is empty or invalid.
func parseBirthday(birthday string) *people.Birthday {
	if birthday == "" {
		return nil
	}

	date := &people.Date{}

	if strings.HasPrefix(birthday, "--") {
		// Format: --MM-DD (month and day only)
		parts := strings.Split(birthday[2:], "-")
		if len(parts) != 2 {
			return nil
		}
		month := 0
		day := 0
		fmt.Sscanf(parts[0], "%d", &month)
		fmt.Sscanf(parts[1], "%d", &day)
		if month < 1 || month > 12 || day < 1 || day > 31 {
			return nil
		}
		date.Month = int64(month)
		date.Day = int64(day)
		// Year is 0 (unknown)
	} else {
		// Format: YYYY-MM-DD
		parts := strings.Split(birthday, "-")
		if len(parts) != 3 {
			return nil
		}
		year := 0
		month := 0
		day := 0
		fmt.Sscanf(parts[0], "%d", &year)
		fmt.Sscanf(parts[1], "%d", &month)
		fmt.Sscanf(parts[2], "%d", &day)
		if year < 1 || month < 1 || month > 12 || day < 1 || day > 31 {
			return nil
		}
		date.Year = int64(year)
		date.Month = int64(month)
		date.Day = int64(day)
	}

	return &people.Birthday{Date: date}
}

// formatBirthday formats a birthday from People API to a display string.
// Returns format: "YYYY-MM-DD" or "--MM-DD" (if year is 0/unknown)
func formatBirthday(birthday *people.Birthday) string {
	if birthday == nil || birthday.Date == nil {
		return ""
	}
	d := birthday.Date
	if d.Year == 0 {
		return fmt.Sprintf("--%02d-%02d", d.Month, d.Day)
	}
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// Regular expression to match French postal codes (5 digits)
var frenchPostalCodeRegex = regexp.MustCompile(`\b(\d{5})\b`)

// ParseAddress parses various address formats into a structured address.
// Supported formats:
// - Simple: "123 Rue Example, Paris, 75001, France"
// - French: "123 Rue Example, 75001 Paris" (auto-detects French postal code format)
// - French: "10 Rue Test, Paris 75001, France"
// - Structured: "street=123 Rue Example;city=Paris;postal=75001;country=France"
//
// Returns the structured address with both formatted value and structured fields.
func ParseAddress(address string) *StructuredAddress {
	if address == "" {
		return nil
	}

	// Check for structured syntax (semicolon-separated key=value pairs)
	if strings.Contains(address, ";") && strings.Contains(address, "=") {
		return parseStructuredAddress(address)
	}

	// Try to parse as a French address or general format
	return parseFormattedAddress(address)
}

// parseStructuredAddress parses "key=value;key=value" format.
func parseStructuredAddress(address string) *StructuredAddress {
	result := &StructuredAddress{}
	parts := strings.Split(address, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, "="); idx > 0 {
			key := strings.ToLower(strings.TrimSpace(part[:idx]))
			value := strings.TrimSpace(part[idx+1:])
			switch key {
			case "street", "streetaddress":
				result.StreetAddress = value
			case "city":
				result.City = value
			case "postal", "postalcode", "zip":
				result.PostalCode = value
			case "region", "state", "province":
				result.Region = value
			case "country":
				result.Country = value
			case "countrycode":
				result.CountryCode = value
			}
		}
	}

	// Build formatted value from structured fields
	result.FormattedValue = buildFormattedAddress(result)
	return result
}

// buildFormattedAddress creates a formatted address string from structured fields.
func buildFormattedAddress(addr *StructuredAddress) string {
	var parts []string
	if addr.StreetAddress != "" {
		parts = append(parts, addr.StreetAddress)
	}
	if addr.PostalCode != "" && addr.City != "" {
		parts = append(parts, fmt.Sprintf("%s %s", addr.PostalCode, addr.City))
	} else if addr.City != "" {
		parts = append(parts, addr.City)
	} else if addr.PostalCode != "" {
		parts = append(parts, addr.PostalCode)
	}
	if addr.Region != "" {
		parts = append(parts, addr.Region)
	}
	if addr.Country != "" {
		parts = append(parts, addr.Country)
	}
	return strings.Join(parts, ", ")
}

// parseFormattedAddress parses comma-separated address formats.
// Handles various patterns including French addresses.
func parseFormattedAddress(address string) *StructuredAddress {
	result := &StructuredAddress{
		FormattedValue: address,
	}

	// Split by comma
	parts := strings.Split(address, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) == 0 {
		return result
	}

	// Try to find French postal code (5 digits) and extract city
	postalMatch := frenchPostalCodeRegex.FindStringSubmatch(address)

	if len(postalMatch) > 0 {
		result.PostalCode = postalMatch[1]
		// Parse as French address
		parseFrenchAddress(result, parts)
	} else {
		// Parse as generic address
		parseGenericAddress(result, parts)
	}

	return result
}

// parseFrenchAddress extracts structured fields from French address formats.
// French patterns:
// - "street, postal city" (e.g., "10 Rue Test, 75001 Paris")
// - "street, city postal" (e.g., "10 Rue Test, Paris 75001")
// - "street, postal city, country" (e.g., "10 Rue Test, 75001 Paris, France")
// - "street, city, postal, country" (e.g., "10 Rue Test, Paris, 75001, France")
func parseFrenchAddress(result *StructuredAddress, parts []string) {
	if len(parts) == 0 {
		return
	}

	// First part is typically the street
	result.StreetAddress = parts[0]

	// Look for the part containing the postal code
	postalIdx := -1
	for i, part := range parts {
		if frenchPostalCodeRegex.MatchString(part) {
			postalIdx = i
			break
		}
	}

	if postalIdx == -1 {
		return
	}

	postalPart := parts[postalIdx]

	// Extract city from the postal code part
	// Pattern 1: "75001 Paris" - postal code before city
	// Pattern 2: "Paris 75001" - postal code after city
	postalLoc := frenchPostalCodeRegex.FindStringIndex(postalPart)
	if postalLoc != nil {
		before := strings.TrimSpace(postalPart[:postalLoc[0]])
		after := strings.TrimSpace(postalPart[postalLoc[1]:])

		if before != "" && after == "" {
			// "Paris 75001" pattern - city is before postal code
			result.City = before
		} else if after != "" && before == "" {
			// "75001 Paris" pattern - city is after postal code
			result.City = after
		} else if after != "" {
			// "75001 Paris" is more common in France
			result.City = after
		}
	}

	// If city is still empty, try to extract from parts
	if result.City == "" {
		// Check if postal code is standalone and city is in another part
		for i, part := range parts {
			if i != postalIdx && i != 0 { // Skip street and postal part
				// If this part is just 5 digits, it's just the postal code
				if part == result.PostalCode {
					continue
				}
				// This might be the city
				if !frenchPostalCodeRegex.MatchString(part) {
					// Check if it's not a country (common countries)
					lowerPart := strings.ToLower(part)
					if lowerPart != "france" && lowerPart != "fr" {
						result.City = part
					}
				}
			}
		}
	}

	// Look for country (typically last part)
	if len(parts) > postalIdx+1 {
		lastPart := strings.TrimSpace(parts[len(parts)-1])
		lowerLast := strings.ToLower(lastPart)
		if lowerLast == "france" || lowerLast == "fr" {
			result.Country = lastPart
			if lowerLast == "fr" {
				result.CountryCode = "FR"
			}
		}
	}

	// If no country but French postal code, assume France
	if result.Country == "" && result.PostalCode != "" {
		result.Country = "France"
		result.CountryCode = "FR"
	}
}

// parseGenericAddress extracts structured fields from generic address formats.
// Generic pattern: "street, city, postal, country"
func parseGenericAddress(result *StructuredAddress, parts []string) {
	switch len(parts) {
	case 1:
		// Just street or full address
		result.StreetAddress = parts[0]
	case 2:
		// street, city OR street, city postal
		result.StreetAddress = parts[0]
		result.City = parts[1]
	case 3:
		// street, city, postal OR street, city, country
		result.StreetAddress = parts[0]
		result.City = parts[1]
		// Check if third part looks like a postal code (digits) or country
		if isPostalCode(parts[2]) {
			result.PostalCode = parts[2]
		} else {
			result.Country = parts[2]
		}
	case 4:
		// street, city, postal, country
		result.StreetAddress = parts[0]
		result.City = parts[1]
		result.PostalCode = parts[2]
		result.Country = parts[3]
	default:
		// 5+ parts: street, city, region, postal, country
		result.StreetAddress = parts[0]
		result.City = parts[1]
		result.Region = parts[2]
		result.PostalCode = parts[3]
		if len(parts) > 4 {
			result.Country = parts[4]
		}
	}
}

// isPostalCode checks if a string looks like a postal code (contains digits).
func isPostalCode(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

// CreateContact creates a new contact in Google Contacts.
// Returns the created contact's resource name and display name.
func (s *Service) CreateContact(ctx context.Context, input ContactInput) (*CreatedContact, error) {
	person := &people.Person{
		Names: []*people.Name{
			{
				GivenName:  input.FirstName,
				FamilyName: input.LastName,
			},
		},
	}

	// Add phone numbers (required, at least one)
	for _, phone := range input.Phones {
		phoneType := phone.Type
		if phoneType == "" {
			phoneType = "mobile"
		}
		person.PhoneNumbers = append(person.PhoneNumbers, &people.PhoneNumber{
			Value: phone.Value,
			Type:  phoneType,
		})
	}

	// Add email addresses (optional, with types)
	for _, email := range input.Emails {
		emailType := email.Type
		if emailType == "" {
			emailType = "work"
		}
		person.EmailAddresses = append(person.EmailAddresses, &people.EmailAddress{
			Value: email.Value,
			Type:  emailType,
		})
	}

	if input.Company != "" || input.Position != "" {
		person.Organizations = []*people.Organization{
			{
				Name:  input.Company,
				Title: input.Position,
			},
		}
	}

	if input.Notes != "" {
		person.Biographies = []*people.Biography{
			{
				Value:       input.Notes,
				ContentType: "TEXT_PLAIN",
			},
		}
	}

	// Add birthday if provided
	if input.Birthday != "" {
		birthday := parseBirthday(input.Birthday)
		if birthday != nil {
			person.Birthdays = []*people.Birthday{birthday}
		}
	}

	// Add addresses (optional, with types) - using structured parsing
	for _, addr := range input.Addresses {
		addrType := addr.Type
		if addrType == "" {
			addrType = "home"
		}

		// Parse address to extract structured fields
		structured := ParseAddress(addr.Value)
		peopleAddr := &people.Address{
			Type: addrType,
		}

		if structured != nil {
			peopleAddr.FormattedValue = structured.FormattedValue
			peopleAddr.StreetAddress = structured.StreetAddress
			peopleAddr.City = structured.City
			peopleAddr.PostalCode = structured.PostalCode
			peopleAddr.Region = structured.Region
			peopleAddr.Country = structured.Country
			peopleAddr.CountryCode = structured.CountryCode
		} else {
			peopleAddr.FormattedValue = addr.Value
		}

		person.Addresses = append(person.Addresses, peopleAddr)
	}

	// Create the contact
	created, err := s.People.CreateContact(person).
		PersonFields("names,phoneNumbers,emailAddresses,organizations,biographies,birthdays,addresses").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	result := &CreatedContact{
		ResourceName: created.ResourceName,
	}

	// Extract display name from created contact
	if len(created.Names) > 0 {
		result.DisplayName = created.Names[0].DisplayName
	}

	return result, nil
}

// SearchContacts searches for contacts matching the given query.
// The query matches on names, emails, phone numbers, and organizations.
// Returns a list of matching contacts with their details.
func (s *Service) SearchContacts(ctx context.Context, query string) ([]SearchResult, error) {
	// Send warmup request first (with empty query) to update cache
	// This is recommended by Google's API documentation
	_, _ = s.People.SearchContacts().
		Query("").
		ReadMask("names").
		Context(ctx).
		Do()

	// Now send the actual search request
	resp, err := s.People.SearchContacts().
		Query(query).
		PageSize(30).
		ReadMask("names,phoneNumbers,emailAddresses,organizations,biographies").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to search contacts: %w", err)
	}

	var results []SearchResult
	for _, r := range resp.Results {
		if r.Person == nil {
			continue
		}
		p := r.Person

		result := SearchResult{
			ResourceName: p.ResourceName,
		}

		// Extract display name
		if len(p.Names) > 0 {
			result.DisplayName = p.Names[0].DisplayName
		}

		// Extract first phone number
		if len(p.PhoneNumbers) > 0 {
			result.Phone = p.PhoneNumbers[0].Value
		}

		// Extract first email
		if len(p.EmailAddresses) > 0 {
			result.Email = p.EmailAddresses[0].Value
		}

		// Extract company and position
		if len(p.Organizations) > 0 {
			result.Company = p.Organizations[0].Name
			result.Position = p.Organizations[0].Title
		}

		// Extract notes
		if len(p.Biographies) > 0 {
			result.Notes = p.Biographies[0].Value
		}

		results = append(results, result)
	}

	return results, nil
}

// GetContact retrieves a single contact by its resource name.
// The resourceName can be a full path (e.g., "people/c123") or just the ID (e.g., "c123").
func (s *Service) GetContact(ctx context.Context, resourceName string) (*SearchResult, error) {
	// Normalize resource name
	if len(resourceName) > 0 && resourceName[0] != 'p' {
		resourceName = "people/" + resourceName
	}

	p, err := s.People.Get(resourceName).
		PersonFields("names,phoneNumbers,emailAddresses,organizations,biographies,metadata").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	result := &SearchResult{
		ResourceName: p.ResourceName,
	}

	// Extract display name
	if len(p.Names) > 0 {
		result.DisplayName = p.Names[0].DisplayName
	}

	// Extract first phone number
	if len(p.PhoneNumbers) > 0 {
		result.Phone = p.PhoneNumbers[0].Value
	}

	// Extract first email
	if len(p.EmailAddresses) > 0 {
		result.Email = p.EmailAddresses[0].Value
	}

	// Extract company and position
	if len(p.Organizations) > 0 {
		result.Company = p.Organizations[0].Name
		result.Position = p.Organizations[0].Title
	}

	// Extract notes
	if len(p.Biographies) > 0 {
		result.Notes = p.Biographies[0].Value
	}

	return result, nil
}

// DeleteContact deletes a contact by its resource name.
// The resourceName can be a full path (e.g., "people/c123") or just the ID (e.g., "c123").
// Returns the contact details before deletion for confirmation display.
func (s *Service) DeleteContact(ctx context.Context, resourceName string) error {
	// Normalize resource name
	if len(resourceName) > 0 && resourceName[0] != 'p' {
		resourceName = "people/" + resourceName
	}

	_, err := s.People.DeleteContact(resourceName).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	return nil
}

// UpdateInput contains the data for updating a contact.
// Only non-nil fields will be updated.
type UpdateInput struct {
	FirstName       *string
	LastName        *string
	Phone           *string        // Replaces first phone (backward compat)
	Phones          []PhoneEntry   // Replaces all phones (new multi-phone)
	AddPhones       []PhoneEntry   // Add phones without removing existing
	RemovePhones    []string       // Remove phones by value
	Email           *string        // Replaces first email (backward compat)
	Emails          []EmailEntry   // Replaces all emails (new multi-email)
	AddEmails       []EmailEntry   // Add emails without removing existing
	RemoveEmails    []string       // Remove emails by value
	Addresses       []AddressEntry // Replaces all addresses
	AddAddresses    []AddressEntry // Add addresses without removing existing
	RemoveAddresses []string       // Remove addresses by street content match
	Company         *string
	Position        *string
	Notes           *string
	Birthday        *string // Format: YYYY-MM-DD or --MM-DD (month/day only)
	ClearBirthday   bool    // Set to true to remove birthday
}

// UpdateContact updates an existing contact with the provided fields.
// Only fields that are non-nil in UpdateInput will be modified.
// Returns the updated contact details.
func (s *Service) UpdateContact(ctx context.Context, resourceName string, input UpdateInput) (*ContactDetails, error) {
	// Normalize resource name
	if len(resourceName) > 0 && resourceName[0] != 'p' {
		resourceName = "people/" + resourceName
	}

	// First, fetch the current contact to get etag and merge changes
	current, err := s.People.Get(resourceName).
		PersonFields("names,phoneNumbers,emailAddresses,addresses,organizations,biographies,birthdays,metadata").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	// Build the update mask for only the fields we're updating
	var updateFields []string

	// Update names if provided
	if input.FirstName != nil || input.LastName != nil {
		if len(current.Names) == 0 {
			current.Names = []*people.Name{{}}
		}
		if input.FirstName != nil {
			current.Names[0].GivenName = *input.FirstName
		}
		if input.LastName != nil {
			current.Names[0].FamilyName = *input.LastName
		}
		updateFields = append(updateFields, "names")
	}

	// Handle phone updates (multiple options available)
	phoneUpdated := false

	// Option 1: --phone flag replaces first phone (backward compatibility)
	if input.Phone != nil {
		if len(current.PhoneNumbers) == 0 {
			current.PhoneNumbers = []*people.PhoneNumber{{Type: "mobile"}}
		}
		current.PhoneNumbers[0].Value = *input.Phone
		phoneUpdated = true
	}

	// Option 2: Phones slice replaces all phones
	if len(input.Phones) > 0 {
		current.PhoneNumbers = nil
		for _, phone := range input.Phones {
			phoneType := phone.Type
			if phoneType == "" {
				phoneType = "mobile"
			}
			current.PhoneNumbers = append(current.PhoneNumbers, &people.PhoneNumber{
				Value: phone.Value,
				Type:  phoneType,
			})
		}
		phoneUpdated = true
	}

	// Option 3: AddPhones adds without removing existing
	if len(input.AddPhones) > 0 {
		for _, phone := range input.AddPhones {
			phoneType := phone.Type
			if phoneType == "" {
				phoneType = "mobile"
			}
			current.PhoneNumbers = append(current.PhoneNumbers, &people.PhoneNumber{
				Value: phone.Value,
				Type:  phoneType,
			})
		}
		phoneUpdated = true
	}

	// Option 4: RemovePhones removes specific phones by value
	if len(input.RemovePhones) > 0 {
		var remaining []*people.PhoneNumber
		for _, phone := range current.PhoneNumbers {
			shouldRemove := false
			for _, removeValue := range input.RemovePhones {
				if phone.Value == removeValue {
					shouldRemove = true
					break
				}
			}
			if !shouldRemove {
				remaining = append(remaining, phone)
			}
		}
		current.PhoneNumbers = remaining
		phoneUpdated = true
	}

	if phoneUpdated {
		updateFields = append(updateFields, "phoneNumbers")
	}

	// Handle email updates (multiple options available)
	emailUpdated := false

	// Option 1: --email flag replaces first email (backward compatibility)
	if input.Email != nil {
		if len(current.EmailAddresses) == 0 {
			current.EmailAddresses = []*people.EmailAddress{{Type: "work"}}
		}
		current.EmailAddresses[0].Value = *input.Email
		emailUpdated = true
	}

	// Option 2: Emails slice replaces all emails
	if len(input.Emails) > 0 {
		current.EmailAddresses = nil
		for _, email := range input.Emails {
			emailType := email.Type
			if emailType == "" {
				emailType = "work"
			}
			current.EmailAddresses = append(current.EmailAddresses, &people.EmailAddress{
				Value: email.Value,
				Type:  emailType,
			})
		}
		emailUpdated = true
	}

	// Option 3: AddEmails adds without removing existing
	if len(input.AddEmails) > 0 {
		for _, email := range input.AddEmails {
			emailType := email.Type
			if emailType == "" {
				emailType = "work"
			}
			current.EmailAddresses = append(current.EmailAddresses, &people.EmailAddress{
				Value: email.Value,
				Type:  emailType,
			})
		}
		emailUpdated = true
	}

	// Option 4: RemoveEmails removes specific emails by value
	if len(input.RemoveEmails) > 0 {
		var remaining []*people.EmailAddress
		for _, email := range current.EmailAddresses {
			shouldRemove := false
			for _, removeValue := range input.RemoveEmails {
				if email.Value == removeValue {
					shouldRemove = true
					break
				}
			}
			if !shouldRemove {
				remaining = append(remaining, email)
			}
		}
		current.EmailAddresses = remaining
		emailUpdated = true
	}

	if emailUpdated {
		updateFields = append(updateFields, "emailAddresses")
	}

	// Handle address updates (multiple options available) - using structured parsing
	addressUpdated := false

	// Option 1: Addresses slice replaces all addresses
	if len(input.Addresses) > 0 {
		current.Addresses = nil
		for _, addr := range input.Addresses {
			addrType := addr.Type
			if addrType == "" {
				addrType = "home"
			}

			// Parse address to extract structured fields
			structured := ParseAddress(addr.Value)
			peopleAddr := &people.Address{
				Type: addrType,
			}

			if structured != nil {
				peopleAddr.FormattedValue = structured.FormattedValue
				peopleAddr.StreetAddress = structured.StreetAddress
				peopleAddr.City = structured.City
				peopleAddr.PostalCode = structured.PostalCode
				peopleAddr.Region = structured.Region
				peopleAddr.Country = structured.Country
				peopleAddr.CountryCode = structured.CountryCode
			} else {
				peopleAddr.FormattedValue = addr.Value
			}

			current.Addresses = append(current.Addresses, peopleAddr)
		}
		addressUpdated = true
	}

	// Option 2: AddAddresses adds without removing existing
	if len(input.AddAddresses) > 0 {
		for _, addr := range input.AddAddresses {
			addrType := addr.Type
			if addrType == "" {
				addrType = "home"
			}

			// Parse address to extract structured fields
			structured := ParseAddress(addr.Value)
			peopleAddr := &people.Address{
				Type: addrType,
			}

			if structured != nil {
				peopleAddr.FormattedValue = structured.FormattedValue
				peopleAddr.StreetAddress = structured.StreetAddress
				peopleAddr.City = structured.City
				peopleAddr.PostalCode = structured.PostalCode
				peopleAddr.Region = structured.Region
				peopleAddr.Country = structured.Country
				peopleAddr.CountryCode = structured.CountryCode
			} else {
				peopleAddr.FormattedValue = addr.Value
			}

			current.Addresses = append(current.Addresses, peopleAddr)
		}
		addressUpdated = true
	}

	// Option 3: RemoveAddresses removes addresses by matching street content
	if len(input.RemoveAddresses) > 0 {
		var remaining []*people.Address
		for _, addr := range current.Addresses {
			shouldRemove := false
			for _, removeValue := range input.RemoveAddresses {
				// Match by checking if the address contains the removal string
				if strings.Contains(addr.FormattedValue, removeValue) {
					shouldRemove = true
					break
				}
			}
			if !shouldRemove {
				remaining = append(remaining, addr)
			}
		}
		current.Addresses = remaining
		addressUpdated = true
	}

	if addressUpdated {
		updateFields = append(updateFields, "addresses")
	}

	// Update organization if provided
	if input.Company != nil || input.Position != nil {
		if len(current.Organizations) == 0 {
			current.Organizations = []*people.Organization{{}}
		}
		if input.Company != nil {
			current.Organizations[0].Name = *input.Company
		}
		if input.Position != nil {
			current.Organizations[0].Title = *input.Position
		}
		updateFields = append(updateFields, "organizations")
	}

	// Update notes if provided
	if input.Notes != nil {
		if len(current.Biographies) == 0 {
			current.Biographies = []*people.Biography{{ContentType: "TEXT_PLAIN"}}
		}
		current.Biographies[0].Value = *input.Notes
		updateFields = append(updateFields, "biographies")
	}

	// Update birthday if provided
	if input.ClearBirthday {
		// Clear birthday by setting to empty slice
		current.Birthdays = nil
		updateFields = append(updateFields, "birthdays")
	} else if input.Birthday != nil {
		birthday := parseBirthday(*input.Birthday)
		if birthday != nil {
			current.Birthdays = []*people.Birthday{birthday}
			updateFields = append(updateFields, "birthdays")
		}
	}

	// If no fields to update, return current details
	if len(updateFields) == 0 {
		return s.GetContactDetails(ctx, resourceName)
	}

	// Perform the update
	updated, err := s.People.UpdateContact(resourceName, current).
		UpdatePersonFields(strings.Join(updateFields, ",")).
		PersonFields("names,phoneNumbers,emailAddresses,addresses,organizations,biographies,birthdays,metadata").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}

	// Convert to ContactDetails
	details := &ContactDetails{
		ResourceName: updated.ResourceName,
	}

	// Extract names
	if len(updated.Names) > 0 {
		name := updated.Names[0]
		details.FirstName = name.GivenName
		details.LastName = name.FamilyName
		details.DisplayName = name.DisplayName
	}

	// Extract all phone numbers with labels
	for _, phone := range updated.PhoneNumbers {
		entry := PhoneEntry{
			Value: phone.Value,
			Type:  phone.Type,
		}
		if entry.Type == "" {
			entry.Type = "other"
		}
		details.Phones = append(details.Phones, entry)
	}

	// Extract all email addresses with labels
	for _, email := range updated.EmailAddresses {
		entry := EmailEntry{
			Value: email.Value,
			Type:  email.Type,
		}
		if entry.Type == "" {
			entry.Type = "other"
		}
		details.Emails = append(details.Emails, entry)
	}

	// Extract all addresses with labels
	for _, addr := range updated.Addresses {
		entry := AddressEntry{
			Value: addr.FormattedValue,
			Type:  addr.Type,
		}
		if entry.Type == "" {
			entry.Type = "other"
		}
		details.Addresses = append(details.Addresses, entry)
	}

	// Extract company and position
	if len(updated.Organizations) > 0 {
		org := updated.Organizations[0]
		details.Company = org.Name
		details.Position = org.Title
	}

	// Extract notes
	if len(updated.Biographies) > 0 {
		details.Notes = updated.Biographies[0].Value
	}

	// Extract birthday
	if len(updated.Birthdays) > 0 {
		details.Birthday = formatBirthday(updated.Birthdays[0])
	}

	// Extract metadata (creation/update times)
	if updated.Metadata != nil {
		for _, source := range updated.Metadata.Sources {
			if source.Type == "CONTACT" {
				if source.UpdateTime != "" {
					details.UpdatedAt = source.UpdateTime
				}
			}
		}
	}

	return details, nil
}

// GetContactDetails retrieves full details for a single contact by its resource name.
// The resourceName can be a full path (e.g., "people/c123") or just the ID (e.g., "c123").
// Returns all available fields including all phones, all emails with labels, and metadata.
func (s *Service) GetContactDetails(ctx context.Context, resourceName string) (*ContactDetails, error) {
	// Normalize resource name
	if len(resourceName) > 0 && resourceName[0] != 'p' {
		resourceName = "people/" + resourceName
	}

	p, err := s.People.Get(resourceName).
		PersonFields("names,phoneNumbers,emailAddresses,addresses,organizations,biographies,birthdays,metadata").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	details := &ContactDetails{
		ResourceName: p.ResourceName,
	}

	// Extract names
	if len(p.Names) > 0 {
		name := p.Names[0]
		details.FirstName = name.GivenName
		details.LastName = name.FamilyName
		details.DisplayName = name.DisplayName
	}

	// Extract all phone numbers with labels
	for _, phone := range p.PhoneNumbers {
		entry := PhoneEntry{
			Value: phone.Value,
			Type:  phone.Type,
		}
		if entry.Type == "" {
			entry.Type = "other"
		}
		details.Phones = append(details.Phones, entry)
	}

	// Extract all email addresses with labels
	for _, email := range p.EmailAddresses {
		entry := EmailEntry{
			Value: email.Value,
			Type:  email.Type,
		}
		if entry.Type == "" {
			entry.Type = "other"
		}
		details.Emails = append(details.Emails, entry)
	}

	// Extract all addresses with labels
	for _, addr := range p.Addresses {
		entry := AddressEntry{
			Value: addr.FormattedValue,
			Type:  addr.Type,
		}
		if entry.Type == "" {
			entry.Type = "other"
		}
		details.Addresses = append(details.Addresses, entry)
	}

	// Extract company and position
	if len(p.Organizations) > 0 {
		org := p.Organizations[0]
		details.Company = org.Name
		details.Position = org.Title
	}

	// Extract notes
	if len(p.Biographies) > 0 {
		details.Notes = p.Biographies[0].Value
	}

	// Extract birthday
	if len(p.Birthdays) > 0 {
		details.Birthday = formatBirthday(p.Birthdays[0])
	}

	// Extract metadata (creation/update times)
	if p.Metadata != nil {
		for _, source := range p.Metadata.Sources {
			if source.Type == "CONTACT" {
				if source.UpdateTime != "" {
					details.UpdatedAt = source.UpdateTime
				}
			}
		}
	}

	return details, nil
}
