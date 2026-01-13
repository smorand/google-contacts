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
