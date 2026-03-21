package nats

import "encoding/json"

// EventDTO represents the event structure from question service
type EventDTO struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

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

type ReportEventDTO struct {
	Kind      string  `json:"kind"`
	Format    string  `json:"format"`
	Payload   []byte  `json:"payload"`
	Recipient UserDTO `json:"recipient"`
}

type UserDTO struct {
	ID       string            `json:"id"`
	Name     string            `json:"Name"`
	Fullname string            `json:"fullname"`
	Contacts map[string]string `json:"contacts"`
	GroupID  string            `json:"group_id,omitempty"`
}

type DynamicRegistrationEventDTO struct {
	Kind      string         `json:"kind"`
	Format    string         `json:"format"`
	Payload   []byte         `json:"payload"`
	Recipient DynamicUserDTO `json:"recipient"`
}

type DynamicUserDTO struct {
	UserID   string            `json:"id"`
	Contacts map[string]string `json:"contacts"`
}
