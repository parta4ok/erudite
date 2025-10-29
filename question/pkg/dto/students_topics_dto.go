package dto

// StudentsTopicsDTO represent map with student id as key and topic with id and title as value
// swagger:model StudentsTopicsDTO
type StudentsTopicsDTO struct {
	// required: true
	StudentsTopics map[string][]TopicWithIDDTO `json:"students_topics"`
}
