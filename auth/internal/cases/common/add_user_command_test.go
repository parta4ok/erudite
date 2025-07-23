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

func TestAddUserCommand(t *testing.T) {
	t.Parallel()

	type args struct {
		notNilStorage   bool
		notNilGenerator bool
		notNilHasher    bool
		notNilUser      bool
		notNilPassword  bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		resErr  error
	}{
		{
			name: "1",
			args: args{
				notNilGenerator: true,
				notNilHasher:    true,
				notNilUser:      true,
				notNilPassword:  true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "2",
			args: args{
				notNilStorage:  true,
				notNilHasher:   true,
				notNilUser:     true,
				notNilPassword: true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "3",
			args: args{
				notNilStorage:   true,
				notNilGenerator: true,
				notNilUser:      true,
				notNilPassword:  true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "4",
			args: args{
				notNilStorage:   true,
				notNilGenerator: true,
				notNilHasher:    true,
				notNilPassword:  true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "5",
			args: args{
				notNilStorage:   true,
				notNilGenerator: true,
				notNilHasher:    true,
				notNilUser:      true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "6",
			args: args{
				notNilStorage:   true,
				notNilGenerator: true,
				notNilHasher:    true,
				notNilUser:      true,
				notNilPassword:  true,
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

			var storage common.Storage
			var generator common.IDGenerator
			var hasher common.Hasher

			ctx := context.TODO()
			var user string
			var password string
			var rights []string
			var contacts map[string]string

			if tc.args.notNilStorage {
				storage = testdata.NewMockStorage(ctrl)
			}

			if tc.args.notNilGenerator {
				generator = testdata.NewMockIDGenerator(ctrl)
			}

			if tc.args.notNilHasher {
				hasher = testdata.NewMockHasher(ctrl)
			}

			if tc.args.notNilUser {
				user = "testuser"
			}

			if tc.args.notNilPassword {
				password = "testtest"
			}

			command, err := common.NewAddUserCommand(ctx, storage, hasher, generator, user, password, rights, contacts)
			if tc.wantErr {
				require.ErrorIs(t, err, tc.resErr)
				require.Nil(t, command)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, command)

		})
	}
}

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
			user := &entities.User{
				ID:           "new-id",
				Username:     login,
				PasswordHash: "hashed-pass",
				Rights:       rights,
				Contacts:     contacts,
			}

			if tc.stage.GetUserByUsernameSettings != nil {
				tc.stage.GetUserByUsernameSettings(ctx, it, storage, login, user, tc.stage.GetUserByUsernameErr)
			}

			if tc.stage.HashSettings != nil {
				tc.stage.HashSettings(ctx, it, hasher, password, "hashed-pass", tc.stage.HashErr)
			}

			if tc.stage.GenerateSettings != nil {
				tc.stage.GenerateSettings(ctx, it, generator, "new-id", tc.stage.GenerateErr)
			}

			if tc.stage.StoreUserSettings != nil {
				tc.stage.StoreUserSettings(ctx, it, storage, user, tc.stage.StoreUserErr)
			}

			command, err := common.NewAddUserCommand(ctx, storage, hasher, generator, login, password, rights, contacts)
			require.NoError(it, err)
			require.NotNil(it, command)

			res, err := command.Exec()
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Nil(it, res)
				return
			}

			require.NoError(it, err)
			require.Equal(it, &entities.CommandResult{Success: true, Message: user.ID}, res)
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
