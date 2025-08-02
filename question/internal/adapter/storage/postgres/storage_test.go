//go:build KVS_TEST_L1

package postgres_test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	cryptoprocessing "github.com/parta4ok/kvs/question/internal/adapter/generator/crypto_processing"
	"github.com/parta4ok/kvs/question/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/internal/entities/testdata"
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
	limit := 3

	db := makeDB(t, postgres.WithQuestionsLimit(limit))
	defer db.Close()

	testTopics := []string{"Базы данных"}
	questions, err := db.GetQuesions(context.TODO(), testTopics)
	require.NoError(t, err)

	require.Equal(t, limit, len(questions))
}

func TestStorage_GetSession(t *testing.T) {
	db := makeDB(t, postgres.WithQuestionsLimit(1))
	defer db.Close()

	testTopics := []string{"Составные типы в Go"}
	userID := "12"

	ctrl := gomock.NewController(t)
	defer t.Cleanup(func() {
		ctrl.Finish()
	})

	SessionStorage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, testTopics, cryptoprocessing.NewUint64Generator(),
		SessionStorage)
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

	questionsMap := make(map[string]entities.Question, len(questions))
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

	userID := fmt.Sprintf("%d", time.Now().UTC().Unix())
	topics := []string{"Базы данных"}
	ctx := context.TODO()

	session, err := entities.NewSession(userID, topics, cryptoprocessing.NewUint64Generator(), db)
	require.NoError(t, err)

	forbidden, err := session.IsDailySessionLimitReached(ctx, session.GetUserID(), session.GetTopics())
	require.NoError(t, err)
	require.False(t, forbidden)

	questions, err := db.GetQuesions(ctx, topics)
	require.NoError(t, err)

	questionsMap := make(map[string]entities.Question, len(questions))

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

	secondSession, err := entities.NewSession(userID, topics,
		cryptoprocessing.NewUint64Generator(), db)
	require.NoError(t, err)

	require.Equal(t, entities.InitState, secondSession.GetStatus())

	forbidden, err = secondSession.IsDailySessionLimitReached(ctx, secondSession.GetUserID(),
		secondSession.GetTopics())
	require.NoError(t, err)
	require.True(t, forbidden)
}

func TestStorage_GetAllCompletedUserSessions(t *testing.T) {
	db := makeDB(t, postgres.WithQuestionsLimit(2))
	defer db.Close()

	ctx := context.TODO()
	userID := fmt.Sprintf("usr_%d", time.Now().UnixNano())
	topics := []string{"Базовые типы в Go"}
	ctrl := gomock.NewController(t)
	defer t.Cleanup(ctrl.Finish)

	questions, err := db.GetQuesions(ctx, topics)
	require.NoError(t, err)
	require.NotEmpty(t, questions)

	questionsMap := make(map[string]entities.Question, len(questions))
	for _, q := range questions {
		questionsMap[q.ID()] = q
	}

	sessionOld, err := entities.NewSession(userID, topics, cryptoprocessing.NewUint64Generator(), db)
	require.NoError(t, err)
	err = sessionOld.SetQuestions(questionsMap, time.Minute*10)
	require.NoError(t, err)
	require.NoError(t, sessionOld.SetUserAnswer([]*entities.UserAnswer{
		mustAnswer(t, questions[0]),
		mustAnswer(t, questions[1]),
	}))
	require.Equal(t, entities.CompletedState, sessionOld.GetStatus())
	require.NoError(t, db.StoreSession(ctx, sessionOld))

	sessionNew, err := entities.NewSession(userID, topics, cryptoprocessing.NewUint64Generator(), db)
	require.NoError(t, err)
	err = sessionNew.SetQuestions(questionsMap, time.Minute*10)
	require.NoError(t, err)
	require.NoError(t, sessionNew.SetUserAnswer([]*entities.UserAnswer{
		mustAnswer(t, questions[0]),
		mustAnswer(t, questions[1]),
	}))
	require.Equal(t, entities.CompletedState, sessionNew.GetStatus())
	require.NoError(t, db.StoreSession(ctx, sessionNew))

	sessions, err := db.GetAllCompletedUserSessions(ctx, userID)
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	require.True(t, sessions[0].GetSesionID() == sessionNew.GetSesionID(), "newest session first")
	require.True(t, sessions[1].GetSesionID() == sessionOld.GetSesionID(), "oldest session second")

	require.Equal(t, sessionNew.GetTopics(), sessions[0].GetTopics())
	require.Equal(t, sessionOld.GetTopics(), sessions[1].GetTopics())

	newQs, err := sessionNew.GetQuestions()
	require.NoError(t, err)
	oldQs, err := sessionOld.GetQuestions()
	require.NoError(t, err)
	fetchedNewQs, _ := sessions[0].GetQuestions()
	fetchedOldQs, _ := sessions[1].GetQuestions()
	require.ElementsMatch(t, newQs, fetchedNewQs)
	require.ElementsMatch(t, oldQs, fetchedOldQs)
	require.Equal(t, entities.CompletedState, sessions[0].GetStatus())
	require.Equal(t, entities.CompletedState, sessions[1].GetStatus())
}

func mustAnswer(t *testing.T, q entities.Question) *entities.UserAnswer {
	t.Helper()

	answer, err := entities.NewUserAnswer(q.ID(), []string{q.Variants()[0]})
	require.NoError(t, err)
	return answer
}
