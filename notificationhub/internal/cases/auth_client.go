package cases

import "github.com/parta4ok/kvs/notificationhub/internal/entities"

//go:generate mockgen -source=auth_client.go -destination=testdata/auth_client.go -package=testdata
type AuthClient interface {
	GetRecipientByID(id string) (*entities.Recipient, error)
}
