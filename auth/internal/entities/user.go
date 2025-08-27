package entities

import "github.com/pkg/errors"

type User struct {
	ID           string
	Username     string
	PasswordHash string
	FullName     string
	Rights       []string
	Contacts     map[string]string
	GroupID      string
}

func NewUser(username string,
	password string,
	fullName string,
	rights []string,
	contacts map[string]string,
	groupID string) (*User, error) {
	if username == "" || password == "" || fullName == "" {
		return nil, errors.Wrap(ErrInvalidParam,
			"some of required fields is empty: username, password, fullname")
	}

	if rights == nil {
		return nil, errors.Wrap(ErrInvalidParam, "user has not rights")
	}

	return &User{
		Username:     username,
		PasswordHash: password,
		FullName:     fullName,
		Rights:       rights,
		Contacts:     contacts,
		GroupID:      groupID,
	}, nil
}
