package dto

// TopicsDTO represents topics response
// swagger:model TopicsDTO
type TopicsDTO struct {
	Topics []string `json:"topics" example:"Базы данных,Go базовые типы"`
}
