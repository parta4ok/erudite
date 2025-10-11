package command_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/parta4ok/kvs/auth/internal/cases/command"
	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/stretchr/testify/require"
)

func TestNewGetMentorGroupsCommand(t *testing.T) {
	t.Parallel()

	type args struct {
		notNilStorage bool
		notEmptyID    bool
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
				notEmptyID: true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "2",
			args: args{
				notNilStorage: true,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "3",
			args: args{
				notNilStorage: true,
				notEmptyID:    true,
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
			ctx := context.TODO()
			var mentorID string

			if tc.args.notNilStorage {
				storage = testdata.NewMockStorage(ctrl)
			}

			if tc.args.notEmptyID {
				mentorID = "mentor-id"
			}

			command, err := command.NewGetMentorGroupsCommand(ctx, storage, mentorID)
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

			command, err := command.NewGetMentorGroupsCommand(ctx, storage, mentorID)
			require.NoError(it, err)
			require.NotNil(it, command)

			res, err := command.Exec()
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
