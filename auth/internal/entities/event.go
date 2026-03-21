package entities

type MessageType string
type Format string

const (
	NotificationAboutShortPassword = MessageType("report.short_password")
)

//go:generate mockgen -source=./event.go -destination=./testdata/event.go -package=testdata
type Event interface {
	Kind() MessageType
	Payload() []byte
	Recipient() *DynamicUser
}

func (m MessageType) String() string {
	return string(m)
}

func (f Format) String() string {
	return string(f)
}
