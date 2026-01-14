// Package contacts provides unit tests for contacts service utilities.
package contacts

import "testing"

func TestExtractID(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		expected     string
	}{
		{
			name:         "full resource name",
			resourceName: "people/c123456789",
			expected:     "c123456789",
		},
		{
			name:         "already ID only",
			resourceName: "c123456789",
			expected:     "c123456789",
		},
		{
			name:         "empty string",
			resourceName: "",
			expected:     "",
		},
		{
			name:         "short prefix only",
			resourceName: "people",
			expected:     "people",
		},
		{
			name:         "people/ with empty id",
			resourceName: "people/",
			expected:     "people/", // len("people/") == 7, condition is > 7, so returns as-is
		},
		{
			name:         "people prefix without slash",
			resourceName: "peoplec123",
			expected:     "peoplec123",
		},
		{
			name:         "different person ID",
			resourceName: "people/c987654321",
			expected:     "c987654321",
		},
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

func TestContactInput_Validation(t *testing.T) {
	// Test ContactInput struct validation patterns

	tests := []struct {
		name    string
		input   ContactInput
		isValid bool
	}{
		{
			name: "valid input with all required fields",
			input: ContactInput{
				FirstName: "John",
				LastName:  "Doe",
				Phones:    []PhoneEntry{{Value: "+33612345678", Type: "mobile"}},
			},
			isValid: true,
		},
		{
			name: "valid input with multiple phones and emails",
			input: ContactInput{
				FirstName: "John",
				LastName:  "Doe",
				Phones: []PhoneEntry{
					{Value: "+33612345678", Type: "mobile"},
					{Value: "+33123456789", Type: "work"},
				},
				Emails: []EmailEntry{
					{Value: "john@example.com", Type: "work"},
					{Value: "john@gmail.com", Type: "home"},
				},
				Company:  "Acme Inc",
				Position: "CTO",
				Notes:    "Met at conference",
			},
			isValid: true,
		},
		{
			name: "missing first name",
			input: ContactInput{
				LastName: "Doe",
				Phones:   []PhoneEntry{{Value: "+33612345678", Type: "mobile"}},
			},
			isValid: false,
		},
		{
			name: "missing last name",
			input: ContactInput{
				FirstName: "John",
				Phones:    []PhoneEntry{{Value: "+33612345678", Type: "mobile"}},
			},
			isValid: false,
		},
		{
			name: "missing phone",
			input: ContactInput{
				FirstName: "John",
				LastName:  "Doe",
			},
			isValid: false,
		},
		{
			name: "empty phones slice",
			input: ContactInput{
				FirstName: "John",
				LastName:  "Doe",
				Phones:    []PhoneEntry{},
			},
			isValid: false,
		},
		{
			name:    "empty input",
			input:   ContactInput{},
			isValid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := validateContactInput(tc.input)
			if result != tc.isValid {
				t.Errorf("validateContactInput() = %v, want %v for input %+v", result, tc.isValid, tc.input)
			}
		})
	}
}

// validateContactInput validates that required fields are present.
// This function is for testing validation logic without API calls.
func validateContactInput(input ContactInput) bool {
	return input.FirstName != "" && input.LastName != "" && len(input.Phones) > 0
}

func TestSearchResult_Fields(t *testing.T) {
	// Test SearchResult struct field access
	result := SearchResult{
		ResourceName: "people/c123456789",
		DisplayName:  "John Doe",
		Phone:        "+33612345678",
		Email:        "john@example.com",
		Company:      "Acme Inc",
		Position:     "CTO",
		Notes:        "Some notes",
	}

	if result.ResourceName != "people/c123456789" {
		t.Errorf("ResourceName = %q, want %q", result.ResourceName, "people/c123456789")
	}
	if result.DisplayName != "John Doe" {
		t.Errorf("DisplayName = %q, want %q", result.DisplayName, "John Doe")
	}
	if result.Phone != "+33612345678" {
		t.Errorf("Phone = %q, want %q", result.Phone, "+33612345678")
	}
	if result.Email != "john@example.com" {
		t.Errorf("Email = %q, want %q", result.Email, "john@example.com")
	}
	if result.Company != "Acme Inc" {
		t.Errorf("Company = %q, want %q", result.Company, "Acme Inc")
	}
	if result.Position != "CTO" {
		t.Errorf("Position = %q, want %q", result.Position, "CTO")
	}
	if result.Notes != "Some notes" {
		t.Errorf("Notes = %q, want %q", result.Notes, "Some notes")
	}
}

