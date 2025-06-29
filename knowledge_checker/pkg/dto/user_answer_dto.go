package dto

// UserAnswerDTO represents user answer
// swagger:model UserAnswerDTO
type UserAnswerDTO struct {
	QuestionID uint64   `json:"question_id" example:"1234"`
	Answers    []string `json:"answers" example:"selection1,selection2"`
}

// UserAnswersListDTO represents list of user answers
// swagger:model UserAnswersListDTO
type UserAnswersListDTO struct {
	AnswersList []UserAnswerDTO `json:"user_answers"`
}
