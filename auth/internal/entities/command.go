package entities

import "context"

//go:generate mockgen -source=command.go -destination=./testdata/command.go -package=testdata
type Command interface {
	Exec(ctx context.Context) (*CommandResult, error)
}
