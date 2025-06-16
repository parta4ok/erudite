//go:build KVS_TEST_L1

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/parta4ok/kvs/knowledge_checker/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
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

func TestStorage_GetTopics(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	ctx := context.TODO()
	topics, err := db.GetTopics(ctx)
	require.NoError(t, err)
	require.NotNil(t, topics)
}

func TestStorage_GetQuestions(t *testing.T) {
	db := makeDB(t, postgres.WithQuestionsLimit(3))
	defer db.Close()

	testTopics := []string{"Базы данных"}
	questions, err := db.GetQuesions(context.TODO(), testTopics)
	require.NoError(t, err)

	typeMap := make(map[entities.QuestionType]struct{}, 0)
	for _, q := range questions {
		require.Equal(t, testTopics[0], q.Topic())
		typeMap[q.Type()] = struct{}{}
	}

	require.Equal(t, 3*len(typeMap), len(questions))
}
