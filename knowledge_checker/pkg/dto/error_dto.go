package dto

type ErrorDTO struct {
	StatusCode int    `json:"status_code"`
	ErrMsg     string `json:"error_message,omitempty"`
}
