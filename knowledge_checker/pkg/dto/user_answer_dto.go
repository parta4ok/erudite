package dto

type UserAnswerDTO struct {
	QuestionID uint64   `json:"question_id"`
	Answers    []string `json:"answers"`
}

type UserAnswersListDTO struct {
	AnswersList []UserAnswerDTO `json:"user_answer"`
}
