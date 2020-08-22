package types

type Session struct {
	User User
}

func (s *Session) IsAnonymous() bool {
	return s.User.Username == ""
}
