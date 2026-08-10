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

func TestGetAllGroupsCommand_Exec(t *testing.T) {
	t.Parallel()

	type stage struct {
		GetAllGroupsSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, groups []*entities.Group, err error)
		GetAllGroupsErr      error
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
				GetAllGroupsSettings: setGetAllGroups,
				GetAllGroupsErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stage: stage{
				GetAllGroupsSettings: setGetAllGroups,
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

			expectedGroups := []*entities.Group{
				entities.NewGroup("group-1", "Group 1"),
				entities.NewGroup("group-2", "Group 2"),
			}

			if tc.stage.GetAllGroupsSettings != nil {
				tc.stage.GetAllGroupsSettings(ctx, it, storage, expectedGroups, tc.stage.GetAllGroupsErr)
			}

			cmd := command.NewGetAllGroupsCommand(storage)
			require.NotNil(it, cmd)

			res, err := cmd.Exec(ctx)
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Nil(it, res)
				return
			}

			require.NoError(it, err)
			require.Equal(it, &entities.CommandResult{Success: true, Payload: expectedGroups}, res)
		})
	}
}

func setGetAllGroups(ctx context.Context, t *testing.T, s *testdata.MockStorage, groups []*entities.Group, err error) {
	t.Helper()
	s.EXPECT().GetAllGroups(ctx).Return(groups, err)
}
