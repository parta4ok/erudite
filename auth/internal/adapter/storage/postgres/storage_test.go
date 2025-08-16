//go:build KVS_TEST_L1

package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/parta4ok/kvs/auth/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/stretchr/testify/require"
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

	require.Equal(t, user.Username, "admin@kvs.ru")

	UserID = fmt.Sprintf("%d", uint64(time.Now().UTC().UnixNano()))
	user, err = db.GetUserByID(ctx, UserID)
	require.ErrorIs(t, err, entities.ErrNotFound)

	require.Nil(t, user)
}

func TestStorage_GetUserByUsername(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	ctx := context.TODO()
	var userName = "admin@kvs.ru"

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

func TestStorage_UpdateUser(t *testing.T) {
	type fields struct {
		checkFunc func(t *testing.T, base, updated, changes *entities.User)
	}
	type args struct {
		updatedUser *entities.User
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "update username",
			args: args{
				updatedUser: &entities.User{
					Username: uuid.NewString(),
				},
			},
			fields: fields{
				checkFunc: userNameUpdatedCheck,
			},
		},
		{
			name: "update password hash",
			args: args{
				updatedUser: &entities.User{
					PasswordHash: uuid.NewString(),
				},
			},
			fields: fields{
				checkFunc: passwordHashUpdatedCheck,
			},
		},
		{
			name: "update password hash",
			args: args{
				updatedUser: &entities.User{
					Rights: []string{uuid.NewString()},
				},
			},
			fields: fields{
				checkFunc: rightsUpdatedCheck,
			},
		},
		{
			name: "update contacts",
			args: args{
				updatedUser: &entities.User{
					Contacts: map[string]string{uuid.NewString(): uuid.NewString()},
				},
			},
			fields: fields{
				checkFunc: contactsUpdatedCheck,
			},
		},
		{
			name: "update linkedID",
			args: args{
				updatedUser: &entities.User{
					LinkedID: uuid.NewString(),
				},
			},
			fields: fields{
				checkFunc: linkedIDUpdatedCheck,
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			db := makeDB(it)
			defer db.Close()

			ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
			defer cancel()

			baseUser := &entities.User{
				ID:           uuid.NewString(),
				Username:     uuid.NewString(),
				PasswordHash: uuid.NewString(),
				Rights:       []string{uuid.NewString()},
				Contacts:     map[string]string{uuid.NewString(): uuid.NewString()},
				LinkedID:     uuid.NewString(),
			}

			err := db.StoreUser(ctx, baseUser)
			require.NoError(t, err)

			tc.args.updatedUser.ID = baseUser.ID

			err = db.UpdateUser(ctx, tc.args.updatedUser)
			require.NoError(t, err)

			resUser, err := db.GetUserByID(ctx, baseUser.ID)
			require.NoError(t, err)

			tc.fields.checkFunc(t, baseUser, resUser, tc.args.updatedUser)
		})
	}
}

func userNameUpdatedCheck(t *testing.T, base, updated, changes *entities.User) {
	t.Helper()

	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, changes.Username, updated.Username)
	require.Equal(t, base.PasswordHash, updated.PasswordHash)
	require.Equal(t, base.Rights, updated.Rights)
	require.Equal(t, base.Contacts, updated.Contacts)
	require.Equal(t, base.LinkedID, updated.LinkedID)
}

func passwordHashUpdatedCheck(t *testing.T, base, updated, changes *entities.User) {
	t.Helper()

	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, base.Username, updated.Username)
	require.Equal(t, changes.PasswordHash, updated.PasswordHash)
	require.Equal(t, base.Rights, updated.Rights)
	require.Equal(t, base.Contacts, updated.Contacts)
	require.Equal(t, base.LinkedID, updated.LinkedID)
}

func rightsUpdatedCheck(t *testing.T, base, updated, changes *entities.User) {
	t.Helper()

	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, base.Username, updated.Username)
	require.Equal(t, base.PasswordHash, updated.PasswordHash)
	require.Equal(t, changes.Rights, updated.Rights)
	require.Equal(t, base.Contacts, updated.Contacts)
	require.Equal(t, base.LinkedID, updated.LinkedID)
}

func contactsUpdatedCheck(t *testing.T, base, updated, changes *entities.User) {
	t.Helper()

	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, base.Username, updated.Username)
	require.Equal(t, base.PasswordHash, updated.PasswordHash)
	require.Equal(t, base.Rights, updated.Rights)
	require.Equal(t, changes.Contacts, updated.Contacts)
	require.Equal(t, base.LinkedID, updated.LinkedID)
}

func linkedIDUpdatedCheck(t *testing.T, base, updated, changes *entities.User) {
	t.Helper()

	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, base.Username, updated.Username)
	require.Equal(t, base.PasswordHash, updated.PasswordHash)
	require.Equal(t, base.Rights, updated.Rights)
	require.Equal(t, base.Contacts, updated.Contacts)
	require.Equal(t, changes.LinkedID, updated.LinkedID)
}
