package entities

import "github.com/pkg/errors"

type DeliveryService string
type Contact string

const (
	Telegram = DeliveryService("telegram")
	Email    = DeliveryService("email")
)

type User struct {
	ID       string
	Contacts map[DeliveryService]Contact
}

func NewUser(id string, contact map[string]string) (*User, error) {
	if id == "" {
		return nil, errors.Wrap(ErrInvalidParam, "user id is empty")
	}
	if len(contact) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "empty contact map")
	}

	contactRes := make(map[DeliveryService]Contact, len(contact))
	for t, c := range contact {
		contactRes[DeliveryService(t)] = Contact(c)
	}

	return &User{
		ID:       id,
		Contacts: contactRes,
	}, nil
}

func (c Contact) String() string {
	return string(c)
}
