package usersmanagement

// User represents a simplified user object mapped from Kratos Identity.
type User struct {
	ID        string                 `json:"id"`
	Email     string                 `json:"email"`
	FirstName string                 `json:"first_name,omitempty"`
	LastName  string                 `json:"last_name,omitempty"`
	Traits    map[string]interface{} `json:"traits"` // Raw traits from Kratos
}
