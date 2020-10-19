package server

type Token struct {
	Token     string      `json:"token"`
	Value     interface{} `json:"-"`
	ExpiredAt int64       `json:"expires_at"`
}

type TokenStore interface {
	// Create a token that store value
	Create(value interface{}) (Token, error)
	// Update an existing token value
	Update(token string, value interface{}) (Token, error)
	// Validate a token and return the value
	Validate(token string) (interface{}, error)
	// Revoke a token, return value is not nil only when an error occurred
	Revoke(token string) error
}
