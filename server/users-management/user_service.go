package usersmanagement

import (
	"context"
	"fmt"
	"log"
	// kratos "github.com/ory/kratos-client-go" // Already imported in KratosClient
)

// UserService provides operations for user management via Kratos.
type UserService struct {
	kratosClient KratosClientAPI // Use the interface type
}

// NewUserService creates a new UserService.
func NewUserService(kratosClient KratosClientAPI) *UserService { // Accept the interface type
	if kratosClient == nil {
		log.Fatal("KratosClientAPI cannot be nil in NewUserService")
	}
	return &UserService{
		kratosClient: kratosClient,
	}
}

// GetUserByID retrieves a user's details from Kratos by their Kratos Identity ID.
func (s *UserService) GetUserByID(ctx context.Context, id string) (*User, error) {
	identity, resp, err := s.kratosClient.GetIdentity(ctx, id)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return nil, fmt.Errorf("failed to get user %s from Kratos (status %d): %w", id, statusCode, err)
	}

	if identity == nil {
		return nil, fmt.Errorf("no identity found for ID %s, though Kratos request was successful", id)
	}

	user := &User{
		ID: identity.Id,
	}
	if traitsMap, ok := identity.Traits.(map[string]interface{}); ok {
		user.Traits = traitsMap // Store all traits
		if email, ok := traitsMap["email"].(string); ok {
			user.Email = email
		}
		if nameMap, ok := traitsMap["name"].(map[string]interface{}); ok {
			if firstName, ok := nameMap["first"].(string); ok {
				user.FirstName = firstName
			}
			if lastName, ok := nameMap["last"].(string); ok {
				user.LastName = lastName
			}
		}
	} else {
		return nil, fmt.Errorf("kratos identity traits for user %s are not in the expected format: %T", id, identity.Traits)
	}

	return user, nil
}

// ValidateKratosSession validates a Kratos session token (cookie value).
// It returns a User model if the session is valid and active.
func (s *UserService) ValidateKratosSession(ctx context.Context, sessionTokenValue string) (*User, error) {
	session, resp, err := s.kratosClient.ToSession(ctx, sessionTokenValue)
	if err != nil {
		// TODO: Differentiate Kratos errors (e.g., 401 vs 500)
		// For now, wrap the error.
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		log.Printf("Kratos ToSession request failed (status %d): %v", statusCode, err)
		return nil, fmt.Errorf("session validation failed with Kratos: %w", err)
	}

	if session == nil || !session.GetActive() || session.Identity == nil {
		log.Printf("Kratos session is invalid, inactive, or identity is missing.")
		return nil, fmt.Errorf("invalid or inactive session")
	}

	// Map Kratos Identity from session to our User model
	identity := session.Identity
	user := &User{
		ID: identity.Id,
	}
	if traitsMap, ok := identity.Traits.(map[string]interface{}); ok {
		user.Traits = traitsMap
		if email, ok := traitsMap["email"].(string); ok {
			user.Email = email
		}
		if nameMap, ok := traitsMap["name"].(map[string]interface{}); ok {
			if firstName, ok := nameMap["first"].(string); ok {
				user.FirstName = firstName
			}
			if lastName, ok := nameMap["last"].(string); ok {
				user.LastName = lastName
			}
		}
	} else {
		return nil, fmt.Errorf("kratos identity traits from session for user %s are not in the expected format: %T", identity.Id, identity.Traits)
	}

	return user, nil
}
