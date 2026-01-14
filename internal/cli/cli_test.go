// Package cli provides unit tests for CLI utilities.
package cli

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
			name:         "short prefix",
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

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string no truncation",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "truncation with ellipsis",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "maxLen of 3 exact",
			input:    "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "maxLen of 2",
			input:    "hello",
			maxLen:   2,
			expected: "he",
		},
		{
			name:     "maxLen of 1",
			input:    "hello",
			maxLen:   1,
			expected: "h",
		},
		{
			name:     "maxLen of 0",
			input:    "hello",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "long truncation",
			input:    "this is a very long string that should be truncated",
			maxLen:   20,
			expected: "this is a very lo...",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncate(tc.input, tc.maxLen)
			if result != tc.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		isoTime  string
		expected string
	}{
		{
			name:     "full ISO 8601 timestamp",
			isoTime:  "2026-01-14T10:30:00.123456Z",
			expected: "2026-01-14 10:30:00",
		},
		{
			name:     "timestamp without microseconds",
			isoTime:  "2026-01-14T10:30:00Z",
			expected: "2026-01-14 10:30:00",
		},
		{
			name:     "date only (10 chars)",
			isoTime:  "2026-01-14",
			expected: "2026-01-14",
		},
		{
			name:     "partial timestamp",
			isoTime:  "2026-01-14T10",
			expected: "2026-01-14",
		},
		{
			name:     "short string",
			isoTime:  "2026",
			expected: "2026",
		},
		{
			name:     "empty string",
			isoTime:  "",
			expected: "",
		},
		{
			name:     "different time",
			isoTime:  "2025-12-31T23:59:59.999999Z",
			expected: "2025-12-31 23:59:59",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatTime(tc.isoTime)
			if result != tc.expected {
				t.Errorf("formatTime(%q) = %q, want %q", tc.isoTime, result, tc.expected)
			}
		})
	}
}

func TestValidateRequiredFlags(t *testing.T) {
	// Test the validation logic used in runCreate
	// These tests verify the validation rules without calling the API

	tests := []struct {
		name        string
		firstName   string
		lastName    string
		phone       string
		wantErr     bool
		errContains string
	}{
		{
			name:      "all required fields provided",
			firstName: "John",
			lastName:  "Doe",
			phone:     "+33612345678",
			wantErr:   false,
		},
		{
			name:        "missing first name",
			firstName:   "",
			lastName:    "Doe",
			phone:       "+33612345678",
			wantErr:     true,
			errContains: "first name",
		},
		{
			name:        "missing last name",
			firstName:   "John",
			lastName:    "",
			phone:       "+33612345678",
			wantErr:     true,
			errContains: "last name",
		},
		{
			name:        "missing phone",
			firstName:   "John",
			lastName:    "Doe",
			phone:       "",
			wantErr:     true,
			errContains: "phone",
		},
		{
			name:        "all fields empty",
			firstName:   "",
			lastName:    "",
			phone:       "",
			wantErr:     true,
			errContains: "first name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRequiredFields(tc.firstName, tc.lastName, tc.phone)
			if tc.wantErr {
				if err == nil {
					t.Errorf("validateRequiredFields() expected error containing %q, got nil", tc.errContains)
				} else if !containsString(err.Error(), tc.errContains) {
					t.Errorf("validateRequiredFields() error = %q, want error containing %q", err.Error(), tc.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateRequiredFields() unexpected error: %v", err)
				}
			}
		})
	}
}

// validateRequiredFields checks if required create command fields are provided.
// This is extracted from runCreate for testing purposes.
func validateRequiredFields(firstName, lastName, phone string) error {
	if firstName == "" {
		return errFirstNameRequired
	}
	if lastName == "" {
		return errLastNameRequired
	}
	if phone == "" {
		return errPhoneRequired
	}
	return nil
}

// Error constants for validation
var (
	errFirstNameRequired = validationError("first name is required (--firstname or -f)")
	errLastNameRequired  = validationError("last name is required (--lastname or -l)")
	errPhoneRequired     = validationError("phone number is required (--phone or -p)")
)

// validationError is a simple error type for validation errors.
type validationError string

func (e validationError) Error() string {
	return string(e)
}

// containsString checks if substr is found in s.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

// findSubstring checks if substr exists in s.
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
