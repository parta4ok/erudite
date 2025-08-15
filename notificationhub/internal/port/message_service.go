package port

import "github.com/parta4ok/kvs/notificationhub/internal/entities"

//go:generate mockgen -source=./message_service.go -destination=testdata/message_service.go -package=testdata
type MessageService interface {
	SendMessage(sessionResult *entities.SessionResult) error
}
