package cases_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/parta4ok/kvs/auth/internal/cases"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/stretchr/testify/require"
)

func TestNewCommandFactory(t *testing.T) {
	t.Parallel()

	type deps struct {
		storage      bool
		jwtProvider  bool
		hasher       bool
		idGenerator  bool
		messgeBroker bool
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
			name:    "no broker",
			deps:    deps{storage: true, jwtProvider: true, hasher: true, idGenerator: true},
			wantErr: true,
			errMsg:  "message broker not set",
		},
		{
			name: "all deps",
			deps: deps{
				storage:      true,
				jwtProvider:  true,
				hasher:       true,
				idGenerator:  true,
				messgeBroker: true,
			},
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
			if tc.deps.messgeBroker{
				opts = append(opts, cases.WithMessageBroker(testdata.NewMockMessageBroker(ctrl)))
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
		cases.WithMessageBroker(testdata.NewMockMessageBroker(ctrl)),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)

	cmd := factory.NewIntrospectedCommand("jwt-token")
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
		cases.WithMessageBroker(testdata.NewMockMessageBroker(ctrl)),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)

	cmd := factory.NewSignInCommand("user", "pass")
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
		cases.WithMessageBroker(testdata.NewMockMessageBroker(ctrl)),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)

	user, err := entities.NewUser(uuid.NewString(), uuid.NewString(), uuid.NewString(),
		[]string{"student"}, nil, "")
	require.NoError(t, err)

	cmd := factory.NewAddUserCommand(user)
	require.NotNil(t, cmd)
}

func TestCommandFactory_NewDeleteUserCommand(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory, err := cases.NewCommandFactory(
		cases.WithStorage(testdata.NewMockStorage(ctrl)),
		cases.WithJWTProvider(testdata.NewMockJWTProvider(ctrl)),
		cases.WithHasher(testdata.NewMockHasher(ctrl)),
		cases.WithIDGenerator(testdata.NewMockIDGenerator(ctrl)),
		cases.WithMessageBroker(testdata.NewMockMessageBroker(ctrl)),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)

	cmd := factory.NewDeleteUserCommand("test_user_id")
	require.NotNil(t, cmd)
}

func TestCommandFactory_NewUpdateUserCommand(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory, err := cases.NewCommandFactory(
		cases.WithStorage(testdata.NewMockStorage(ctrl)),
		cases.WithJWTProvider(testdata.NewMockJWTProvider(ctrl)),
		cases.WithHasher(testdata.NewMockHasher(ctrl)),
		cases.WithIDGenerator(testdata.NewMockIDGenerator(ctrl)),
		cases.WithMessageBroker(testdata.NewMockMessageBroker(ctrl)),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)

	cmd := factory.NewUpdateUserCommand(&entities.User{ID: uuid.NewString()})
	require.NotNil(t, cmd)
}

func TestCommandFactory_NewGetLinkedUsersCommand(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory, err := cases.NewCommandFactory(
		cases.WithStorage(testdata.NewMockStorage(ctrl)),
		cases.WithJWTProvider(testdata.NewMockJWTProvider(ctrl)),
		cases.WithHasher(testdata.NewMockHasher(ctrl)),
		cases.WithIDGenerator(testdata.NewMockIDGenerator(ctrl)),
		cases.WithMessageBroker(testdata.NewMockMessageBroker(ctrl)),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)

	cmd := factory.NewGetLinkedUsersCommand(uuid.NewString())
	require.NoError(t, err)
	require.NotNil(t, cmd)
}
