package entities

const (
	ReportFinishedSession = MessageType("report.session_result")
	ReportPassedTopics    = MessageType("report.passed_topics")
)

const (
	HTTPFormat = Format("http")
)

type MessageType string
type Format string

type Event interface {
	Kind() MessageType
	Format() Format
	Payload() []byte
	GetRecipient() *User
}

func (m MessageType) String() string {
	return string(m)
}
