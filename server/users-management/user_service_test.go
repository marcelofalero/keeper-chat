package usersmanagement

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time" // For session timestamps

	kratos "github.com/ory/kratos-client-go"
	// "github.com/ory/kratos-client-go/ptr" // For Kratos pointers - package does not exist
)

// Helper functions for creating pointers to values
func boolPtr(b bool) *bool { return &b }
func timePtr(t time.Time) *time.Time { return &t }

// MockKratosClient is a mock implementation of the KratosClient methods needed for testing UserService.
type MockKratosClient struct {
	GetIdentityFunc func(ctx context.Context, id string) (*kratos.Identity, *http.Response, error)
	ToSessionFunc   func(ctx context.Context, sessionToken string) (*kratos.Session, *http.Response, error)
}

func (m *MockKratosClient) GetIdentity(ctx context.Context, id string) (*kratos.Identity, *http.Response, error) {
	if m.GetIdentityFunc != nil {
		return m.GetIdentityFunc(ctx, id)
	}
	return nil, nil, errors.New("GetIdentityFunc not implemented in mock")
}

func (m *MockKratosClient) ToSession(ctx context.Context, sessionToken string) (*kratos.Session, *http.Response, error) {
	if m.ToSessionFunc != nil {
		return m.ToSessionFunc(ctx, sessionToken)
	}
	return nil, nil, errors.New("ToSessionFunc not implemented in mock")
}

func TestUserService_GetUserByID_Success(t *testing.T) {
	mockIdentity := &kratos.Identity{
		Id: "test-id",
		SchemaId: "default",
		Traits: map[string]interface{}{
			"email": "test@example.com",
			"name": map[string]interface{}{
				"first": "Test",
				"last":  "User",
			},
		},
	}

	mockClient := &MockKratosClient{
		GetIdentityFunc: func(ctx context.Context, id string) (*kratos.Identity, *http.Response, error) {
			if id == "test-id" {
				return mockIdentity, &http.Response{StatusCode: http.StatusOK}, nil
			}
			return nil, &http.Response{StatusCode: http.StatusNotFound}, errors.New("not found")
		},
	}

	userService := NewUserService(mockClient)
	user, err := userService.GetUserByID(context.Background(), "test-id")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("Expected user, got nil")
	}
	if user.ID != "test-id" {
		t.Errorf("Expected user ID 'test-id', got '%s'", user.ID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected user email 'test@example.com', got '%s'", user.Email)
	}
	if user.FirstName != "Test" {
		t.Errorf("Expected user FirstName 'Test', got '%s'", user.FirstName)
	}
	if user.LastName != "User" {
		t.Errorf("Expected user LastName 'User', got '%s'", user.LastName)
	}
}

func TestUserService_GetUserByID_KratosClientError(t *testing.T) {
	mockClientError := errors.New("kratos client error")
	mockClient := &MockKratosClient{
		GetIdentityFunc: func(ctx context.Context, id string) (*kratos.Identity, *http.Response, error) {
			return nil, &http.Response{StatusCode: http.StatusInternalServerError}, mockClientError
		},
	}

	userService := NewUserService(mockClient)
	user, err := userService.GetUserByID(context.Background(), "test-id")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, mockClientError) { // Check if the underlying error is our mockClientError
		t.Errorf("Expected error to wrap '%v', got '%v'", mockClientError, err)
	}
	if user != nil {
		t.Errorf("Expected nil user, got %+v", user)
	}
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	mockClient := &MockKratosClient{
		GetIdentityFunc: func(ctx context.Context, id string) (*kratos.Identity, *http.Response, error) {
			return nil, &http.Response{StatusCode: http.StatusNotFound}, errors.New("identity not found") // Kratos client would return an error
		},
	}

	userService := NewUserService(mockClient)
	_, err := userService.GetUserByID(context.Background(), "unknown-id")

	if err == nil {
		t.Fatal("Expected error for non-existent user, got nil")
	}
	// We can check for a specific error message or type if UserService wraps it distinctly
}


