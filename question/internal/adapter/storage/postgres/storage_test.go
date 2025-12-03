//go:build KVS_TEST_L1

package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgxpool"
	cryptoprocessing "github.com/parta4ok/kvs/question/internal/adapter/generator/crypto_processing"
	"github.com/parta4ok/kvs/question/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/internal/entities/event"
	"github.com/parta4ok/kvs/question/internal/entities/testdata"
	"github.com/stretchr/testify/require"
)

var (
	cstr = os.Getenv("TEST_PG_CONN")
)

var (
	ErrTest = errors.New("test error")
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
	require.Equal(t, os.Truncate(time.Second), rs.Truncate(time.Second))
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

	forbidden, err := session.IsDailySessionLimitReached(ctx, session.GetUserID(), session.GetTopics(), 1)
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
		secondSession.GetTopics(), 1)
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

func TestStorage_GetPassedUserTopics_WithStudentGroup(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	ctx := context.Background()
	generator := cryptoprocessing.NewUint64Generator()

	student1ID := fmt.Sprintf("student1_%d", time.Now().UnixNano())
	student2ID := fmt.Sprintf("student2_%d", time.Now().UnixNano())

	session1, err := entities.NewSession(student1ID, []string{"Базы данных"}, generator, db)
	require.NoError(t, err)

	questions1, err := db.GetQuesions(ctx, []string{"Базы данных"})
	require.NoError(t, err)
	require.NotEmpty(t, questions1)

	questionsMap1 := make(map[string]entities.Question, len(questions1))
	for _, q := range questions1 {
		questionsMap1[q.ID()] = q
	}

	err = session1.SetQuestions(questionsMap1, time.Minute*10)
	require.NoError(t, err)
	require.Equal(t, entities.ActiveState, session1.GetStatus())

	answers1 := make([]*entities.UserAnswer, 0)
	for _, q := range questions1 {
		correctAnswer := findCorrectAnswer(q)
		answer, err := entities.NewUserAnswer(q.ID(), correctAnswer)
		require.NoError(t, err)
		answers1 = append(answers1, answer)
	}

	err = session1.SetUserAnswer(answers1)
	require.NoError(t, err)
	require.Equal(t, entities.CompletedState, session1.GetStatus())

	err = db.StoreSession(ctx, session1)
	require.NoError(t, err)

	session2, err := entities.NewSession(student2ID, []string{"Базы данных", "Базовые типы в Go"}, generator, db)
	require.NoError(t, err)

	questions2, err := db.GetQuesions(ctx, []string{"Базы данных", "Базовые типы в Go"})
	require.NoError(t, err)
	require.NotEmpty(t, questions2)

	questionsMap2 := make(map[string]entities.Question, len(questions2))
	for _, q := range questions2 {
		questionsMap2[q.ID()] = q
	}

	err = session2.SetQuestions(questionsMap2, time.Minute*10)
	require.NoError(t, err)

	answers2 := make([]*entities.UserAnswer, 0)
	for _, q := range questions2 {
		correctAnswer := findCorrectAnswer(q)
		answer, err := entities.NewUserAnswer(q.ID(), correctAnswer)
		require.NoError(t, err)
		answers2 = append(answers2, answer)
	}

	err = session2.SetUserAnswer(answers2)
	require.NoError(t, err)
	require.Equal(t, entities.CompletedState, session2.GetStatus())

	err = db.StoreSession(ctx, session2)
	require.NoError(t, err)

	session3, err := entities.NewSession(student2ID, []string{"Составные типы в Go"}, generator, db)
	require.NoError(t, err)

	questions3, err := db.GetQuesions(ctx, []string{"Составные типы в Go"})
	require.NoError(t, err)
	require.NotEmpty(t, questions3)

	questionsMap3 := make(map[string]entities.Question, len(questions3))
	for _, q := range questions3 {
		questionsMap3[q.ID()] = q
	}

	err = session3.SetQuestions(questionsMap3, time.Minute*10)
	require.NoError(t, err)

	answers3 := make([]*entities.UserAnswer, 0)
	for _, q := range questions3 {
		wrongAnswers := []string{q.Variants()[len(q.Variants())-1]}
		answer, err := entities.NewUserAnswer(q.ID(), wrongAnswers)
		require.NoError(t, err)
		answers3 = append(answers3, answer)
	}

	err = session3.SetUserAnswer(answers3)
	require.NoError(t, err)
	require.Equal(t, entities.CompletedState, session3.GetStatus())

	err = db.StoreSession(ctx, session3)
	require.NoError(t, err)

	students := []string{student1ID, student2ID}
	passedTopics, err := db.GetPassedUserTopics(ctx, students)
	require.NoError(t, err)
	require.NotNil(t, passedTopics)

	require.Contains(t, passedTopics, student1ID)
	student1Topics := passedTopics[student1ID]
	require.Len(t, student1Topics, 1)
	require.Equal(t, "Базы данных", student1Topics[0].Title)

	require.Contains(t, passedTopics, student2ID)
	student2Topics := passedTopics[student2ID]
	require.Len(t, student2Topics, 2)

	student2TopicsMap := make(map[string]bool)
	for _, topic := range student2Topics {
		student2TopicsMap[topic.Title] = true
	}

	require.True(t, student2TopicsMap["Базы данных"], "Student 2 should have passed 'Базы данных'")
	require.True(t, student2TopicsMap["Базовые типы в Go"], "Student 2 should have passed 'Базовые типы в Go'")
	require.False(t, student2TopicsMap["Составные типы в Go"], "Student 2 should NOT have passed 'Составные типы в Go'")
}

