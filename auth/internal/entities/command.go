package entities

//go:generate mockgen -source=command.go -destination=./testdata/command.go -package=testdata
type Command interface {
	Exec() (*CommandResult, error)
}
