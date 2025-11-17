package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/parta4ok/kvs/auth/internal/cases/command"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/stretchr/testify/require"
)

var (
	errTest = errors.New("test error")
)

func TestIntrospectCommand_Exec(t *testing.T) {
	t.Parallel()

	type stage struct {
		GetUserByIDSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, userID string, user *entities.User, err error)
		GetUserByIDErr      error
		IntrospectSettings  func(t *testing.T, p *testdata.MockJWTProvider, jwt string, claims *entities.UserClaims, err error)
		IntrospectErr       error
		RightsProblem       bool
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
				IntrospectSettings: setIntrospect,
				IntrospectErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stage: stage{
				IntrospectSettings:  setIntrospect,
				GetUserByIDSettings: setGetUserByID,
				GetUserByIDErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "3",
			stage: stage{
				IntrospectSettings:  setIntrospect,
				GetUserByIDSettings: setGetUserByID,
				RightsProblem:       true,
			},
			wantErr: true,
			resErr:  entities.ErrForbidden,
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
				tc.stage.IntrospectSettings(it, jwtProvider, jwt, claims, tc.stage.IntrospectErr)
			}

			cmd := command.NewIntrospectCommand(jwt, storage, jwtProvider)
			require.NotNil(it, cmd)

			res, err := cmd.Exec(ctx)
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
