package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/parta4ok/kvs/auth/internal/cases/command"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/stretchr/testify/require"
)

var (
	ErrTest = errors.New("test error")
)

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

			command := command.NewUpdateUserCommand(storage, hasher, updatedUser)
			require.NotNil(it, command)

			result, err := command.Exec(ctx)
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
