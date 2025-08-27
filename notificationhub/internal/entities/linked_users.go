package entities

type User struct {
	ID       string
	Name     string
	Fullname string
	Rights   []string
	Contacts map[string]string
	GroupID  string
}

type LinkedUsers struct {
	Recipient *User
	Student   *User
}
