package dto

type StudentDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Fullname string `json:"fullname"`
}

type GroupDTO struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	LinkedID string       `json:"linked_id,omitempty"`
	Students []StudentDTO `json:"students"`
}
