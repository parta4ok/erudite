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

func TestGetGroupTitleByIDCommand_Exec(t *testing.T) {
	t.Parallel()

	type stage struct {
		GetGroupTitleByIDSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, groupID string, title string, err error)
		GetGroupTitleByIDErr      error
		groupID                   string
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
				groupID: "",
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "2",
			stage: stage{
				groupID:                   "group-id",
				GetGroupTitleByIDSettings: setGetGroupTitleByID,
				GetGroupTitleByIDErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "3",
			stage: stage{
				groupID:                   "group-id",
				GetGroupTitleByIDSettings: setGetGroupTitleByID,
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

			expectedTitle := "Group Title"

			if tc.stage.GetGroupTitleByIDSettings != nil {
				tc.stage.GetGroupTitleByIDSettings(ctx, it, storage, tc.stage.groupID, expectedTitle, tc.stage.GetGroupTitleByIDErr)
			}

			command := command.NewGetGroupTitleByIDCommand(storage, tc.stage.groupID)
			require.NotNil(it, command)

			res, err := command.Exec(ctx)
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Nil(it, res)
				return
			}

			require.NoError(it, err)
			require.Equal(it, &entities.CommandResult{Success: true, Payload: expectedTitle}, res)
		})
	}
}

func setGetGroupTitleByID(ctx context.Context, t *testing.T, s *testdata.MockStorage, groupID string, title string, err error) {
	t.Helper()
	s.EXPECT().GetGroupTitleByID(ctx, groupID).Return(title, err)
}
