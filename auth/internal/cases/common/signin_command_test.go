package common_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/stretchr/testify/require"
)

func TestNewSignInCommand(t *testing.T) {
	t.Parallel()

	type cases struct {
		BadUserName bool
		BadPassword bool
		NilStorage  bool
		NilProvider bool
	}
	tests := []struct {
		name    string
		cases   cases
		wantErr bool
		resErr  error
	}{
		{
			name: "1",
			cases: cases{
				BadUserName: true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "2",
			cases: cases{
				BadPassword: true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "3",
			cases: cases{
				NilStorage: true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "4",
			cases: cases{
				NilProvider: true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "5",
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

			var provider common.JWTProvider
			var storage common.Storage

			ctx := context.TODO()

			password := "testPassword"
			userName := "testUserName"

			if tc.cases.BadUserName {
				userName = ""
			}

			if tc.cases.BadPassword {
				password = ""
			}

			if !tc.cases.NilProvider {
				provider = testdata.NewMockJWTProvider(ctrl)
			}

			if !tc.cases.NilStorage {
				storage = testdata.NewMockStorage(ctrl)
			}

			command, err := common.NewSignInCommand(ctx, userName, password, storage, provider)
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Nil(t, command)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, command)
		})
	}
}

func TestSignInCommand_Exec(t *testing.T) {
	t.Parallel()

	type stage struct {
		GetUserByUsernameSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, name string, user *entities.User, err error)
		GetUserByUsernameErr      error
		IncorrectPass             bool
		GenerateSettings          func(t *testing.T, g *testdata.MockJWTProvider, user *entities.User, jwt string, err error)
		GenerateErr               error
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
				GetUserByUsernameSettings: setGetUserByUsername,
				GetUserByUsernameErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsername,
				IncorrectPass:             true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidPassword,
		},
		{
			name: "3",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsername,
				GenerateSettings:          setGenerate,
				GenerateErr:               errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "4",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsername,
				GenerateSettings:          setGenerate,
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

			provider := testdata.NewMockJWTProvider(ctrl)
			storage := testdata.NewMockStorage(ctrl)

			ctx := context.TODO()
			userName := "testUserName"
			password := `password123`
			jwt := "testjwt"
			user := &entities.User{
				ID:           "1",
				Username:     "testUserName",
				PasswordHash: "$2a$10$ft9DCzVOqK1EzQ.tLgAAVOBG.89o0zjQqzWpqRrtKdcv1iEu/G84u",
				Rights:       []string{"read", "write"},
				Contacts:     map[string]string{"phone": "89123131231"},
			}

			if tc.stage.IncorrectPass {
				password = "incorrectPasswort"
			}

			if tc.stage.GetUserByUsernameSettings != nil {
				tc.stage.GetUserByUsernameSettings(ctx, it, storage, userName, user, tc.stage.GetUserByUsernameErr)
			}

			if tc.stage.GenerateSettings != nil {
				tc.stage.GenerateSettings(it, provider, user, jwt, tc.stage.GenerateErr)
			}

			command, err := common.NewSignInCommand(ctx, userName, password, storage, provider)
			require.NoError(it, err)
			require.NotNil(t, command)

			res, err := command.Exec()
			if tc.wantErr {
				require.ErrorIs(t, err, tc.resErr)
				require.Nil(t, res)
				return
			}
			require.NoError(t, err)
			require.Equal(t, &entities.CommandResult{Success: true, Message: jwt}, res)
		})
	}
}

func setGetUserByUsername(ctx context.Context, t *testing.T, s *testdata.MockStorage, name string, user *entities.User, err error) {
	t.Helper()

	s.EXPECT().GetUserByUsername(ctx, name).Return(user, err)
}

func setGenerate(t *testing.T, g *testdata.MockJWTProvider, user *entities.User, jwt string, err error) {
	t.Helper()

	g.EXPECT().Generate(user).Return(jwt, err)
}
