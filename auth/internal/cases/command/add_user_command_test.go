package command_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/parta4ok/kvs/auth/internal/cases/command"

	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/stretchr/testify/require"
)

func TestAddUserCommand_Exec(t *testing.T) {
	t.Parallel()

	type stage struct {
		GetUserByUsernameSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, login string, user *entities.User, err error)
		HashSettings              func(ctx context.Context, t *testing.T, h *testdata.MockHasher, password string, hash string, err error)
		GenerateSettings          func(ctx context.Context, t *testing.T, g *testdata.MockIDGenerator, id string, err error)
		StoreUserSettings         func(ctx context.Context, t *testing.T, s *testdata.MockStorage, user *entities.User, err error)
		GenerateErr               error
		GetUserByUsernameErr      error
		HashErr                   error
		UserExists                bool
		StoreUserErr              error
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
				GetUserByUsernameSettings: setGetUserByUsernameAdd,
				GetUserByUsernameErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsernameAdd,
			},
			wantErr: true,
			resErr:  entities.ErrAlreadyExists,
		},
		{
			name: "3",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsernameAdd,
				GetUserByUsernameErr:      entities.ErrNotFound,
				GenerateSettings:          setGenerateID,
				GenerateErr:               errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "4",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsernameAdd,
				GetUserByUsernameErr:      entities.ErrNotFound,
				GenerateSettings:          setGenerateID,
				HashSettings:              setHash,
				HashErr:                   errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "5",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsernameAdd,
				GetUserByUsernameErr:      entities.ErrNotFound,
				GenerateSettings:          setGenerateID,
				HashSettings:              setHash,
				StoreUserSettings:         setStoreUser,
				StoreUserErr:              errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "6",
			stage: stage{
				GetUserByUsernameSettings: setGetUserByUsernameAdd,
				GetUserByUsernameErr:      entities.ErrNotFound,
				GenerateSettings:          setGenerateID,
				HashSettings:              setHash,
				StoreUserSettings:         setStoreUser,
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			ctrl := gomock.NewController(it)
			it.Cleanup(ctrl.Finish)

			storage := testdata.NewMockStorage(ctrl)
			hasher := testdata.NewMockHasher(ctrl)
			generator := testdata.NewMockIDGenerator(ctrl)

			ctx := context.TODO()
			login := "testuser"
			password := "testpass"
			rights := []string{"admin"}
			contacts := map[string]string{"email": "test@test.com"}

			inputUser := &entities.User{
				Username:     login,
				PasswordHash: password,
				Rights:       rights,
				Contacts:     contacts,
			}

			expectedUser := &entities.User{
				ID:           "new-id",
				Username:     login,
				PasswordHash: "hashed-pass",
				Rights:       rights,
				Contacts:     contacts,
			}

			if tc.stage.GetUserByUsernameSettings != nil {
				tc.stage.GetUserByUsernameSettings(ctx, it, storage, login, expectedUser, tc.stage.GetUserByUsernameErr)
			}

			if tc.stage.HashSettings != nil {
				tc.stage.HashSettings(ctx, it, hasher, password, "hashed-pass", tc.stage.HashErr)
			}

			if tc.stage.GenerateSettings != nil {
				tc.stage.GenerateSettings(ctx, it, generator, "new-id", tc.stage.GenerateErr)
			}

			if tc.stage.StoreUserSettings != nil {
				tc.stage.StoreUserSettings(ctx, it, storage, expectedUser, tc.stage.StoreUserErr)
			}

			command := command.NewAddUserCommand(storage, hasher, generator, inputUser)
			require.NotNil(it, command)

			res, err := command.Exec(ctx)
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Nil(it, res)
				return
			}

			require.NoError(it, err)
			require.Equal(it, &entities.CommandResult{Success: true, Message: expectedUser.ID}, res)
		})
	}
}

func setGetUserByUsernameAdd(ctx context.Context, t *testing.T, s *testdata.MockStorage, login string, user *entities.User, err error) {
	t.Helper()
	s.EXPECT().GetUserByUsername(ctx, login).Return(user, err)
}

func setHash(ctx context.Context, t *testing.T, h *testdata.MockHasher, password string, hash string, err error) {
	t.Helper()

	h.EXPECT().Hash(ctx, password).Return(hash, err)
}

func setGenerateID(ctx context.Context, t *testing.T, g *testdata.MockIDGenerator, id string, err error) {
	t.Helper()

	g.EXPECT().Generate(ctx).Return(id, err)
}

func setStoreUser(ctx context.Context, t *testing.T, s *testdata.MockStorage, user *entities.User, err error) {
	t.Helper()
	s.EXPECT().StoreUser(ctx, user).Return(err)
}
