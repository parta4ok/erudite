package dto

// EventDTO represents all types event
type EventDTO struct {
	// required: true
	Kind string `json:"event_type"`
	// required: true
	Format string `json:"message_format"`
	// required: true
	Payload []byte `json:"message"`
	// required: true
	Recipient UserDTO `json:"recipient"`
}

// UserDTO represents user contacts data
type UserDTO struct {
	// required: true
	ID string `json:"user_id"`
	// required: true
	Contacts map[string]string `json:"contacts"`
}
