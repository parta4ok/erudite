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

// var errTest = entities.ErrInvalidParam

func TestNewSignInCommand(t *testing.T) {
	t.Parallel()

	type cases struct {
		BadUserName bool
		BadPassword bool
		NilStorage  bool
		NilProvider bool
		NilHasher   bool
	}
	tests := []struct {
		name    string
		cases   cases
		wantErr bool
		resErr  error
	}{
		{"bad username", cases{BadUserName: true}, true, entities.ErrInvalidParam},
		{"bad password", cases{BadPassword: true}, true, entities.ErrInvalidParam},
		{"nil storage", cases{NilStorage: true}, true, entities.ErrInvalidParam},
		{"nil provider", cases{NilProvider: true}, true, entities.ErrInvalidParam},
		{"nil hasher", cases{NilHasher: true}, true, entities.ErrInvalidParam},
		{"ok", cases{}, false, nil},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()
			ctrl := gomock.NewController(it)
			it.Cleanup(ctrl.Finish)

			var provider common.JWTProvider
			var storage common.Storage
			var hasher common.Hasher

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
			if !tc.cases.NilHasher {
				hasher = testdata.NewMockHasher(ctrl)
			}

			command, err := common.NewSignInCommand(ctx, storage, provider, hasher, userName, password)
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Nil(it, command)
				return
			}
			require.NoError(it, err)
			require.NotNil(it, command)
		})
	}
}

func TestSignInCommand_Exec(t *testing.T) {
	t.Parallel()

	type stage struct {
		GetUserByUsernameSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, name string, user *entities.User, err error)
		GetUserByUsernameErr      error
		IsHashSettings            func(ctx context.Context, t *testing.T, h *testdata.MockHasher, reqPass, hashPass string, result bool, err error)
		IsHashErr                 error
		IsHashResult              bool
		GenerateSettings          func(ctx context.Context, t *testing.T, g *testdata.MockJWTProvider, user *entities.User, jwt string, err error)
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
				IsHashSettings:            setIsHash,
				IsHashResult:              false,
				IsHashErr:                 errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "3",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsername,
				IsHashSettings:            setIsHash,
				IsHashResult:              false,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidPassword,
		},
		{
			name: "4",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsername,
				IsHashSettings:            setIsHash,
				IsHashResult:              true,
				GenerateSettings:          setGenerate,
				GenerateErr:               errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "5",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsername,
				IsHashSettings:            setIsHash,
				IsHashResult:              true,
				GenerateSettings:          setGenerate,
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			ctrl := gomock.NewController(it)
			it.Cleanup(ctrl.Finish)

			provider := testdata.NewMockJWTProvider(ctrl)
			storage := testdata.NewMockStorage(ctrl)
			hasher := testdata.NewMockHasher(ctrl)

			ctx := context.TODO()
			name := "testname"
			password := "testpassword"
			passwordHash := "testpasswordhash"

			jwt := "testjwt"

			user := &entities.User{
				ID:           "testID",
				Username:     name,
				PasswordHash: passwordHash,
				Rights:       []string{},
				Contacts:     map[string]string{},
			}

			if tc.stage.GetUserByUsernameSettings != nil {
				tc.stage.GetUserByUsernameSettings(ctx, it, storage, name, user, tc.stage.GetUserByUsernameErr)
			}

			if tc.stage.IsHashSettings != nil {
				tc.stage.IsHashSettings(ctx, it, hasher, password, passwordHash, tc.stage.IsHashResult, tc.stage.IsHashErr)
			}

			if tc.stage.GenerateSettings != nil {
				tc.stage.GenerateSettings(ctx, it, provider, user, jwt, tc.stage.GenerateErr)
			}

			command, err := common.NewSignInCommand(ctx, storage, provider, hasher, name, password)
			require.NoError(t, err)
			require.NotNil(t, command)

			res, err := command.Exec()
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Nil(it, res)
				return
			}
			require.NoError(it, err)
			require.Equal(it, &entities.CommandResult{Success: true, Message: jwt}, res)
		})
	}
}

func setGetUserByUsername(ctx context.Context, t *testing.T, s *testdata.MockStorage, name string, user *entities.User, err error) {
	t.Helper()

	s.EXPECT().GetUserByUsername(ctx, name).Return(user, err)
}

func setIsHash(ctx context.Context, t *testing.T, h *testdata.MockHasher, reqPass, hashPass string, result bool, err error) {
	t.Helper()

	h.EXPECT().IsHash(ctx, reqPass, hashPass).Return(result, err)
}

func setGenerate(ctx context.Context, t *testing.T, g *testdata.MockJWTProvider, user *entities.User, jwt string, err error) {
	t.Helper()

	g.EXPECT().Generate(user).Return(jwt, err)
}
