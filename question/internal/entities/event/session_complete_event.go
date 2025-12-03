package event

import (
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/pkg/errors"
)

var (
	SessionCompleteEventType = EventType("SessionResultEvent")
)

var (
	_ Event = (*SessionCompleteEvent)(nil)
)

type SessionCompleteEvent struct {
	num     int
	payload []byte
}

func NewSessionCompleteEvent(payload []byte) (*SessionCompleteEvent, error) {
	if len(payload) == 0 {
		return nil, errors.Wrap(entities.ErrInvalidParam, "payload not set")
	}

	return &SessionCompleteEvent{
		payload: payload,
	}, nil
}

func (event *SessionCompleteEvent) Type() EventType {
	return SessionCompleteEventType
}

func (event *SessionCompleteEvent) Payload() []byte {
	return event.payload
}

func (event *SessionCompleteEvent) SetNum(num int) {
	event.num = num
}

func (event *SessionCompleteEvent) Num() int {
	return event.num
}
