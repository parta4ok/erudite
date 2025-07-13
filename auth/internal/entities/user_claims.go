package entities

type UserClaims struct {
	Username string
	Issuer   string
	Audience []string
	Subject  string
	Rights   []string
}
