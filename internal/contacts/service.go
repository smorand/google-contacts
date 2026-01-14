// Package contacts provides the Google People API service wrapper.
package contacts

import (
	"context"
	"fmt"

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
	Phone     string
	Email     string
	Company   string
	Position  string
	Notes     string
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

// ContactDetails contains full information for a single contact.
type ContactDetails struct {
	ResourceName string
	FirstName    string
	LastName     string
	DisplayName  string
	Phones       []PhoneEntry
	Emails       []EmailEntry
	Company      string
	Position     string
	Notes        string
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
		PhoneNumbers: []*people.PhoneNumber{
			{
				Value: input.Phone,
				Type:  "mobile",
			},
		},
	}

	// Add optional fields
	if input.Email != "" {
		person.EmailAddresses = []*people.EmailAddress{
			{
				Value: input.Email,
				Type:  "work",
			},
		}
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

	// Create the contact
	created, err := s.People.CreateContact(person).
		PersonFields("names,phoneNumbers,emailAddresses,organizations,biographies").
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

// GetContactDetails retrieves full details for a single contact by its resource name.
// The resourceName can be a full path (e.g., "people/c123") or just the ID (e.g., "c123").
// Returns all available fields including all phones, all emails with labels, and metadata.
func (s *Service) GetContactDetails(ctx context.Context, resourceName string) (*ContactDetails, error) {
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
