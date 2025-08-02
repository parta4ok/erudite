package common_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/parta4ok/kvs/auth/internal/entities"
)

func TestNewDeleteUserCommand(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	mockStorage := testdata.NewMockStorage(gomock.NewController(t))

	tests := []struct {
		name    string
		storage common.Storage
		userID  string
		wantErr bool
		resErr  error
	}{
		{
			name:    "nil storage",
			userID:  "uid",
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name:    "empty userID",
			storage: mockStorage,
			userID:  "",
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name:    "success",
			storage: mockStorage,
			userID:  "uid",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := common.NewDeleteUserCommand(ctx, tc.storage, tc.userID)
			if tc.wantErr {
				require.ErrorIs(t, err, tc.resErr)
				require.Nil(t, cmd)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cmd)
		})
	}
}

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
			cmd, err := common.NewDeleteUserCommand(ctx, mockStorage, userID)
			require.NoError(t, err)
			require.NotNil(t, cmd)
			res, err := cmd.Exec()
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
