package types

const (
	AdminUserGroup = "admin"
)

// AuthType describes how a session was authenticated.
type AuthType string

const (
	// AuthTypeNone is an unauthenticated (anonymous) session.
	AuthTypeNone AuthType = ""
	// AuthTypeToken is a session authenticated by a login token.
	AuthTypeToken AuthType = "token"
	// AuthTypeSignature is a session authenticated by a valid signature (access
	// key). A valid signature already proves authorization for the path.
	AuthTypeSignature AuthType = "signature"
	// AuthTypeBasic is a session authenticated by HTTP Basic credentials (used
	// by WebDAV).
	AuthTypeBasic AuthType = "basic"
)

// Principal is the request-scoped, authenticated context of the caller. Unlike
// the persisted Session, it is rebuilt for every request from the token (user),
// the signature, and request headers (path password), and is never stored.
type Principal struct {
	User User
	// AuthType records how this principal was authenticated.
	AuthType AuthType
	// PathPassword is the path password provided for the current request (from
	// the request header). It is request-scoped and never persisted.
	PathPassword string
}

func (p *Principal) IsAnonymous() bool {
	return p.User.Username == ""
}

func (p *Principal) HasUserGroup(group string) bool {
	for _, r := range p.User.Groups {
		if r.Name == group {
			return true
		}
	}
	return false
}

type Token struct {
	Token string    `json:"token"`
	Value Principal `json:"-"`
	// ExpiredAt is unix timestamp
	ExpiredAt int64 `json:"expiresAt"`
}

type TokenStore interface {
	// Create a token that store value
	Create(value Principal) (Token, error)
	// Validate a token and return the value
	Validate(token string) (Token, error)
	// Revoke a token, return value is not nil only when an error occurred
	Revoke(token string) error
}
