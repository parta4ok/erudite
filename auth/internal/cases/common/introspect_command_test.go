package common_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/stretchr/testify/require"
)

var (
	errTest = errors.New("test error")
)

func TestNewIntrospectCommand(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	ctx := context.TODO()
	storage := testdata.NewMockStorage(ctrl)
	provider := testdata.NewMockJWTProvider(ctrl)

	command, err := common.NewIntrospectCommand(ctx, "", "jwt", storage, provider)
	require.ErrorIs(t, err, entities.ErrInvalidParam)
	require.Contains(t, err.Error(), "user ID is required")
	require.Nil(t, command)

	command, err = common.NewIntrospectCommand(ctx, "user1", "", storage, provider)
	require.ErrorIs(t, err, entities.ErrInvalidJWT)
	require.Contains(t, err.Error(), "jwt is required")
	require.Nil(t, command)

	cmd, err := common.NewIntrospectCommand(ctx, "user1", "jwt", storage, provider)
	require.NoError(t, err)
	require.NotNil(t, cmd)
}

func TestIntrospectCommand_Exec(t *testing.T) {
	t.Parallel()

	type stage struct {
		GetUserByIDSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, userID string, user *entities.User, err error)
		GetUserByIDErr      error
		IntrospectSettings  func(t *testing.T, p *testdata.MockJWTProvider, jwt string, claims *entities.UserClaims, err error)
		IntrospectErr       error
		RightsProblem       bool
		FakeSub             bool
	}

	tests := []struct {
		name    string
		stage   stage
		wantErr bool
		resErr  error
	}{
		{
			name: "1",
			stage: stage{
				GetUserByIDSettings: setGetUserByID,
				GetUserByIDErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stage: stage{
				GetUserByIDSettings: setGetUserByID,
				IntrospectSettings:  setIntrospect,
				IntrospectErr:       errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "3",
			stage: stage{
				GetUserByIDSettings: setGetUserByID,
				IntrospectSettings:  setIntrospect,
				RightsProblem:       true,
			},
			wantErr: true,
			resErr:  entities.ErrForbidden,
		},
		{
			name: "4",
			stage: stage{
				GetUserByIDSettings: setGetUserByID,
				IntrospectSettings:  setIntrospect,
				FakeSub:             true,
			},
			wantErr: true,
			resErr:  entities.ErrForbidden,
		},
		{
			name: "5",
			stage: stage{
				GetUserByIDSettings: setGetUserByID,
				IntrospectSettings:  setIntrospect,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			ctrl := gomock.NewController(it)
			it.Cleanup(func() {
				ctrl.Finish()
			})

			storage := testdata.NewMockStorage(ctrl)
			jwtProvider := testdata.NewMockJWTProvider(ctrl)

			ctx := context.TODO()
			user := &entities.User{
				ID:       "1",
				Username: "user",
				Rights:   []string{"test", "view_result"},
			}
			jwt := "simpletext"

			claims := &entities.UserClaims{
				Username: "user",
				Issuer:   "erudite",
				Audience: []string{"students"},
				Subject:  "1",
				Rights:   []string{"test", "view_result"},
			}

			if tc.stage.GetUserByIDSettings != nil {
				tc.stage.GetUserByIDSettings(ctx, it, storage, user.ID, user, tc.stage.GetUserByIDErr)
			}

			if tc.stage.IntrospectSettings != nil {
				if tc.stage.RightsProblem {
					user.Rights = []string{"another_right"}
				}
				if tc.stage.FakeSub {
					claims.Subject = "2"
				}
				tc.stage.IntrospectSettings(it, jwtProvider, jwt, claims, tc.stage.IntrospectErr)
			}

			cmd, err := common.NewIntrospectCommand(ctx, user.ID, jwt, storage, jwtProvider)
			require.NoError(t, err)
			res, err := cmd.Exec()
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Nil(it, res)
				return
			}
			require.NoError(it, err)
			require.Equal(it, &entities.CommandResult{Success: true}, res)
		})
	}
}

func setGetUserByID(ctx context.Context, t *testing.T, s *testdata.MockStorage, userID string, user *entities.User, err error) {
	t.Helper()

	s.EXPECT().GetUserByID(ctx, userID).Return(user, err)
}

func setIntrospect(t *testing.T, p *testdata.MockJWTProvider, jwt string, claims *entities.UserClaims, err error) {
	t.Helper()

	p.EXPECT().Introspect(jwt).Return(claims, err)
}
