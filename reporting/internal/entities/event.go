package entities

type MessageType string
type Format string

const (
	NotificationAboutFinishedSession      = MessageType("notification about finished session")
	ReportAboutPassedTopicsByMentorGroups = MessageType("report about passed topics by mentor groups")
)

type User struct {
	ID       string
	Name     string
	Fullname string
	Contacts map[string]string
	GroupID  string
}

type LinkedMentorAndStudent struct {
	Mentor  *User
	Student *User
}

//go:generate mockgen -source=./event.go -destination=./testdata/event.go -package=testdata
type Event interface {
	Kind() MessageType
	Format() Format
	Payload() []byte
	Recipient() *User
}

func (m MessageType) String() string {
	return string(m)
}

func (f Format) String() string {
	return string(f)
}
