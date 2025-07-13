package entities

var (
	_ Question = (*SingleSelectionQuestion)(nil)
)

type SingleSelectionQuestion struct {
	id            string
	topic         string
	subject       string
	variants      []string
	correctAnswer string
}

func NewSingleSelectionQuestion(id string, topic string, subject string, variants []string,
	correctAnswer string) *SingleSelectionQuestion {

	return &SingleSelectionQuestion{
		id:            id,
		topic:         topic,
		subject:       subject,
		variants:      variants,
		correctAnswer: correctAnswer,
	}
}

func (q *SingleSelectionQuestion) ID() string {
	return q.id
}

func (q *SingleSelectionQuestion) Type() QuestionType {
	return SingleSelection
}

func (q *SingleSelectionQuestion) Topic() string {
	return q.topic
}

func (q *SingleSelectionQuestion) Subject() string {
	return q.subject
}
func (q *SingleSelectionQuestion) Variants() []string {
	return q.variants
}

func (q *SingleSelectionQuestion) IsAnswerCorrect(ans *UserAnswer) bool {
	if len(ans.answer) == 0 || len(ans.answer) != 1 {
		return false
	}

	return q.correctAnswer == ans.answer[0]
}
