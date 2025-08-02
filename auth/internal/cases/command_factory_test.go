package cases_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/parta4ok/kvs/auth/internal/cases"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/stretchr/testify/require"
)

func TestNewCommandFactory(t *testing.T) {
	t.Parallel()

	type deps struct {
		storage     bool
		jwtProvider bool
		hasher      bool
		idGenerator bool
	}
	tests := []struct {
		name    string
		deps    deps
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no storage",
			deps:    deps{jwtProvider: true, hasher: true, idGenerator: true},
			wantErr: true,
			errMsg:  "storage not set",
		},
		{
			name:    "no jwtProvider",
			deps:    deps{storage: true, hasher: true, idGenerator: true},
			wantErr: true,
			errMsg:  "jwt provider not set",
		},
		{
			name:    "no hasher",
			deps:    deps{storage: true, jwtProvider: true, idGenerator: true},
			wantErr: true,
			errMsg:  "hasher not set",
		},
		{
			name:    "no idGenerator",
			deps:    deps{storage: true, jwtProvider: true, hasher: true},
			wantErr: true,
			errMsg:  "id generator not set",
		},
		{
			name:    "all deps",
			deps:    deps{storage: true, jwtProvider: true, hasher: true, idGenerator: true},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()
			ctrl := gomock.NewController(it)
			it.Cleanup(ctrl.Finish)

			opts := []cases.CommandFactoryOption{}
			if tc.deps.storage {
				opts = append(opts, cases.WithStorage(testdata.NewMockStorage(ctrl)))
			}
			if tc.deps.jwtProvider {
				opts = append(opts, cases.WithJWTProvider(testdata.NewMockJWTProvider(ctrl)))
			}
			if tc.deps.hasher {
				opts = append(opts, cases.WithHasher(testdata.NewMockHasher(ctrl)))
			}
			if tc.deps.idGenerator {
				opts = append(opts, cases.WithIDGenerator(testdata.NewMockIDGenerator(ctrl)))
			}

			factory, err := cases.NewCommandFactory(opts...)
			if tc.wantErr {
				require.Error(it, err)
				require.Nil(it, factory)
				require.Contains(it, err.Error(), tc.errMsg)
				return
			}
			require.NoError(it, err)
			require.NotNil(it, factory)
		})
	}
}

func TestCommandFactory_NewIntrospectedCommand(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory, err := cases.NewCommandFactory(
		cases.WithStorage(testdata.NewMockStorage(ctrl)),
		cases.WithJWTProvider(testdata.NewMockJWTProvider(ctrl)),
		cases.WithHasher(testdata.NewMockHasher(ctrl)),
		cases.WithIDGenerator(testdata.NewMockIDGenerator(ctrl)),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)

	cmd, err := factory.NewIntrospectedCommand(context.TODO(), "jwt-token")
	require.NoError(t, err)
	require.NotNil(t, cmd)
}

func TestCommandFactory_NewSignInCommand(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory, err := cases.NewCommandFactory(
		cases.WithStorage(testdata.NewMockStorage(ctrl)),
		cases.WithJWTProvider(testdata.NewMockJWTProvider(ctrl)),
		cases.WithHasher(testdata.NewMockHasher(ctrl)),
		cases.WithIDGenerator(testdata.NewMockIDGenerator(ctrl)),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)

	cmd, err := factory.NewSignInCommand(context.TODO(), "user", "pass")
	require.NoError(t, err)
	require.NotNil(t, cmd)
}

func TestCommandFactory_NewAddUserCommand(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory, err := cases.NewCommandFactory(
		cases.WithStorage(testdata.NewMockStorage(ctrl)),
		cases.WithJWTProvider(testdata.NewMockJWTProvider(ctrl)),
		cases.WithHasher(testdata.NewMockHasher(ctrl)),
		cases.WithIDGenerator(testdata.NewMockIDGenerator(ctrl)),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)

	cmd, err := factory.NewAddUserCommand(context.TODO(), "login", "pass", []string{"admin"}, map[string]string{"email": "test@test.com"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
}

func TestCommandFactory_NewDeleteUserCimmand(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory, err := cases.NewCommandFactory(
		cases.WithStorage(testdata.NewMockStorage(ctrl)),
		cases.WithJWTProvider(testdata.NewMockJWTProvider(ctrl)),
		cases.WithHasher(testdata.NewMockHasher(ctrl)),
		cases.WithIDGenerator(testdata.NewMockIDGenerator(ctrl)),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)

	cmd, err := factory.NewDeleteUserCommand(context.TODO(), "test_user_id")
	require.NoError(t, err)
	require.NotNil(t, cmd)
}
