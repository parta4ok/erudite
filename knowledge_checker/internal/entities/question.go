package entities

const (
	SingleSelection QuestionType = iota + 1
	MultiSelection
)

type QuestionType int

//go:generate mockgen -source=question.go -destination=./testdata/question.go -package=testdata
type Question interface {
	ID() uint64
	Type() QuestionType
	Topic() string
	Payload() interface{}
	CorrectAnswer() map[string]struct{}
}
