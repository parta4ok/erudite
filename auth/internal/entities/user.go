package entities

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Rights       []string
	Contacts     map[string]string
	LinkedID     string
}
