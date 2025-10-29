package dto

// StudentsDTO represents students
// swagger:model Students
type StudentsIDsDTO struct {
	// list of students id
	// required: true
	Students []string `json:"students_ids"`
}
