package entities

import "github.com/pkg/errors"

var (
	_ Event = (*BaseEvent)(nil)
)

type BaseEvent struct {
	kind      MessageType
	payload   []byte
	recipient *DynamicUser
}

func NewBaseEvent(
	kind MessageType,
	payload []byte,
	recipient *DynamicUser,
) (*BaseEvent, error) {
	if kind == MessageType("") {
		return nil, errors.Wrap(ErrInvalidParam, "kind not set")
	}

	if len(payload) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "payload not set")
	}

	if recipient == nil {
		return nil, errors.Wrap(ErrInvalidParam, "recipient not set")
	}

	return &BaseEvent{
		kind:      kind,
		payload:   payload,
		recipient: recipient,
	}, nil
}

func (e *BaseEvent) Kind() MessageType {
	return e.kind
}

func (e *BaseEvent) Payload() []byte {
	return e.payload
}

func (e *BaseEvent) Recipient() *DynamicUser {
	return e.recipient
}
