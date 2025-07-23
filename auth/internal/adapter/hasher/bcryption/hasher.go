package bcryption

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

var (
	_ common.Hasher = (*Hasher)(nil)
)

type Hasher struct {
}

func NewHasher() (*Hasher, error) {
	h := &Hasher{}

	return h, nil
}

func (h *Hasher) Hash(_ context.Context, pass string) (string, error) {
	slog.Info("Hash started")

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "generate hash from pass failure: %v", err)
		slog.Error(err.Error())
		return "", err
	}

	return string(hash), nil

}

func (h *Hasher) IsHash(_ context.Context, password string, hash string) (bool, error) {
	slog.Info("IsHash started")

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		err := errors.Wrapf(entities.ErrInvalidPassword, "compare hash and password: %v", err)
		slog.Warn(err.Error())
		return false, err
	}

	return true, nil
}
