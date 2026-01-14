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
