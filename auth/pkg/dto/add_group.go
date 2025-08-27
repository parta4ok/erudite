package dto

type AddGroupRequestDTO struct {
	// required: true
	Title string `json:"title"`
	// required: true
	LinkedID string `json:"linked_id"`
}

type AddGroupResponseDTO struct {
	GroupID string `json:"group_id"`
}
