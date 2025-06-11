package services

import (
	"context"
	"errors"
	"log"
	// "time" // No longer generating JWTs here

	// "github.com/golang-jwt/jwt/v5" // No longer generating JWTs here
	// "golang.org/x/crypto/bcrypt" // No longer hashing passwords here
	// "keeper/server/core/ports" // Old port no longer directly used by this service for user repo
	usersmanagement "keeper/server/users-management" // New user service
	// "keeper/server/models" // Old user model no longer used here
)

// jwtExpiration defines the duration for which JWT tokens are valid.
// const jwtExpiration = time.Hour * 72 // Kratos manages session expiration

// ErrUserAlreadyExists is returned when trying to register a user that already exists.
var ErrUserAlreadyExists = errors.New("user already exists") // Kratos handles this, but error might be mapped.

// ErrUserNotFound is returned when a user is not found.
var ErrUserNotFound = errors.New("user not found") // Kratos might return 404, map to this.

// ErrInvalidCredentials is returned for login attempts with wrong username or password.
var ErrInvalidCredentials = errors.New("invalid username or password") // Kratos handles this.

// ErrInvalidToken is returned when a JWT token is invalid or expired.
var ErrInvalidToken = errors.New("invalid or expired session") // For Kratos sessions

// JwtCustomClaims defines the custom claims for JWT.
// type JwtCustomClaims struct { // No longer using custom JWTs from this service
// 	Username string `json:"username"`
// 	UserID   int64  `json:"user_id"`
// 	jwt.RegisteredClaims
// }

// AuthServiceImpl implements the ports.AuthService interface.
// The interface itself might need to be updated or re-evaluated later.
type AuthServiceImpl struct {
	userSvc *usersmanagement.UserService
	// jwtSecret []byte // No longer needed
}

// Verify AuthServiceImpl implements ports.AuthService (this might need adjustment)
// var _ ports.AuthService = (*AuthServiceImpl)(nil) // Comment out if ports.AuthService changes significantly

// NewAuthService creates a new instance of AuthServiceImpl.
// It now takes the new UserService for Kratos integration.
func NewAuthService(userSvc *usersmanagement.UserService /*, jwtSecret string - no longer needed */) *AuthServiceImpl {
	if userSvc == nil {
		log.Fatal("UserService cannot be nil in NewAuthService")
	}
	return &AuthServiceImpl{
		userSvc: userSvc,
		// jwtSecret: []byte(jwtSecret), // No longer needed
	}
}

// Register creates a new user account.
// THIS IS NOW HANDLED BY KRATOS. This method should either be removed
// or adapted to trigger a Kratos registration flow if applicable for this backend.
// For now, commenting out.
/*
func (s *AuthServiceImpl) Register(username, password string) (*models.User, error) {
	// ... old implementation ...
	// This logic is now managed by Ory Kratos.
	// The server might redirect to Kratos UI or use Kratos API for admin-driven registration.
	log.Println("AuthServiceImpl.Register called, but user registration is now handled by Ory Kratos.")
	return nil, errors.New("registration via this endpoint is deprecated; use Kratos flows")
}
*/

// Login authenticates a user and returns a JWT token.
// THIS IS NOW HANDLED BY KRATOS. This method should either be removed
// or adapted to trigger a Kratos login flow.
// For now, commenting out.
/*
func (s *AuthServiceImpl) Login(username, password string) (string, error) {
	// ... old implementation ...
	// This logic is now managed by Ory Kratos.
	// The server might redirect to Kratos UI. Kratos issues its own session cookie/token.
	log.Println("AuthServiceImpl.Login called, but user login is now handled by Ory Kratos.")
	return "", errors.New("login via this endpoint is deprecated; use Kratos flows")
}
*/

// ValidateToken checks the validity of a Kratos session token/cookie.
// It now returns the new usersmanagement.User model.
// The tokenString is expected to be the Kratos session cookie value (e.g., ory_kratos_session).
func (s *AuthServiceImpl) ValidateToken(ctx context.Context, sessionToken string) (*usersmanagement.User, error) {
	if sessionToken == "" {
		return nil, ErrInvalidToken // Or a more specific error like "missing session token"
	}

	user, err := s.userSvc.ValidateKratosSession(ctx, sessionToken)
	if err != nil {
		// Log the specific error from userSvc for server-side diagnostics
		log.Printf("Kratos session validation failed: %v", err)
		// Map to a generic error for the client
		// TODO: Check error type from userSvc to return more specific errors like ErrInvalidToken vs internal error
		return nil, ErrInvalidToken
	}
	return user, nil
}
