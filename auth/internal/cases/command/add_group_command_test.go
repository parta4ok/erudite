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

func TestAddGroupCommand_Exec(t *testing.T) {
	t.Parallel()

	type stage struct {
		GenerateSettings func(ctx context.Context, t *testing.T, g *testdata.MockIDGenerator, id string, err error)
		AddGroupSettings func(ctx context.Context, t *testing.T, s *testdata.MockStorage, gid, title, linkedID string, err error)
		GenerateErr      error
		AddGroupErr      error
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
				GenerateSettings: setGenerateID,
				GenerateErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stage: stage{
				GenerateSettings: setGenerateID,
				AddGroupSettings: setAddGroup,
				AddGroupErr:      errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "3",
			stage: stage{
				GenerateSettings: setGenerateID,
				AddGroupSettings: setAddGroup,
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
			generator := testdata.NewMockIDGenerator(ctrl)
			ctx := context.TODO()
			title := "test-title"
			linkedID := "linked-id"
			generatedID := "generated-id"

			if tc.stage.GenerateSettings != nil {
				tc.stage.GenerateSettings(ctx, it, generator, generatedID, tc.stage.GenerateErr)
			}

			if tc.stage.AddGroupSettings != nil {
				tc.stage.AddGroupSettings(ctx, it, storage, generatedID, title, linkedID, tc.stage.AddGroupErr)
			}

			command := command.NewAddGroupCommand(storage, generator, title, linkedID)
			require.NotNil(it, command)

			res, err := command.Exec(ctx)
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Nil(it, res)
				return
			}

			require.NoError(it, err)
			require.Equal(it, &entities.CommandResult{Success: true, Message: generatedID}, res)
		})
	}
}

func setAddGroup(ctx context.Context, t *testing.T, s *testdata.MockStorage, gid, title, linkedID string, err error) {
	t.Helper()
	s.EXPECT().AddGroup(ctx, gid, title, linkedID).Return(err)
}
