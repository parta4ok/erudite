package accessor

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
)

type KeyName string

const (
	UserClaims KeyName = "UserClaims"
)

var (
	ErrAssertion = errors.New("assertion error")
)

type RightAccessor struct{}

func NewRightAccessor() (*RightAccessor, error) {
	return &RightAccessor{}, nil
}

func (accessor *RightAccessor) HasPermission(ctx context.Context, rights []string) (bool, error) {
	slog.Info("HasPermission started")

	claimsRaw := ctx.Value(UserClaims)
	claims, ok := claimsRaw.(*Claims)
	if !ok {
		err := errors.Wrap(ErrAssertion, "assert data from context to claims failure")
		slog.Error(err.Error())
		return false, err
	}

	userRightsMap := make(map[string]struct{}, len(claims.Rights))
	for _, right := range claims.Rights {
		userRightsMap[right] = struct{}{}
	}

	for _, right := range rights {
		if _, ok := userRightsMap[right]; !ok {
			return false, nil
		}
	}

	return true, nil
}
