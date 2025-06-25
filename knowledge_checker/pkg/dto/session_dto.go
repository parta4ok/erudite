package dto

import (
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
)

type SessionDTO struct {
	SessionID uint64                       `json:"session_id"`
	Topics    []string                     `json:"topics"`
	Questions map[uint64]entities.Question `json:"questions"`
}

type SessionResultDTO struct {
	IsSuccess bool   `json:"is_success" example:"true"`
	Grade     string `json:"grade" example:"75.00 percents"`
}
