package entities

type CommandResult struct {
	Success bool
	Message string
	Payload interface{}
}
