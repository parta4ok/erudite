package entities

import (
	"strings"

	"github.com/pkg/errors"
)

type Recipient struct {
	ID       string
	Contacts map[string]string
}

func NewRecipient(id string, contacts map[string]string) (*Recipient, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.Wrap(ErrInvalidParam, "recipient id is empty")
	}

	if len(contacts) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "contacts is empty")
	}

	return &Recipient{
		ID:       strings.TrimSpace(id),
		Contacts: contacts,
	}, nil
}
