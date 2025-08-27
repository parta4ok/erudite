package dto

type AddUserDTO struct {
	// required: true
	Username string `json:"name"`
	// required: true
	Password string `json:"password"`
	// required: true
	FullName string `json:"fullname"`
	// required: true
	Rights   []string          `json:"rights"`
	Contacts map[string]string `json:"contacts,omitempty"`
	GroupID  string            `json:"group_id,omitempty"`
}

type AddUserResponseDTO struct {
	// required: true
	UserID string `json:"user_id"`
}