func TestContactDetails_PhoneEntries(t *testing.T) {
	// Test ContactDetails with multiple phone entries
	details := ContactDetails{
		ResourceName: "people/c123456789",
		FirstName:    "John",
		LastName:     "Doe",
		DisplayName:  "John Doe",
		Phones: []PhoneEntry{
			{Value: "+33612345678", Type: "mobile"},
			{Value: "+33142001234", Type: "work"},
			{Value: "+33555123456", Type: "home"},
		},
		Emails: []EmailEntry{
			{Value: "john@example.com", Type: "work"},
			{Value: "john.doe@personal.com", Type: "home"},
		},
		Company:   "Acme Inc",
		Position:  "CTO",
		Notes:     "Important contact",
		UpdatedAt: "2026-01-14T10:30:00.123456Z",
	}

	// Test phone entries
	if len(details.Phones) != 3 {
		t.Errorf("len(Phones) = %d, want 3", len(details.Phones))
	}

	expectedPhones := []struct {
		value string
		typ   string
	}{
		{"+33612345678", "mobile"},
		{"+33142001234", "work"},
		{"+33555123456", "home"},
	}

	for i, expected := range expectedPhones {
		if details.Phones[i].Value != expected.value {
			t.Errorf("Phones[%d].Value = %q, want %q", i, details.Phones[i].Value, expected.value)
		}
		if details.Phones[i].Type != expected.typ {
			t.Errorf("Phones[%d].Type = %q, want %q", i, details.Phones[i].Type, expected.typ)
		}
	}

	// Test email entries
	if len(details.Emails) != 2 {
		t.Errorf("len(Emails) = %d, want 2", len(details.Emails))
	}

	expectedEmails := []struct {
		value string
		typ   string
	}{
		{"john@example.com", "work"},
		{"john.doe@personal.com", "home"},
	}

	for i, expected := range expectedEmails {
		if details.Emails[i].Value != expected.value {
			t.Errorf("Emails[%d].Value = %q, want %q", i, details.Emails[i].Value, expected.value)
		}
		if details.Emails[i].Type != expected.typ {
			t.Errorf("Emails[%d].Type = %q, want %q", i, details.Emails[i].Type, expected.typ)
		}
	}
}

func TestPhoneEntry_EmptyType(t *testing.T) {
	// Test default type handling for phone entries
	entry := PhoneEntry{
		Value: "+33612345678",
		Type:  "",
	}

	if entry.Type != "" {
		t.Errorf("PhoneEntry.Type should be empty when not set, got %q", entry.Type)
	}

	// Simulate the default logic used in GetContactDetails
	defaultType := entry.Type
	if defaultType == "" {
		defaultType = "other"
	}

	if defaultType != "other" {
		t.Errorf("Default type should be 'other', got %q", defaultType)
	}
}

func TestEmailEntry_EmptyType(t *testing.T) {
	// Test default type handling for email entries
	entry := EmailEntry{
		Value: "test@example.com",
		Type:  "",
	}

	if entry.Type != "" {
		t.Errorf("EmailEntry.Type should be empty when not set, got %q", entry.Type)
	}

	// Simulate the default logic used in GetContactDetails
	defaultType := entry.Type
	if defaultType == "" {
		defaultType = "other"
	}

	if defaultType != "other" {
		t.Errorf("Default type should be 'other', got %q", defaultType)
	}
}

func TestCreatedContact_Fields(t *testing.T) {
	// Test CreatedContact struct
	created := CreatedContact{
		ResourceName: "people/c123456789",
		DisplayName:  "John Doe",
	}

	if created.ResourceName != "people/c123456789" {
		t.Errorf("ResourceName = %q, want %q", created.ResourceName, "people/c123456789")
	}
	if created.DisplayName != "John Doe" {
		t.Errorf("DisplayName = %q, want %q", created.DisplayName, "John Doe")
	}
}

