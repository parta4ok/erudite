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

func TestGetAllUsersCommand_Exec(t *testing.T) {
	t.Parallel()

	type stage struct {
		GetAllUsersSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, users []*entities.User, err error)
		GetAllUsersErr      error
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
				GetAllUsersSettings: setGetAllUsers,
				GetAllUsersErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stage: stage{
				GetAllUsersSettings: setGetAllUsers,
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
			ctx := context.TODO()

			expectedUsers := []*entities.User{
				{ID: "user-1", Username: "user1", FullName: "User One", Rights: []string{"student"}},
				{ID: "user-2", Username: "user2", FullName: "User Two", Rights: []string{"mentor"}},
			}

			if tc.stage.GetAllUsersSettings != nil {
				tc.stage.GetAllUsersSettings(ctx, it, storage, expectedUsers, tc.stage.GetAllUsersErr)
			}

			cmd := command.NewGetAllUsersCommand(storage)
			require.NotNil(it, cmd)

			res, err := cmd.Exec(ctx)
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Nil(it, res)
				return
			}

			require.NoError(it, err)
			require.Equal(it, &entities.CommandResult{Success: true, Payload: expectedUsers}, res)
		})
	}
}

func setGetAllUsers(ctx context.Context, t *testing.T, s *testdata.MockStorage, users []*entities.User, err error) {
	t.Helper()
	s.EXPECT().GetAllUsers(ctx).Return(users, err)
}
