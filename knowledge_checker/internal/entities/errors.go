package entities

import (
	"github.com/pkg/errors"
)

var (
	ErrInvalidParam        = errors.New("invalid param")
	ErrUnprocessibleEntity = errors.New("unprocessible entity")
	ErrInvalidState        = errors.New("invalid state")
	ErrInternal            = errors.New("internal error")
	ErrForbidden           = errors.New("forbidden")
)