func TestResourceNameNormalization(t *testing.T) {
	// Test the resource name normalization logic used in GetContact and GetContactDetails
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "people/c123456789",
			expected: "people/c123456789",
		},
		{
			name:     "needs prefix",
			input:    "c123456789",
			expected: "people/c123456789",
		},
		{
			name:     "starts with p (people)",
			input:    "people/c987",
			expected: "people/c987",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeResourceName(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeResourceName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// normalizeResourceName adds "people/" prefix if missing.
// This mirrors the logic in GetContact and GetContactDetails.
func normalizeResourceName(resourceName string) string {
	if len(resourceName) > 0 && resourceName[0] != 'p' {
		return "people/" + resourceName
	}
	return resourceName
}

func TestParseAddress_Empty(t *testing.T) {
	result := ParseAddress("")
	if result != nil {
		t.Errorf("ParseAddress('') should return nil, got %+v", result)
	}
}

func TestParseAddress_FrenchFormat_PostalCity(t *testing.T) {
	// Format: "street, postal city" (e.g., "10 Rue Test, 75001 Paris")
	result := ParseAddress("10 Rue Test, 75001 Paris")
	if result == nil {
		t.Fatal("ParseAddress returned nil")
	}

	if result.StreetAddress != "10 Rue Test" {
		t.Errorf("StreetAddress = %q, want %q", result.StreetAddress, "10 Rue Test")
	}
	if result.PostalCode != "75001" {
		t.Errorf("PostalCode = %q, want %q", result.PostalCode, "75001")
	}
	if result.City != "Paris" {
		t.Errorf("City = %q, want %q", result.City, "Paris")
	}
	if result.Country != "France" {
		t.Errorf("Country = %q, want %q", result.Country, "France")
	}
}

func TestParseAddress_FrenchFormat_CityPostal(t *testing.T) {
	// Format: "street, city postal" (e.g., "10 Rue Test, Paris 75001")
	result := ParseAddress("10 Rue Test, Paris 75001")
	if result == nil {
		t.Fatal("ParseAddress returned nil")
	}

	if result.StreetAddress != "10 Rue Test" {
		t.Errorf("StreetAddress = %q, want %q", result.StreetAddress, "10 Rue Test")
	}
	if result.PostalCode != "75001" {
		t.Errorf("PostalCode = %q, want %q", result.PostalCode, "75001")
	}
	if result.City != "Paris" {
		t.Errorf("City = %q, want %q", result.City, "Paris")
	}
}

func TestParseAddress_FrenchFormat_WithCountry(t *testing.T) {
	// Format: "street, postal city, country"
	result := ParseAddress("10 Rue Test, 75001 Paris, France")
	if result == nil {
		t.Fatal("ParseAddress returned nil")
	}

	if result.StreetAddress != "10 Rue Test" {
		t.Errorf("StreetAddress = %q, want %q", result.StreetAddress, "10 Rue Test")
	}
	if result.PostalCode != "75001" {
		t.Errorf("PostalCode = %q, want %q", result.PostalCode, "75001")
	}
	if result.City != "Paris" {
		t.Errorf("City = %q, want %q", result.City, "Paris")
	}
	if result.Country != "France" {
		t.Errorf("Country = %q, want %q", result.Country, "France")
	}
}

func TestParseAddress_FrenchFormat_FourParts(t *testing.T) {
	// Format: "street, city, postal, country"
	result := ParseAddress("10 Rue Test, Paris, 75001, France")
	if result == nil {
		t.Fatal("ParseAddress returned nil")
	}

	if result.StreetAddress != "10 Rue Test" {
		t.Errorf("StreetAddress = %q, want %q", result.StreetAddress, "10 Rue Test")
	}
	if result.PostalCode != "75001" {
		t.Errorf("PostalCode = %q, want %q", result.PostalCode, "75001")
	}
	if result.City != "Paris" {
		t.Errorf("City = %q, want %q", result.City, "Paris")
	}
}

func TestParseAddress_StructuredSyntax(t *testing.T) {
	// Structured format: "key=value;key=value"
	result := ParseAddress("street=123 Rue Example;city=Paris;postal=75001;country=France")
	if result == nil {
		t.Fatal("ParseAddress returned nil")
	}

	if result.StreetAddress != "123 Rue Example" {
		t.Errorf("StreetAddress = %q, want %q", result.StreetAddress, "123 Rue Example")
	}
	if result.City != "Paris" {
		t.Errorf("City = %q, want %q", result.City, "Paris")
	}
	if result.PostalCode != "75001" {
		t.Errorf("PostalCode = %q, want %q", result.PostalCode, "75001")
	}
	if result.Country != "France" {
		t.Errorf("Country = %q, want %q", result.Country, "France")
	}

	// FormattedValue should be built from structured fields
	if result.FormattedValue == "" {
		t.Error("FormattedValue should not be empty")
	}
}

func TestParseAddress_StructuredSyntax_AllFields(t *testing.T) {
	// All structured fields including region
	result := ParseAddress("street=50 Avenue Business;city=Lyon;postal=69001;region=Rhone;country=France;countrycode=FR")
	if result == nil {
		t.Fatal("ParseAddress returned nil")
	}

	if result.StreetAddress != "50 Avenue Business" {
		t.Errorf("StreetAddress = %q, want %q", result.StreetAddress, "50 Avenue Business")
	}
	if result.City != "Lyon" {
		t.Errorf("City = %q, want %q", result.City, "Lyon")
	}
	if result.PostalCode != "69001" {
		t.Errorf("PostalCode = %q, want %q", result.PostalCode, "69001")
	}
	if result.Region != "Rhone" {
		t.Errorf("Region = %q, want %q", result.Region, "Rhone")
	}
	if result.Country != "France" {
		t.Errorf("Country = %q, want %q", result.Country, "France")
	}
	if result.CountryCode != "FR" {
		t.Errorf("CountryCode = %q, want %q", result.CountryCode, "FR")
	}
}

func TestParseAddress_GenericFormat_ThreeParts(t *testing.T) {
	// Generic format without French postal code: "street, city, country"
	result := ParseAddress("123 Main Street, New York, USA")
	if result == nil {
		t.Fatal("ParseAddress returned nil")
	}

	if result.StreetAddress != "123 Main Street" {
		t.Errorf("StreetAddress = %q, want %q", result.StreetAddress, "123 Main Street")
	}
	if result.City != "New York" {
		t.Errorf("City = %q, want %q", result.City, "New York")
	}
	if result.Country != "USA" {
		t.Errorf("Country = %q, want %q", result.Country, "USA")
	}
}

func TestParseAddress_GenericFormat_FourParts(t *testing.T) {
	// Generic format: "street, city, postal, country"
	// Note: 10001 looks like a French postal code (5 digits), so it will be parsed as French
	// Use a non-French postal code format for generic test
	result := ParseAddress("123 Main Street, London, SW1A 1AA, UK")
	if result == nil {
		t.Fatal("ParseAddress returned nil")
	}

	if result.StreetAddress != "123 Main Street" {
		t.Errorf("StreetAddress = %q, want %q", result.StreetAddress, "123 Main Street")
	}
	if result.City != "London" {
		t.Errorf("City = %q, want %q", result.City, "London")
	}
	if result.PostalCode != "SW1A 1AA" {
		t.Errorf("PostalCode = %q, want %q", result.PostalCode, "SW1A 1AA")
	}
	if result.Country != "UK" {
		t.Errorf("Country = %q, want %q", result.Country, "UK")
	}
}

func TestParseAddress_FormattedValuePreserved(t *testing.T) {
	// FormattedValue should preserve the original input
	input := "10 Rue Example, 75001 Paris, France"
	result := ParseAddress(input)
	if result == nil {
		t.Fatal("ParseAddress returned nil")
	}

	if result.FormattedValue != input {
		t.Errorf("FormattedValue = %q, want %q", result.FormattedValue, input)
	}
}

func TestParseAddress_StructuredBuildsFormattedValue(t *testing.T) {
	// Structured syntax should build FormattedValue from fields
	result := ParseAddress("street=10 Rue Test;city=Paris;postal=75001")
	if result == nil {
		t.Fatal("ParseAddress returned nil")
	}

	// FormattedValue should contain the key information
	if result.FormattedValue == "" {
		t.Error("FormattedValue should not be empty for structured input")
	}
	if result.StreetAddress == "" || result.City == "" || result.PostalCode == "" {
		t.Error("Structured fields should be extracted")
	}
}

func TestIsPostalCode(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"75001", true},
		{"10001", true},
		{"NYC", false},
		{"Paris", false},
		{"12345-6789", true}, // US ZIP+4
		{"", false},
	}

	for _, tc := range tests {
		result := isPostalCode(tc.input)
		if result != tc.expected {
			t.Errorf("isPostalCode(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}
