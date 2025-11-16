package entities

import (
	"strings"

	"github.com/pkg/errors"
)

var (
	_ Event = (*NormalEvent)(nil)
)

type NormalEvent struct {
	kind      MessageType
	format    Format
	payload   []byte
	recipient *User
}

func NewNormalEvent(
	kind string,
	format string,
	payload []byte,
	recipient *User,
) (*NormalEvent, error) {
	if !strings.HasPrefix(kind, ReportFinishedSession.String()) &&
		!strings.HasPrefix(kind, ReportPassedTopics.String()) {
		return nil, errors.Wrapf(ErrInvalidParam, "unknown event type: %s", kind)
	}

	if format == "" {
		return nil, errors.Wrap(ErrInvalidParam, "unknown message format")
	}

	if len(payload) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "empty message")
	}

	if recipient == nil {
		return nil, errors.Wrap(ErrInvalidParam, "recipient not set")
	}

	return &NormalEvent{
		kind:      MessageType(kind),
		format:    Format(format),
		payload:   payload,
		recipient: recipient,
	}, nil

}

func (e *NormalEvent) Kind() MessageType {
	return e.kind
}
func (e *NormalEvent) Format() Format {
	return e.format
}
func (e *NormalEvent) Payload() []byte {
	return e.payload
}
func (e *NormalEvent) GetRecipient() *User {
	return e.recipient
}
