package dto

// QuestionDTO represents question
// swagger:model Question
type QuestionDTO struct {
	ID           uint64   `json:"question_id" example:"112441"`
	QuestionType string   `json:"question_type" example:"1"`
	Topic        string   `json:"topic" example:"Базы данных"`
	Subject      string   `json:"subject" example:"К какой категории языков относится SQL?"`
	Variants     []string `json:"variants" example:"Императивный,Декларативный,Смешной,Противный"`
}

// SessionDTO represents session
// swagger:model Session
type SessionDTO struct {
	SessionID uint64        `json:"session_id" example:"12312"`
	Topics    []string      `json:"topics" example:"Базы данных,Базовые типы в Go"`
	Questions []QuestionDTO `json:"questions"`
}

// SessionResultDTO represents session result
// swagger:model SessionResult
type SessionResultDTO struct {
	IsSuccess bool   `json:"is_success" example:"true"`
	Grade     string `json:"grade" example:"75.00 percents"`
}
