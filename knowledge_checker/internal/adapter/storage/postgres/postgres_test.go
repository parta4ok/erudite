//go:build KVS_TEST_L1

package postgres_test

import (
	"context"
	"testing"

	"github.com/parta4ok/kvs/knowledge_checker/internal/adapter/storage/postgres"
	envsettings "github.com/parta4ok/kvs/knowledge_checker/pkg/env_settings"
	"github.com/stretchr/testify/require"
)

var (
	cstr = envsettings.Getenv("KVS_TEST_PG_CONN_STR")
)

func makeDB(t *testing.T) *postgres.Storage {
	t.Helper()

	db, err := postgres.NewStorage(cstr)
	require.NoError(t, err)
	require.NotNil(t, db)

	return db
}

func TestStorage_GetTopics(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	ctx := context.TODO()
	topics, err := db.GetTopics(ctx)
	require.NoError(t, err)
	require.NotNil(t, topics)
}
