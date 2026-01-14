// Package contacts provides the Google People API service wrapper.
package contacts

import (
	"context"
	"fmt"
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
	Phones    []PhoneEntry // Multiple phones with types
	Emails    []EmailEntry // Multiple emails with types
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

// UpdateInput contains the data for updating a contact.
// Only non-nil fields will be updated.
type UpdateInput struct {
	FirstName    *string
	LastName     *string
	Phone        *string      // Replaces first phone (backward compat)
	Phones       []PhoneEntry // Replaces all phones (new multi-phone)
	AddPhones    []PhoneEntry // Add phones without removing existing
	RemovePhones []string     // Remove phones by value
	Email        *string      // Replaces first email (backward compat)
	Emails       []EmailEntry // Replaces all emails (new multi-email)
	AddEmails    []EmailEntry // Add emails without removing existing
	RemoveEmails []string     // Remove emails by value
	Company      *string
	Position     *string
	Notes        *string
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
		PersonFields("names,phoneNumbers,emailAddresses,organizations,biographies,metadata").
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

	// If no fields to update, return current details
	if len(updateFields) == 0 {
		return s.GetContactDetails(ctx, resourceName)
	}

	// Perform the update
	updated, err := s.People.UpdateContact(resourceName, current).
		UpdatePersonFields(strings.Join(updateFields, ",")).
		PersonFields("names,phoneNumbers,emailAddresses,organizations,biographies,metadata").
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
