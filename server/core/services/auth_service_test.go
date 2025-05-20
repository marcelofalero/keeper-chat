package services_test

import (
	"errors"
	"fmt"
	"keeper/server/core/ports"
	"keeper/server/core/services"
	"keeper/server/models"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const testJWTSecret = "test-secret-key"

// --- Mock UserRepository ---

type mockUserRepository struct {
	usersByID         map[int64]models.User
	usersByUsername   map[string]models.User
	forceCreateError  bool
	forceGetError     bool
	nextUserID        int64
	createUserCalled  bool
	getUserByIDCalled bool
	getUserByUsernameCalled bool
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		usersByID:       make(map[int64]models.User),
		usersByUsername: make(map[string]models.User),
		nextUserID:      1,
	}
}

func (m *mockUserRepository) CreateUser(user models.User) error {
	m.createUserCalled = true
	if m.forceCreateError {
		return errors.New("forced create user error")
	}
	if _, exists := m.usersByUsername[user.Username]; exists {
		return services.ErrUserAlreadyExists // Or the specific error your service expects
	}
	user.ID = m.nextUserID
	m.nextUserID++
	m.usersByID[user.ID] = user
	m.usersByUsername[user.Username] = user
	return nil
}

func (m *mockUserRepository) GetUserByUsername(username string) (*models.User, error) {
	m.getUserByUsernameCalled = true
	if m.forceGetError {
		return nil, errors.New("forced get user error")
	}
	user, exists := m.usersByUsername[username]
	if !exists {
		return nil, nil // Standard behavior for "not found" that service layer should handle
	}
	return &user, nil
}

func (m *mockUserRepository) GetUserByID(id int64) (*models.User, error) {
	m.getUserByIDCalled = true
	if m.forceGetError {
		return nil, errors.New("forced get user by id error")
	}
	user, exists := m.usersByID[id]
	if !exists {
		return nil, nil // Standard behavior for "not found"
	}
	return &user, nil
}

// Reset mock state
func (m *mockUserRepository) reset() {
	m.usersByID = make(map[int64]models.User)
	m.usersByUsername = make(map[string]models.User)
	m.forceCreateError = false
	m.forceGetError = false
	m.nextUserID = 1
	m.createUserCalled = false
	m.getUserByIDCalled = false
	m.getUserByUsernameCalled = false
}

// --- AuthServiceImpl Tests ---

func TestRegister(t *testing.T) {
	mockRepo := newMockUserRepository()
	authService := services.NewAuthService(mockRepo, testJWTSecret)

	t.Run("Successful Registration", func(t *testing.T) {
		mockRepo.reset()
		username := "testuser"
		password := "password123"

		user, err := authService.Register(username, password)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if user == nil {
			t.Fatal("Expected user to be non-nil")
		}
		if user.Username != username {
			t.Errorf("Expected username %s, got %s", username, user.Username)
		}
		if user.PasswordHash != "" { // Service should clear hash on return
			t.Error("Expected PasswordHash to be empty on returned user model")
		}

		// Verify user in mock repo
		storedUser, exists := mockRepo.usersByUsername[username]
		if !exists {
			t.Fatal("User not found in mock repository after registration")
		}
		if storedUser.Username != username {
			t.Errorf("Stored username mismatch: expected %s, got %s", username, storedUser.Username)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(storedUser.PasswordHash), []byte(password)); err != nil {
			t.Errorf("Stored password hash does not match original password: %v", err)
		}
		if !mockRepo.createUserCalled {
			t.Error("CreateUser was not called on repository")
		}
	})

	t.Run("Registration with Existing Username", func(t *testing.T) {
		mockRepo.reset()
		username := "existinguser"
		password := "password123"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		mockRepo.usersByUsername[username] = models.User{ID: 1, Username: username, PasswordHash: string(hashedPassword)}

		_, err := authService.Register(username, "newpassword")
		if !errors.Is(err, services.ErrUserAlreadyExists) {
			t.Fatalf("Expected ErrUserAlreadyExists, got %v", err)
		}
	})

	t.Run("Registration with Repository CreateUser Error", func(t *testing.T) {
		mockRepo.reset()
		mockRepo.forceCreateError = true
		username := "erroruser"
		password := "password123"

		_, err := authService.Register(username, password)
		if err == nil {
			t.Fatal("Expected error due to repository failure, got nil")
		}
		// Check for a generic error message if specific error is not exposed
		expectedErrorMsg := "failed to save user"
		if err.Error() != expectedErrorMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
		}
	})

    t.Run("Registration Password Hashing Check", func(t *testing.T) {
		mockRepo.reset()
		username := "hashcheckuser"
		password := "securepassword"

		_, err := authService.Register(username, password)
		if err != nil {
			t.Fatalf("Registration failed: %v", err)
		}

		storedUser, exists := mockRepo.usersByUsername[username]
		if !exists {
			t.Fatalf("User %s not found in mock repo after registration", username)
		}
		if storedUser.PasswordHash == "" {
			t.Error("PasswordHash is empty in stored user")
		}
		if storedUser.PasswordHash == password {
			t.Error("PasswordHash is the same as plain text password")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(storedUser.PasswordHash), []byte(password)); err != nil {
			t.Errorf("bcrypt comparison failed for stored hash and original password: %v", err)
		}
	})
}


