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
