package entities

type UserClaims struct {
	Username string
	Issuer   string
	Audience []string
	Subject  uint64
	Rights   []string
}
