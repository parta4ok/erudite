//go:build KVS_TEST_L1

package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/auth/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/auth/internal/entities"
)

var (
	cstr = os.Getenv("TEST_PG_CONN")
)

func makeDB(t *testing.T, opts ...postgres.StorageOption) *postgres.Storage {
	t.Helper()

	db, err := postgres.NewStorage(cstr, opts...)
	require.NoError(t, err)
	require.NotNil(t, db)

	return db
}

func TestStorage_GetUserByID(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	ctx := context.TODO()
	var UserID = "1"

	user, err := db.GetUserByID(ctx, UserID)
	require.NoError(t, err)

	require.Equal(t, user.Username, "admin")

	UserID = fmt.Sprintf("%d", uint64(time.Now().UTC().UnixNano()))
	user, err = db.GetUserByID(ctx, UserID)
	require.ErrorIs(t, err, entities.ErrNotFound)

	require.Nil(t, user)
}

func TestStorage_GetUserByUsername(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	ctx := context.TODO()
	var userName = "admin"

	user, err := db.GetUserByUsername(ctx, userName)
	require.NoError(t, err)

	require.Equal(t, user.ID, "1")

	userName = "John Doe"
	user, err = db.GetUserByUsername(ctx, userName)
	require.ErrorIs(t, err, entities.ErrNotFound)

	require.Nil(t, user)
}

func TestStorage_StoreUser(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	ctx := context.TODO()
	id := fmt.Sprintf("%d", uint64(time.Now().UTC().UnixNano()))
	testUser := &entities.User{
		ID:           id,
		Username:     uuid.New().String(),
		PasswordHash: uuid.New().String(),
		Rights:       []string{"read", "write"},
		Contacts:     map[string]string{"phone": "891111-11", "tg": "@JDoe"},
	}

	err := db.StoreUser(ctx, testUser)
	require.NoError(t, err)

	user, err := db.GetUserByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, testUser, user)
}

func TestStorage_RemoveUser_Success(t *testing.T) {
	db := makeDB(t)
	defer db.Close()
	ctx := context.TODO()

	id := uuid.New().String()
	user := &entities.User{
		ID:           id,
		Username:     uuid.New().String(),
		PasswordHash: uuid.New().String(),
		Rights:       []string{"read", "write"},
		Contacts:     map[string]string{"phone": "1234567890"},
	}

	require.NoError(t, db.StoreUser(ctx, user))

	require.NoError(t, db.RemoveUser(ctx, id))

	usr, err := db.GetUserByID(ctx, id)
	require.ErrorIs(t, err, entities.ErrNotFound)
	require.Nil(t, usr)
}

func TestStorage_RemoveUser_NotFound(t *testing.T) {
	db := makeDB(t)
	defer db.Close()
	ctx := context.TODO()

	err := db.RemoveUser(ctx, "non-existent-id")
	require.ErrorIs(t, err, entities.ErrNotFound)
}
