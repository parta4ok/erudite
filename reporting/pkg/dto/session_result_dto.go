package dto

// PayloadDTO represents the payload structure from question service
type SessionResultDTO struct {
	UserID      string              `json:"user_id"`
	Topics      []string            `json:"topics"`
	Questions   map[string][]string `json:"questions"`
	UserAnswers map[string][]string `json:"user_answers"`
	IsExpire    bool                `json:"is_expire"`
	IsSuccess   bool                `json:"is_success"`
	Grade       string              `json:"grade"`
}

// EventDTO represents the event structure from question service
type EventDTO struct {
	EventType string      `json:"event_type"`
	Payload   interface{} `json:"payload"`
}