func TestUserService_ValidateKratosSession_Success(t *testing.T) {
	mockSession := &kratos.Session{
		Id:     "session-id",
		Active: boolPtr(true),
		ExpiresAt: timePtr(time.Now().Add(1 * time.Hour)),
		AuthenticatedAt: timePtr(time.Now()),
		Identity: &kratos.Identity{
			Id: "user-from-session-id",
			SchemaId: "default",
			Traits: map[string]interface{}{
				"email": "sessionuser@example.com",
				"name": map[string]interface{}{
					"first": "Session",
					"last":  "User",
				},
			},
		},
	}

	mockClient := &MockKratosClient{
		ToSessionFunc: func(ctx context.Context, sessionToken string) (*kratos.Session, *http.Response, error) {
			if sessionToken == "valid-token" {
				return mockSession, &http.Response{StatusCode: http.StatusOK}, nil
			}
			return nil, &http.Response{StatusCode: http.StatusUnauthorized}, errors.New("invalid token")
		},
	}

	userService := NewUserService(mockClient)
	user, err := userService.ValidateKratosSession(context.Background(), "valid-token")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("Expected user, got nil")
	}
	if user.ID != "user-from-session-id" {
		t.Errorf("Expected user ID 'user-from-session-id', got '%s'", user.ID)
	}
	if user.Email != "sessionuser@example.com" {
		t.Errorf("Expected user email 'sessionuser@example.com', got '%s'", user.Email)
	}
}

func TestUserService_ValidateKratosSession_InactiveSession(t *testing.T) {
	mockSession := &kratos.Session{
		Id:     "session-id",
		Active: boolPtr(false), // Inactive session
		Identity: &kratos.Identity{Id: "some-user"}, // Identity might still be present
	}

	mockClient := &MockKratosClient{
		ToSessionFunc: func(ctx context.Context, sessionToken string) (*kratos.Session, *http.Response, error) {
			return mockSession, &http.Response{StatusCode: http.StatusOK}, nil // Kratos might return 200 but active:false
		},
	}

	userService := NewUserService(mockClient)
	user, err := userService.ValidateKratosSession(context.Background(), "any-token")

	if err == nil {
		t.Fatal("Expected error for inactive session, got nil")
	}
	if user != nil {
		t.Errorf("Expected nil user for inactive session, got %+v", user)
	}
	// Check for specific error message if desired, e.g., "invalid or inactive session"
	if err.Error() != "invalid or inactive session" {
		t.Errorf("Expected error message 'invalid or inactive session', got '%s'", err.Error())
	}
}

func TestUserService_ValidateKratosSession_KratosClientError(t *testing.T) {
	mockClientError := errors.New("kratos ToSession error")
	mockClient := &MockKratosClient{
		ToSessionFunc: func(ctx context.Context, sessionToken string) (*kratos.Session, *http.Response, error) {
			return nil, &http.Response{StatusCode: http.StatusInternalServerError}, mockClientError
		},
	}

	userService := NewUserService(mockClient)
	user, err := userService.ValidateKratosSession(context.Background(), "any-token")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, mockClientError) {
	    t.Errorf("Expected error to wrap '%v', got '%v'", mockClientError, err)
	}
	if user != nil {
		t.Errorf("Expected nil user, got %+v", user)
	}
}

func TestUserService_ValidateKratosSession_NoIdentityInSession(t *testing.T) {
	mockSession := &kratos.Session{
		Id:     "session-id",
		Active: boolPtr(true),
		Identity: nil, // No identity
	}

	mockClient := &MockKratosClient{
		ToSessionFunc: func(ctx context.Context, sessionToken string) (*kratos.Session, *http.Response, error) {
			return mockSession, &http.Response{StatusCode: http.StatusOK}, nil
		},
	}
	userService := NewUserService(mockClient)
	user, err := userService.ValidateKratosSession(context.Background(), "any-token")

	if err == nil {
		t.Fatal("Expected error when session has no identity, got nil")
	}
	if user != nil {
		t.Errorf("Expected nil user when session has no identity, got %+v", user)
	}
	if err.Error() != "invalid or inactive session" { // Current mapping
		t.Errorf("Expected error 'invalid or inactive session', got '%s'", err.Error())
	}
}
