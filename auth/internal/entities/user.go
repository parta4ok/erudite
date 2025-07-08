package entities

type User struct {
	ID           uint64
	Username     string
	PasswordHash string
	Rights       []string
	Contacts     map[string]string
}