func TestLogin(t *testing.T) {
	mockRepo := newMockUserRepository()
	authService := services.NewAuthService(mockRepo, testJWTSecret)

	// Setup a user for login tests
	username := "loginuser"
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	
	t.Run("Successful Login", func(t *testing.T) {
		mockRepo.reset()
		mockRepo.usersByUsername[username] = models.User{ID: 1, Username: username, PasswordHash: string(hashedPassword)}
		mockRepo.usersByID[1] = mockRepo.usersByUsername[username]


		tokenString, err := authService.Login(username, password)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if tokenString == "" {
			t.Fatal("Expected token string, got empty")
		}
		if !mockRepo.getUserByUsernameCalled {
			t.Error("GetUserByUsername was not called")
		}
		// Further token validation can be done in TestValidateToken
	})

	t.Run("Login with Wrong Password", func(t *testing.T) {
		mockRepo.reset()
		mockRepo.usersByUsername[username] = models.User{ID: 1, Username: username, PasswordHash: string(hashedPassword)}
		mockRepo.usersByID[1] = mockRepo.usersByUsername[username]

		_, err := authService.Login(username, "wrongpassword")
		if !errors.Is(err, services.ErrInvalidCredentials) {
			t.Fatalf("Expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("Login for Non-existent User", func(t *testing.T) {
		mockRepo.reset() // Ensure user from previous subtest is cleared

		_, err := authService.Login("nonexistentuser", "password123")
		if !errors.Is(err, services.ErrInvalidCredentials) { // Service returns this for user not found too
			t.Fatalf("Expected ErrInvalidCredentials for non-existent user, got %v", err)
		}
	})

	t.Run("Login with Repository GetUserByUsername Error", func(t *testing.T) {
		mockRepo.reset()
		mockRepo.forceGetError = true

		_, err := authService.Login(username, password)
		if !errors.Is(err, services.ErrInvalidCredentials) { // Service translates underlying error
			t.Fatalf("Expected ErrInvalidCredentials due to repo error, got %v", err)
		}
	})
}

func TestValidateToken(t *testing.T) {
	mockRepo := newMockUserRepository()
	authService := services.NewAuthService(mockRepo, testJWTSecret)

	// Setup a user for token generation
	userID := int64(1)
	username := "validatetokenuser"
	password := "password123" // Not strictly needed for this test if we directly gen token
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	
	mockRepo.usersByUsername[username] = models.User{ID: userID, Username: username, PasswordHash: string(hashedPassword)}
	mockRepo.usersByID[userID] = mockRepo.usersByUsername[username]
	

	// Helper to generate a token for tests
	generateTestToken := func(claims services.JwtCustomClaims, secret []byte) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, _ := token.SignedString(secret)
		return signedToken
	}

	t.Run("Successful Token Validation", func(t *testing.T) {
		mockRepo.reset() // Reset for each sub-test if necessary
		mockRepo.usersByUsername[username] = models.User{ID: userID, Username: username, PasswordHash: string(hashedPassword)}
		mockRepo.usersByID[userID] = mockRepo.usersByUsername[username]

		// Generate a token using the service's Login method for realism
		tokenString, err := authService.Login(username, password)
		if err != nil {
			t.Fatalf("Login failed, cannot proceed with token validation: %v", err)
		}

		user, err := authService.ValidateToken(tokenString)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if user == nil {
			t.Fatal("Expected user, got nil")
		}
		if user.ID != userID {
			t.Errorf("Expected user ID %d, got %d", userID, user.ID)
		}
		if user.Username != username {
			t.Errorf("Expected username %s, got %s", username, user.Username)
		}
		if !mockRepo.getUserByIDCalled {
			t.Error("GetUserByID was not called")
		}
	})

	t.Run("Malformed Token", func(t *testing.T) {
		mockRepo.reset()
		_, err := authService.ValidateToken("this.is.not.a.valid.token")
		// Error might be generic "could not parse token" or more specific from JWT library
		if err == nil {
			t.Fatal("Expected error for malformed token, got nil")
		}
		// A more specific check could be: if !strings.Contains(err.Error(), "token is malformed") && !strings.Contains(err.Error(), "could not parse token")
		// For now, just checking for any error is fine. services.ErrInvalidToken is for valid structure but bad content.
		expectedErr := "could not parse token" // Based on current service impl.
        if err.Error() != expectedErr {
            t.Errorf("Expected error '%s' for malformed token, got '%v'", expectedErr, err)
        }
	})

	t.Run("Expired Token", func(t *testing.T) {
		mockRepo.reset()
		claims := services.JwtCustomClaims{
			Username: username,
			UserID:   userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired 1 hour ago
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}
		expiredToken := generateTestToken(claims, []byte(testJWTSecret))

		_, err := authService.ValidateToken(expiredToken)
		if !errors.Is(err, services.ErrInvalidToken) {
			t.Fatalf("Expected ErrInvalidToken for expired token, got %v", err)
		}
	})

	t.Run("Token Signed with Different Secret", func(t *testing.T) {
		mockRepo.reset()
		claims := services.JwtCustomClaims{
			Username: username,
			UserID:   userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		wrongSecretToken := generateTestToken(claims, []byte("different-secret"))

		_, err := authService.ValidateToken(wrongSecretToken)
		if !errors.Is(err, services.ErrInvalidToken) { // The service wraps this in ErrInvalidToken
			t.Fatalf("Expected ErrInvalidToken for token with wrong secret, got %v", err)
		}
	})

	t.Run("Valid Token but User Not Found in Repo", func(t *testing.T) {
		mockRepo.reset() // Important: User will NOT be in the repo for this test

		// Generate a token that would otherwise be valid
		claims := services.JwtCustomClaims{
			Username: "ghostuser", // User that won't be in mockRepo
			UserID:   999,         // ID that won't be in mockRepo
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		validTokenForGhostUser := generateTestToken(claims, []byte(testJWTSecret))
		
		_, err := authService.ValidateToken(validTokenForGhostUser)
		if !errors.Is(err, services.ErrUserNotFound) { // Service should check if user still exists
			t.Fatalf("Expected ErrUserNotFound for valid token but non-existent user, got %v", err)
		}
	})

	t.Run("Token with future NotBefore (NBF) claim", func(t *testing.T) {
		mockRepo.reset()
		claims := services.JwtCustomClaims{
			Username: username,
			UserID:   userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)), // Issued in the past
				NotBefore: jwt.NewNumericDate(time.Now().Add(time.Minute * 5)), // Not valid for another 5 minutes
			},
		}
		nbfToken := generateTestToken(claims, []byte(testJWTSecret))

		_, err := authService.ValidateToken(nbfToken)
		if !errors.Is(err, services.ErrInvalidToken) { // JWT library usually flags this as invalid
			t.Fatalf("Expected ErrInvalidToken for token with future NBF, got %v", err)
		}
	})

    t.Run("Token missing UserID claim", func(t *testing.T) {
        mockRepo.reset()
        // Custom claims struct but UserID is zero, which might be problematic if service expects it
        claims := struct {
            Username string `json:"username"`
            // UserID int64 `json:"user_id"` // UserID is missing
            jwt.RegisteredClaims
        }{
            Username: username,
            RegisteredClaims: jwt.RegisteredClaims{
                ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
            },
        }
        tokenMissingClaim := generateTestToken(claims, []byte(testJWTSecret))

        // The behavior might depend on how robust GetUserByID is, or if UserID is checked before call
        // The current service implementation would fetch user by ID 0 if UserID is not parsed,
        // which would likely result in ErrUserNotFound.
        _, err := authService.ValidateToken(tokenMissingClaim)
         if !errors.Is(err, services.ErrUserNotFound) && !errors.Is(err, services.ErrInvalidToken) {
            t.Fatalf("Expected ErrUserNotFound or ErrInvalidToken for token missing UserID, got %v", err)
        }
		// If UserID is essential and its absence is caught before DB lookup, ErrInvalidToken is more suitable.
		// If DB lookup for ID 0 happens and fails, ErrUserNotFound is fine.
    })

	t.Run("Token with invalid signing method", func(t *testing.T) {
		mockRepo.reset()
		// Manually create a token header with an unexpected algorithm
		header := fmt.Sprintf(`{"alg":"%s","typ":"JWT"}`, "none") // "none" alg or any other than HS256
		payload := fmt.Sprintf(`{"user_id":%d,"username":"%s","exp":%d}`, 
			userID, username, time.Now().Add(time.Hour).Unix())
		
		// Base64 encode header and payload
		encodedHeader := jwt.EncodeSegment([]byte(header))
		encodedPayload := jwt.EncodeSegment([]byte(payload))
		
		// Create token string (without signature part for "none" alg or with a dummy one for others)
		tokenString := fmt.Sprintf("%s.%s.", encodedHeader, encodedPayload) // For "none" alg
		// If testing HS256 with a bad signature, you'd add a dummy signature part.
		// For this test, the key is that the authService.ValidateToken function has a check:
		// if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok { return nil, errors.New("unexpected signing method") }

		_, err := authService.ValidateToken(tokenString)
		if err == nil {
			t.Fatal("Expected error for token with invalid signing method, got nil")
		}
		// The service wraps "unexpected signing method" into "could not parse token"
		expectedErrorMsg := "could not parse token" 
		if err.Error() != expectedErrorMsg {
			 t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
		}
	})
}

// Note: A more comprehensive test suite might also include tests for:
// - Race conditions if the service or repository had shared mutable state (not apparent here).
// - Behavior with very long usernames or passwords (edge cases for hashing or storage).
// - Specific error types returned by the repository being correctly handled or wrapped.
// - Different time zones for token expiration/issuance if that's relevant.
```
