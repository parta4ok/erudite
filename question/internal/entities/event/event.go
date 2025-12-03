package event

type EventType string

func (et EventType) String() string {
	return string(et)
}

type Event interface {
	SetNum(num int)
	Num() int
	Type() EventType
	Payload() []byte
}
