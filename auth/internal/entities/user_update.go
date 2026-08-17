package entities

type UserUpdate struct {
	ID           string
	Username     *string
	PasswordHash *string
	FullName     *string
	Rights       *[]string
	Contacts     *map[string]string
	GroupID      *string
}
