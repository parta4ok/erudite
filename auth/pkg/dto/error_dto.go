package dto

// ErrorDTO represents error response
// swagger:model ErrorDTO
type ErrorDTO struct {
	// HTTP status code
	// required: true
	// example: 404
	StatusCode int `json:"status_code"`

	// error message
	// required: false
	// example: user not found
	ErrMsg string `json:"error_message,omitempty"`
}