func findCorrectAnswer(q entities.Question) []string {
	for _, variant := range q.Variants() {
		userAnswer, err := entities.NewUserAnswer(q.ID(), []string{variant})
		if err != nil {
			continue
		}
		if q.IsAnswerCorrect(userAnswer) {
			return []string{variant}
		}
	}

	variants := q.Variants()
	for i := 0; i < len(variants); i++ {
		for j := i + 1; j < len(variants); j++ {
			userAnswer, err := entities.NewUserAnswer(q.ID(), []string{variants[i], variants[j]})
			if err != nil {
				continue
			}
			if q.IsAnswerCorrect(userAnswer) {
				return []string{variants[i], variants[j]}
			}
		}
	}

	return []string{q.Variants()[0]}
}

func TestStorage_MarkEventAsPublished_Rollback(t *testing.T) {
	t.Parallel()

	e, err := event.NewSessionCompleteEvent([]byte(`{"event_type":"SessionResultEvent", "payload":{"id":1}}`))
	require.NoError(t, err)

	query := `INSERT INTO kvs.outbox (type, payload) VALUES($1, $2) RETURNING id`
	params := []interface{}{event.SessionCompleteEventType.String(), e.Payload()}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, cstr)
	require.NoError(t, err)

	var id int
	err = db.QueryRow(ctx, query, params...).Scan(&id)
	require.NoError(t, err)
	require.Greater(t, id, 0)

	storage := makeDB(t)

	fnWithErr := func(ctx context.Context) error {
		return ErrTest
	}

	err = storage.MarkEventAsPublished(ctx, id, fnWithErr)
	require.ErrorIs(t, err, ErrTest)

	var published bool
	err = db.QueryRow(ctx, `SELECT published FROM kvs.outbox WHERE id = $1`, id).Scan(&published)
	require.NoError(t, err)
	require.False(t, published)
}

func TestStorage_MarkEventAsPublished_Commit(t *testing.T) {
	t.Parallel()

	e, err := event.NewSessionCompleteEvent([]byte(`{"event_type":"SessionResultEvent", "payload":{"id":1}}`))
	require.NoError(t, err)

	query := `INSERT INTO kvs.outbox (type, payload) VALUES($1, $2) RETURNING id`
	params := []interface{}{event.SessionCompleteEventType.String(), e.Payload()}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, cstr)
	require.NoError(t, err)

	var id int
	err = db.QueryRow(ctx, query, params...).Scan(&id)
	require.NoError(t, err)
	require.Greater(t, id, 0)

	storage := makeDB(t)

	fnSuccess := func(ctx context.Context) error {
		return nil
	}

	err = storage.MarkEventAsPublished(ctx, id, fnSuccess)
	require.NoError(t, err)

	var published bool
	err = db.QueryRow(ctx, `SELECT published FROM kvs.outbox WHERE id = $1`, id).Scan(&published)
	require.NoError(t, err)
	require.True(t, published)
}
