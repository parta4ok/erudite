package entities

type Command interface {
	Exec() (*CommandResult, error)
}
