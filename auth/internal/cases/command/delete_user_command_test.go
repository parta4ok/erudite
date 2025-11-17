package command_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/auth/internal/cases/command"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/parta4ok/kvs/auth/internal/entities"
)

func TestDeleteUserCommand_Exec(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.TODO()
	mockStorage := testdata.NewMockStorage(ctrl)
	userID := "some-uid"

	type stage struct {
		RemoveUserSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, userID string, err error)
		RemoveUserErr      error
	}

	tests := []struct {
		name    string
		stage   stage
		wantErr bool
		resErr  error
	}{
		{
			name: "RemoveUser returns error",
			stage: stage{
				RemoveUserSettings: setRemoveUser,
				RemoveUserErr:      entities.ErrNotFound,
			},
			wantErr: true,
			resErr:  entities.ErrNotFound,
		},
		{
			name: "RemoveUser success",
			stage: stage{
				RemoveUserSettings: setRemoveUser,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.stage.RemoveUserSettings != nil {
				tc.stage.RemoveUserSettings(ctx, t, mockStorage, userID, tc.stage.RemoveUserErr)
			}
			cmd := command.NewDeleteUserCommand(mockStorage, userID)
			require.NotNil(t, cmd)

			res, err := cmd.Exec(ctx)
			if tc.wantErr {
				require.ErrorIs(t, err, tc.resErr)
				require.Nil(t, res)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			require.True(t, res.Success)
		})
	}
}

func setRemoveUser(ctx context.Context, t *testing.T, s *testdata.MockStorage, userID string, err error) {
	t.Helper()

	s.EXPECT().RemoveUser(ctx, userID).Return(err)
}
