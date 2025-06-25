package dto

type UserAnswerDTO struct {
	QuestionID uint64   `json:"question_id" example:"1234"`
	Answers    []string `json:"answers" example:"selection1,selection2"`
}

type UserAnswersListDTO struct {
	AnswersList []UserAnswerDTO `json:"user_answer"`
}
