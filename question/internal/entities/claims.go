package entities

type Claims struct {
	Username string
	Issuer   string
	Subject  string
	Audience []string
	Rights   []string
}
