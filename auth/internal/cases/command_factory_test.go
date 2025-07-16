package cases_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/parta4ok/kvs/auth/internal/cases"
	"github.com/parta4ok/kvs/auth/internal/cases/common/testdata"
	"github.com/stretchr/testify/require"
)

func TestNewCommandFactory_AllOptionsSet(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	st := testdata.NewMockStorage(ctrl)
	jp := testdata.NewMockJWTProvider(ctrl)

	factory, err := cases.NewCommandFactory(
		cases.WithStorage(st),
		cases.WithJWTProvider(jp),
	)
	require.NoError(t, err)
	require.NotNil(t, factory)
}

func TestNewCommandFactory_NoStorage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	jp := testdata.NewMockJWTProvider(ctrl)

	factory, err := cases.NewCommandFactory(
		cases.WithJWTProvider(jp),
	)
	require.Error(t, err)
	require.Nil(t, factory)
	require.Contains(t, err.Error(), "storage not set")
}

func TestNewCommandFactory_NoJWTProvider(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	st := testdata.NewMockStorage(ctrl)

	factory, err := cases.NewCommandFactory(
		cases.WithStorage(st),
	)
	require.Error(t, err)
	require.Nil(t, factory)
	require.Contains(t, err.Error(), "jwt provider not set")
}

func TestCommandFactory_NewIntrospectedCommand(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	st := testdata.NewMockStorage(ctrl)
	jp := testdata.NewMockJWTProvider(ctrl)

	factory, _ := cases.NewCommandFactory(
		cases.WithStorage(st),
		cases.WithJWTProvider(jp),
	)
	cmd, err := factory.NewIntrospectedCommand(context.Background(), "jwt-token")
	require.NoError(t, err)
	require.NotNil(t, cmd)
}

func TestCommandFactory_NewSignInCommand(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	st := testdata.NewMockStorage(ctrl)
	jp := testdata.NewMockJWTProvider(ctrl)

	factory, _ := cases.NewCommandFactory(
		cases.WithStorage(st),
		cases.WithJWTProvider(jp),
	)
	cmd, err := factory.NewSignInCommand(context.Background(), "user", "pass")
	require.NoError(t, err)
	require.NotNil(t, cmd)
}
