package dto

type UpdateUserDTO struct {
	Username *string            `json:"name,omitempty"`
	Fullname *string            `json:"fullname,omitempty"`
	Password *string            `json:"password,omitempty"`
	Rights   *[]string          `json:"rights,omitempty"`
	Contacts *map[string]string `json:"contacts,omitempty"`
	GroupID  *string            `json:"group_id,omitempty"`
}

type UpdateUserResponseDTO struct {
	// required: true
	UserID string `json:"user_id"`
}
