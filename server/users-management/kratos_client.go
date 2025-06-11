package usersmanagement

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	kratos "github.com/ory/kratos-client-go"
)

// KratosClient wraps the Ory Kratos SDK client.
type KratosClient struct {
	adminAPI    *kratos.APIClient
	frontendAPI *kratos.APIClient // For session validation (ToSession)
}

// KratosClientAPI defines the interface for Kratos client operations.
// This allows for mocking the KratosClient in tests.
type KratosClientAPI interface {
	GetIdentity(ctx context.Context, id string) (*kratos.Identity, *http.Response, error)
	ToSession(ctx context.Context, sessionCookieValue string) (*kratos.Session, *http.Response, error)
}

// NewKratosClient creates a new KratosClient.
// kratosAdminURL is the base URL of the Kratos admin API (e.g., "http://kratos:4434").
// kratosPublicURL is the base URL of the Kratos public/frontend API (e.g., "http://127.0.0.1:4433" or "http://kratos:4433")
func NewKratosClient(kratosAdminURL string, kratosPublicURL string) (*KratosClient, error) {
	if kratosAdminURL == "" {
		return nil, fmt.Errorf("Kratos admin URL cannot be empty")
	}
	if kratosPublicURL == "" {
		return nil, fmt.Errorf("Kratos public URL cannot be empty")
	}

	// Admin API Client
	adminConf := kratos.NewConfiguration()
	parsedAdminURL, err := url.Parse(kratosAdminURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Kratos admin URL: %w", err)
	}
	if parsedAdminURL.Scheme == "" {
		parsedAdminURL.Scheme = "http"
	}
	if parsedAdminURL.Host == "" {
		parsedAdminURL.Host = parsedAdminURL.Path
		parsedAdminURL.Path = ""
	}
	adminConf.Servers = kratos.ServerConfigurations{{URL: parsedAdminURL.String()}}
	adminAPIClient := kratos.NewAPIClient(adminConf)

	// Frontend API Client
	publicConf := kratos.NewConfiguration()
	parsedPublicURL, err := url.Parse(kratosPublicURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Kratos public URL: %w", err)
	}
	if parsedPublicURL.Scheme == "" {
		parsedPublicURL.Scheme = "http"
	}
	if parsedPublicURL.Host == "" {
		parsedPublicURL.Host = parsedPublicURL.Path
		parsedPublicURL.Path = ""
	}
	publicConf.Servers = kratos.ServerConfigurations{{URL: parsedPublicURL.String()}}
	frontendAPIClient := kratos.NewAPIClient(publicConf)

	return &KratosClient{
		adminAPI:    adminAPIClient,
		frontendAPI: frontendAPIClient,
	}, nil
}

// GetIdentity fetches an identity from Kratos by its ID using the Admin API.
func (c *KratosClient) GetIdentity(ctx context.Context, id string) (*kratos.Identity, *http.Response, error) {
	identity, resp, err := c.adminAPI.IdentityAPI.GetIdentity(ctx, id).Execute()
	if err != nil {
		return nil, resp, fmt.Errorf("failed to get identity %s from Kratos Admin API: %w", id, err)
	}
	return identity, resp, nil
}

// WhoAmI validates a Kratos session cookie and returns the session details.
// It uses the Kratos Frontend API's ToSession endpoint.
// The `cookie` parameter should be the value of the Kratos session cookie (e.g., "ory_kratos_session=VALUE").
// Or just the VALUE if the SDK/API handles adding the cookie name.
// The SDK's ToSession method typically expects the cookie to be in the HTTP request context
// or passed as an explicit parameter if the method signature allows.
// Let's assume for now we need to pass the cookie value directly.
// The kratos-client-go SDK's `FrontendApi.ToSession` method uses the request's cookies.
// So, this method signature might be better if it takes *http.Request
// or we find a way to pass the cookie value directly if the SDK supports it.
// For now, let's try a simpler approach. The SDK call might need to be more complex,
// involving constructing a request or using a specific option.
// The `ToSession` method in the Go SDK is:
// `func (a *FrontendApiService) ToSession(ctx context.Context) FrontendApiToSessionRequest`
// `Execute() (*Session, *http.Response, error)`
// It relies on cookies being in the request context or X-Session-Token.
// This KratosClient method might be better placed to accept a *http.Request
// and extract the cookie there, or the caller (AuthService) can extract cookie and pass it.
// Let's assume `sessionCookieValue` is the full "ory_kratos_session=..." string or just the value.
// The SDK's `ToSession` does not directly take a token string.
// It expects the session to be available via the request's cookies or `X-Session-Token` header.
// We will need to construct a request for the SDK call if we only have the token string.
// This is a common point of confusion with the Kratos SDK.

// ToSession validates a Kratos session using the session cookie value and returns the session.
// The `sessionCookieValue` is the value of the `ory_kratos_session` cookie.
func (c *KratosClient) ToSession(ctx context.Context, sessionCookieValue string) (*kratos.Session, *http.Response, error) {
	// The Go SDK's ToSession call is part of FrontendApi and it usually relies on the http.Request
	// to have the cookie. If we only have the cookie *value*, we might need to use X-Session-Token.
	// Let's try with X-Session-Token.

	// Create a new context for this specific API call, adding the X-Session-Token.
	// Note: This is how you can pass the session token as a header.
	// The SDK should pick it up.
	// The `ctx` passed to `ToSessionExecute` is used for cancellation, deadlines, etc.
	// The actual request headers are often configured on the API request object itself.

	apiToSessionRequest := c.frontendAPI.FrontendAPI.ToSession(ctx)
	// How to add X-Session-Token: The generated SDK clients often have a way to set headers
	// per request, or the underlying HTTP client needs to be configured.
	// Looking at kratos-client-go, `ToSessionExecute` eventually calls `prepareRequest`
	// which uses `r.header` to set headers. `r` is `FrontendApiToSessionRequest`.
	// `FrontendApiToSessionRequest` doesn't seem to have a direct `Header()` method.
	// Instead, the `XSessionToken` is a parameter to `ToSession`:
	// `req := client.FrontendApi.ToSession(context.Background()).XSessionToken(token)`

	session, resp, err := apiToSessionRequest.XSessionToken(sessionCookieValue).Execute()
	if err != nil {
		return nil, resp, fmt.Errorf("Kratos ToSession call failed: %w", err)
	}
	if !session.GetActive() {
		return nil, resp, fmt.Errorf("Kratos session is inactive")
	}
	return session, resp, nil
}
