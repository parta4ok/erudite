package entities

import "github.com/pkg/errors"

var (
	_ Event = (*BaseEvent)(nil)
)

type BaseEvent struct {
	kind      MessageType
	format    Format
	payload   []byte
	recipient *User
}

func NewBaseEvent(
	kind MessageType,
	format Format,
	payload []byte,
	recipient *User,
) (*BaseEvent, error) {
	if kind == MessageType("") {
		return nil, errors.Wrap(ErrInvalidParam, "kind not set")
	}

	if format == Format("") {
		return nil, errors.Wrap(ErrInvalidParam, "format not set")
	}

	if len(payload) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "payload not set")
	}

	if recipient == nil {
		return nil, errors.Wrap(ErrInvalidParam, "recipient not set")
	}

	return &BaseEvent{
		kind:      kind,
		format:    format,
		payload:   payload,
		recipient: recipient,
	}, nil
}

func (e *BaseEvent) Kind() MessageType {
	return e.kind
}

func (e *BaseEvent) Format() Format {
	return e.format
}

func (e *BaseEvent) Payload() []byte {
	return e.payload
}

func (e *BaseEvent) Recipient() *User {
	return e.recipient
}
