package dto

// TopicsDTO represents topics response
// swagger:model TopicsDTO
type TopicsDTO struct {
	Topics []string `json:"topics" example:"Базы данных,Go базовые типы"`
}

// TopicWithIDDTO represents topic with id and title
// swagger:model TopicWithIDDTO
type TopicWithIDDTO struct {
	// required: true
	ID string `json:"id"`
	// required: true
	Title string `json:"title"`
}
