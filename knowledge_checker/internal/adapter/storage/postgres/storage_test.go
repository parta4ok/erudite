//go:build KVS_TEST_L1

package postgres_test

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/parta4ok/kvs/knowledge_checker/internal/adapter/generator"
	"github.com/parta4ok/kvs/knowledge_checker/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities/testdata"
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

func TestStorage_GetSession(t *testing.T) {
	db := makeDB(t, postgres.WithQuestionsLimit(1))
	defer db.Close()

	testTopics := []string{"Составные типы в Go"}
	userID := uint64(12)

	ctrl := gomock.NewController(t)
	defer t.Cleanup(func() {
		ctrl.Finish()
	})

	SessionStorage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, testTopics, generator.NewUint64Generator(), SessionStorage)
	require.NoError(t, err)
	require.Equal(t, session.GetStatus(), entities.InitState)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	err = db.StoreSession(ctx, session)
	require.NoError(t, err)

	restoredInitSession, err := db.GetSessionBySessionID(ctx, session.GetSesionID())
	require.Equal(t, restoredInitSession.GetStatus(), entities.InitState)
	compareSession(t, session, restoredInitSession)

	questions, err := db.GetQuesions(context.TODO(), testTopics)
	require.NoError(t, err)

	questionsMap := make(map[uint64]entities.Question, len(questions))
	for _, q := range questions {
		questionsMap[q.ID()] = q
	}

	err = session.SetQuestions(questionsMap, time.Minute*10)
	require.NoError(t, err)
	require.Equal(t, session.GetStatus(), entities.ActiveState)

	err = db.StoreSession(ctx, session)
	require.NoError(t, err)

	restoredActiveSession, err := db.GetSessionBySessionID(ctx, session.GetSesionID())
	require.Equal(t, restoredActiveSession.GetStatus(), entities.ActiveState)
	compareSession(t, session, restoredActiveSession)

	userAnswers := make([]*entities.UserAnswer, 0, len(questions))
	for _, q := range questions {
		answer, err := entities.NewUserAnswer(q.ID(), []string{q.Variants()[1]})
		require.NoError(t, err)
		userAnswers = append(userAnswers, answer)
	}

	err = restoredActiveSession.SetUserAnswer(userAnswers)
	require.NoError(t, err)
	require.Equal(t, entities.CompletedState, restoredActiveSession.GetStatus())

	err = db.StoreSession(ctx, restoredActiveSession)
	require.NoError(t, err)

	restoredCompletedSession, err := db.GetSessionBySessionID(ctx, session.GetSesionID())
	require.NoError(t, err)
	compareSession(t, restoredActiveSession, restoredCompletedSession)
}

func compareSession(t *testing.T, originalSession, recoveredSession *entities.Session) {
	t.Helper()

	oq, oErr := originalSession.GetQuestions()
	rq, rErr := recoveredSession.GetQuestions()

	sort.Slice(oq, func(i, j int) bool {
		return oq[i].ID() > oq[j].ID()
	})

	sort.Slice(rq, func(i, j int) bool {
		return rq[i].ID() > rq[j].ID()
	})

	require.Equal(t, oq, rq)
	if oErr != nil {
		require.Contains(t, oErr.Error(), rErr.Error())
	}

	require.Equal(t, originalSession.GetSesionID(), recoveredSession.GetSesionID())
	ol, oErr := originalSession.GetSessionDurationLimit()
	rl, rErr := originalSession.GetSessionDurationLimit()
	require.Equal(t, ol, rl)
	if oErr != nil {
		require.Contains(t, oErr.Error(), rErr.Error())
	}
	or, oErr := originalSession.GetSessionResult()
	rr, rErr := recoveredSession.GetSessionResult()
	require.Equal(t, or, rr)
	if oErr != nil {
		require.Contains(t, oErr.Error(), rErr.Error())
	}
	os, oErr := originalSession.GetStartedAt()
	rs, rErr := recoveredSession.GetStartedAt()
	require.Equal(t, os, rs)
	if oErr != nil {
		require.Contains(t, oErr.Error(), rErr.Error())
	}
	require.Equal(t, originalSession.GetStatus(), recoveredSession.GetStatus())
	require.Equal(t, originalSession.GetTopics(), recoveredSession.GetTopics())
	oa, oErr := originalSession.GetUserAnswers()
	ra, rErr := recoveredSession.GetUserAnswers()
	require.Equal(t, oa, ra)
	if oErr != nil {
		require.Contains(t, oErr.Error(), rErr.Error())
	}
	require.Equal(t, originalSession.GetUserID(), recoveredSession.GetUserID())
	oe, oErr := originalSession.IsExpired()
	re, rErr := recoveredSession.IsExpired()
	require.Equal(t, oe, re)
	if oErr != nil {
		require.Contains(t, oErr.Error(), rErr.Error())
	}
}

func TestStorage_IsDailySessionLimitReached(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	userID := uint64(time.Now().UTC().Unix())
	topics := []string{"Базы данных"}
	ctx := context.TODO()

	session, err := entities.NewSession(userID, topics, generator.NewUint64Generator(), db)
	require.NoError(t, err)

	forbidden, err := session.IsDailySessionLimitReached(ctx, session.GetUserID(), session.GetTopics())
	require.NoError(t, err)
	require.False(t, forbidden)

	questions, err := db.GetQuesions(ctx, topics)
	require.NoError(t, err)

	questionsMap := make(map[uint64]entities.Question, len(questions))

	for _, q := range questions {
		questionsMap[q.ID()] = q
	}

	session.SetQuestions(questionsMap, time.Minute*time.Duration(len(questions)))
	answers := make([]*entities.UserAnswer, 0, len(questions))

	for qid, q := range questionsMap {
		answer, err := entities.NewUserAnswer(qid, q.Variants()[:1])
		require.NoError(t, err)

		answers = append(answers, answer)
	}

	session.SetUserAnswer(answers)
	require.Equal(t, entities.CompletedState, session.GetStatus())

	err = db.StoreSession(ctx, session)
	require.NoError(t, err)

	secondSession, err := entities.NewSession(userID, topics, generator.NewUint64Generator(), db)
	require.NoError(t, err)

	require.Equal(t, entities.InitState, secondSession.GetStatus())

	forbidden, err = secondSession.IsDailySessionLimitReached(ctx, secondSession.GetUserID(), secondSession.GetTopics())
	require.NoError(t, err)
	require.True(t, forbidden)
}
