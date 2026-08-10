package dto

// swagger:model UserDTO
type UserDTO struct {
	ID       string            `json:"id"`
	Username string            `json:"name"`
	FullName string            `json:"fullname"`
	Rights   []string          `json:"rights"`
	Contacts map[string]string `json:"contacts,omitempty"`
	GroupID  string            `json:"group_id,omitempty"`
}
