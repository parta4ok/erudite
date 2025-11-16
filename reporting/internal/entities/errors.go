package entities

import (
	"github.com/pkg/errors"
)

var (
	ErrInvalidParam = errors.New("invalid param")
	ErrInternal     = errors.New("internal error")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
)
