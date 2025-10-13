package dto

type StudentDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Fullname string `json:"fullname"`
}

type GroupDTO struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Students []StudentDTO `json:"students"`
}
