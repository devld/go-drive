package types

const (
	AdminUserGroup = "admin"
)

type Session struct {
	User User
}

func (s *Session) IsAnonymous() bool {
	return s.User.Username == ""
}

func (s *Session) HasUserGroup(group string) bool {
	for _, r := range s.User.Groups {
		if r.Name == group {
			return true
		}
	}
	return false
}

type Token struct {
	Token string  `json:"token"`
	Value Session `json:"-"`
	// ExpiredAt is unix timestamp
	ExpiredAt int64 `json:"expiresAt"`
}

type TokenStore interface {
	// Create a token that store value
	Create(value Session) (Token, error)
	// Update an existing token value
	Update(token string, value Session) (Token, error)
	// Validate a token and return the value
	Validate(token string) (Token, error)
	// Revoke a token, return value is not nil only when an error occurred
	Revoke(token string) error
}
