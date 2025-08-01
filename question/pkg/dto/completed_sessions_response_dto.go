package dto

import "time"

// CompletedSessionResponseDTO represents completed session
// swagger:model CompletedSessionResponseDTO
type CompletedSessionResponseDTO struct {
	StartedAt     time.Time          `json:"started_at"`
	Topics        []string           `json:"topics"`
	UserAnswers   UserAnswersListDTO `json:"user_answers"`
	IsExpired     bool               `json:"is_expired"`
	SessionResult SessionResultDTO   `json:"session_result"`
}

// CompletedSessionsResponseListDTO represents list of completed sessions
// swagger:model CompletedSessionsResponseListDTO
type CompletedSessionsResponseListDTO struct {
	CompletedSessions []CompletedSessionResponseDTO `json:"completed_sessions"`
}
