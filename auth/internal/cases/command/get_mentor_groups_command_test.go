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

func TestGetMentorGroupsCommand_Exec(t *testing.T) {
	t.Parallel()

	type stage struct {
		GetMentorGroupsSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, mentorID string, groups []*entities.Group, err error)
		GetMentorGroupsErr      error
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
				GetMentorGroupsSettings: setGetMentorGroups,
				GetMentorGroupsErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stage: stage{
				GetMentorGroupsSettings: setGetMentorGroups,
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
			mentorID := "mentor-id"

			expectedGroups := []*entities.Group{
				entities.NewGroup("group-1", "Group 1"),
				entities.NewGroup("group-2", "Group 2"),
			}

			if tc.stage.GetMentorGroupsSettings != nil {
				tc.stage.GetMentorGroupsSettings(ctx, it, storage, mentorID, expectedGroups, tc.stage.GetMentorGroupsErr)
			}

			command := command.NewGetMentorGroupsCommand(storage, mentorID)
			require.NotNil(it, command)

			res, err := command.Exec(ctx)
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

func setGetMentorGroups(ctx context.Context, t *testing.T, s *testdata.MockStorage, mentorID string, groups []*entities.Group, err error) {
	t.Helper()
	s.EXPECT().GetMentorGroups(ctx, mentorID).Return(groups, err)
}
