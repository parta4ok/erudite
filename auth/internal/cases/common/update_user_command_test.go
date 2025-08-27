package common_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/stretchr/testify/require"
)

var (
	ErrTest = errors.New("test error")
)

func TestNewUpdateUserCommand(t *testing.T) {
	t.Parallel()

	type args struct {
		storage func(t *testing.T, ctrl *gomock.Controller) common.Storage
		hasher  func(t *testing.T, ctrl *gomock.Controller) common.Hasher
		user    func(t *testing.T) *entities.User
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		resErr  error
	}{
		{
			name: "nil storage - failure",
			args: args{
				storage: getNilStorage,
				hasher:  getHasher,
				user:    getUser,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "nil hasher - failure",
			args: args{
				storage: getStorage,
				hasher:  getNilHasher,
				user:    getUser,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "nil user - failure",
			args: args{
				storage: getStorage,
				hasher:  getHasher,
				user:    getNilUser,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			ctx := context.Background()

			ctrl := gomock.NewController(it)
			it.Cleanup(func() {
				ctrl.Finish()
			})

			res, err := common.NewUpdateUserCommand(
				ctx,
				tc.args.storage(it, ctrl),
				tc.args.hasher(it, ctrl),
				tc.args.user(it),
			)
			if tc.wantErr {
				require.Nil(it, res)
				require.ErrorIs(it, err, tc.resErr)
				return
			}

			require.NoError(it, err)
			require.Equal(it, nil, res)
		})
	}
}

func getStorage(t *testing.T, ctrl *gomock.Controller) common.Storage {
	t.Helper()

	return testdata.NewMockStorage(ctrl)
}

func getNilStorage(t *testing.T, _ *gomock.Controller) common.Storage {
	t.Helper()

	var storage common.Storage
	return storage
}

func getHasher(t *testing.T, ctrl *gomock.Controller) common.Hasher {
	t.Helper()

	return testdata.NewMockHasher(ctrl)
}

func getNilHasher(t *testing.T, ctrl *gomock.Controller) common.Hasher {
	t.Helper()

	var hasher common.Hasher
	return hasher
}

func getUser(t *testing.T) *entities.User {
	t.Helper()

	return &entities.User{
		ID: uuid.NewString(),
	}
}

func getNilUser(t *testing.T) *entities.User {
	t.Helper()

	var user *entities.User
	return user
}

func TestUpdateUserCommand_Exec(t *testing.T) {
	t.Parallel()

	type fields struct {
		changedPass       bool
		hashStageSettings func(ctx context.Context, t *testing.T, hasher *testdata.MockHasher, pass string, hash string, err error)
		hashStageErr      error
		updateSerrings    func(ctx context.Context, t *testing.T, storage *testdata.MockStorage, user *entities.User, err error)
		updateErr         error
	}
	tests := []struct {
		name    string
		fields  fields
		want    *entities.CommandResult
		wantErr bool
		resErr  error
	}{
		{
			name: "changed pass hash err",
			fields: fields{
				changedPass:       true,
				hashStageSettings: setHashStage,
				hashStageErr:      ErrTest,
			},
			wantErr: true,
			resErr:  ErrTest,
		},
		{
			name: "changed pass and then update err",
			fields: fields{
				changedPass:       true,
				hashStageSettings: setHashStage,
				updateSerrings:    setUpdate,
				updateErr:         ErrTest,
			},
			wantErr: true,
			resErr:  ErrTest,
		},
		{
			name: "changed pass and then update",
			fields: fields{
				changedPass:       true,
				hashStageSettings: setHashStage,
				updateSerrings:    setUpdate,
			},
			wantErr: false,
		},
		{
			name: "update",
			fields: fields{
				updateSerrings: setUpdate,
			},
			wantErr: false,
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

			ctx := context.TODO()
			storage := testdata.NewMockStorage(ctrl)
			hasher := testdata.NewMockHasher(ctrl)
			updatedUser := &entities.User{
				ID:      uuid.NewString(),
				GroupID: uuid.NewString(),
			}

			if tc.fields.changedPass {
				updatedUser.PasswordHash = uuid.NewString()
			}

			if tc.fields.hashStageSettings != nil {
				tc.fields.hashStageSettings(ctx, it, hasher, updatedUser.PasswordHash, uuid.NewString(), tc.fields.hashStageErr)
			}

			if tc.fields.updateSerrings != nil {
				tc.fields.updateSerrings(ctx, it, storage, updatedUser, tc.fields.updateErr)
			}

			command, err := common.NewUpdateUserCommand(ctx, storage, hasher, updatedUser)
			require.NoError(it, err)

			result, err := command.Exec()
			if tc.wantErr {
				require.Nil(it, result)
				require.ErrorIs(it, err, tc.resErr)
				return
			}
			require.NoError(it, err)
			require.Equal(it, &entities.CommandResult{Success: true, Message: updatedUser.ID}, result)
		})
	}
}

func setHashStage(ctx context.Context, t *testing.T, hasher *testdata.MockHasher, pass string, hash string, err error) {
	t.Helper()

	hasher.EXPECT().Hash(ctx, pass).Return(hash, err)
}

func setUpdate(ctx context.Context, t *testing.T, storage *testdata.MockStorage, user *entities.User, err error) {
	t.Helper()

	storage.EXPECT().UpdateUser(ctx, user).Return(err)
}
