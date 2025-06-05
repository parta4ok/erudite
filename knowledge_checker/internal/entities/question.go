package entities

//go:generate mockgen -source=question.go -destination=./testdata/question.go -package=testdata
type Question interface {
	ID() uint64
	Type() string
	Topic() string
	Payload() []byte
	UserAnswer() []byte
	CorrectAnswer() []byte
}
