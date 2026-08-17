package postgres

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	cryptoprocessing "github.com/parta4ok/kvs/question/internal/adapter/generator/crypto_processing"
	"github.com/parta4ok/kvs/question/internal/cases"
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/internal/entities/event"
	_ "github.com/parta4ok/kvs/question/internal/port/http/public"
	"github.com/parta4ok/kvs/question/pkg/dto"
	natsDTO "github.com/parta4ok/kvs/toolkit/pkg/broker/nats"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
)

var (
	_ cases.Storage           = (*Storage)(nil)
	_ entities.SessionStorage = (*Storage)(nil)
)

const (
	DefaultTopicLimit = 10
)

type Storage struct {
	questionsLimits int
	db              *pgxpool.Pool
	once            sync.Once
	cancel          context.CancelFunc
	questionFactory *entities.QuestionFactory
}

type StorageOption func(s *Storage)

func WithQuestionsLimit(questionsLimit int) StorageOption {
	return func(s *Storage) {
		s.questionsLimits = questionsLimit
	}
}

func (s *Storage) setOptions(opts ...StorageOption) {
	for _, opt := range opts {
		opt(s)
	}
}

func NewStorage(connectionString string, opts ...StorageOption) (*Storage, error) {
	if strings.TrimSpace(connectionString) == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "connection string is empty")
	}
	st := &Storage{
		questionsLimits: DefaultTopicLimit,
	}

	st.setOptions(opts...)

	ctx, cancel := context.WithCancel(context.Background())
	st.cancel = cancel

	db, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInvalidParam, "connection creating error: %v", err.Error())
	}
	st.db = db
	st.questionFactory = &entities.QuestionFactory{}

	return st, nil
}

func (s *Storage) Close() {
	s.once.Do(func() {
		s.cancel()
		s.db.Close()
	})
}

func (s *Storage) GetTopics(ctx context.Context) ([]string, error) {
	slog.Info("GetTopics started")
	ctx, span, cancel := tracer.Start(ctx, "GetTopicsPostgresSpan")
	defer cancel()

	query := `SELECT t.name FROM kvs.topics t`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "getting topic names failure: %v", err)
		slog.Error("getting topic names failure", "error", err.Error())
		span.SetError(err)
		return nil, err
	}
	defer rows.Close()

	topics := make([]string, 0)

	for rows.Next() {
		var topicName string
		if err := rows.Scan(&topicName); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "scan topic name failure: %v", err)
			slog.Error("scan topic name failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}
		topics = append(topics, topicName)
	}

	if err := rows.Err(); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "rows err: %v", err)
		slog.Error("rows err", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("GetTopics completed")
	return topics, nil
}

//nolint:funlen //ok
func (s *Storage) GetQuestions(ctx context.Context, topics []string) (
	[]entities.Question, error) {
	slog.Info("GetQuestions started")
	ctx, span, cancel := tracer.Start(ctx, "GetQuestionsPostgresSpan")
	defer cancel()

	if err := s.checkTopics(ctx, topics); err != nil {
		return nil, err
	}

	params := []interface{}{topics, s.questionsLimits}

	query := `
	SELECT
    question_id, question_type, topic, subject, variants, correct_answers
	FROM (
		SELECT q.*, qt.name AS question_type, t.name AS topic,
		ROW_NUMBER() OVER (PARTITION BY t.topic_id ORDER BY random()) AS rn
    	FROM kvs.questions q
    	JOIN kvs.topics t ON q.topic_id = t.topic_id
    	JOIN kvs.question_types qt ON q.question_type_id = qt.id
    	WHERE t.name = ANY($1)
	) random_questions
	WHERE rn <= $2;`

	rows, errDB := s.db.Query(ctx, query, params...)
	if errDB != nil {
		err := errors.Wrapf(entities.ErrInternal, "get questions from db failure: %v", errDB)
		slog.Error("get questions from db failure", "error", err.Error())
		span.SetError(err)
		return nil, err
	}
	defer rows.Close()

	questions, err := s.processingQuestionsRows(ctx, rows)
	if err != nil {
		err := errors.Wrap(err, "processingQuestionsRows")
		slog.Error("processingQuestionsRows", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("GetQuestions completed")
	return questions, nil
}

//nolint:funlen,gosec //ok
func (s *Storage) StoreSession(ctx context.Context, session *entities.Session) error {
	slog.Info("StoreSession started")
	ctx, span, cancel := tracer.Start(ctx, "StoreSessionPostgresSpan")
	defer cancel()

	userID := session.GetUserID()
	sessionID := session.GetSessionID()
	sessionStatus := session.GetStatus()
	topics := session.GetTopics()

	parameters := make([]interface{}, 0)
	parameters = append(parameters, sessionID, userID, sessionStatus, topics)

	query := `INSERT INTO kvs.sessions (session_id, user_id, state, topics`

	switch sessionStatus {
	case entities.InitState:
		query += s.makeInitStateSessionQuery()
	case entities.ActiveState:
		query += s.makeActiveStateSessionQuery()

		questionsIDs, err := s.getQuestionsIDs(session)
		if err != nil {
			slog.Error("get questions ids failure", "error", err.Error())
			span.SetError(err)
			return err
		}

		startedAt, err := session.GetStartedAt()
		if err != nil {
			err := errors.Wrap(err, "session GetStartedAt failure")
			slog.Error("session GetStartedAt failure", "error", err.Error())
			span.SetError(err)
			return err
		}

		duration, err := session.GetSessionDurationLimit()
		if err != nil {
			err := errors.Wrap(err, "session GetSessionDurationLimit failure")
			slog.Error("session GetSessionDurationLimit failure", "error", err.Error())
			span.SetError(err)
			return err
		}

		parameters = append(parameters, questionsIDs, startedAt, duration)

	case entities.CompletedState:
		query += s.makeCompletedStateSessionQuery()

		questionsIDs, err := s.getQuestionsIDs(session)
		if err != nil {
			err := errors.Wrap(err, "getQuestionsIDs failure")
			slog.Error("getQuestionsIDs failure", "error", err.Error())
			span.SetError(err)
			return err
		}

		startedAt, err := session.GetStartedAt()
		if err != nil {
			err := errors.Wrap(err, "session GetStartedAt failure")
			slog.Error("session GetStartedAt failure", "error", err.Error())
			span.SetError(err)
			return err
		}

		userAnswers, err := session.GetUserAnswers()
		if err != nil {
			err := errors.Wrap(err, "session GetUserAnswers failure")
			slog.Error("session GetUserAnswers failure", "error", err.Error())
			span.SetError(err)
			return err
		}

		userAnswersDTO := make([]dto.UserAnswerDTO, 0, len(userAnswers))
		for _, answer := range userAnswers {
			userAnswersDTO = append(userAnswersDTO, dto.UserAnswerDTO{
				QuestionID: answer.GetQuestionID(),
				Answers:    answer.GetSelections(),
			})
		}

		userAnswersList := dto.UserAnswersListDTO{AnswersList: userAnswersDTO}
		answersListJSON, err := json.Marshal(userAnswersList)
		if err != nil {
			err := errors.Wrapf(entities.ErrInternal, "marshalling failure: %v", err)
			slog.Error("marshalling failure", "error", err.Error())
			span.SetError(err)
			return err
		}

		isExpired, err := session.IsExpired()
		if err != nil {
			err := errors.Wrap(err, "session IsExpired failure")
			slog.Error("session IsExpired failure", "error", err.Error())
			span.SetError(err)
			return err
		}

		sesseionResult, err := session.GetSessionResult()
		if err != nil {
			err := errors.Wrap(err, "session GetSessionResult failure")
			slog.Error("session GetSessionResult failure", "error", err.Error())
			span.SetError(err)
			return err
		}

		parameters = append(parameters, questionsIDs, startedAt, answersListJSON, isExpired,
			sesseionResult.IsSuccess, sesseionResult.Grade)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal,
			"create new transaction finished with failure: %v", err)
		slog.Error("create new transaction finished with failure", "error", err.Error())
		span.SetError(err)
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx) //nolint:errcheck //ok
		}
	}()

	_, err = tx.Exec(ctx, query, parameters...)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "store session finished with failure: %v", err)
		slog.Error("store session finished with failure", "error", err.Error())
		span.SetError(err)
		return err
	}

	_, err = s.storeSessionResultEvent(ctx, tx, session)
	if err != nil {
		slog.Error("store session result event failure", "error", err.Error())
		span.SetError(err)
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "commit store session finished with failure: %v", err)
		slog.Error("commit store session finished with failure", "error", err.Error())
		span.SetError(err)
		return err
	}

	slog.Info("StoreSession completed")
	return nil
}

func (s *Storage) GetSessionBySessionID(ctx context.Context, sessionID string) (*entities.Session,
	error) {
	slog.Info("GetSessionBySessionID started")
	ctx, span, cancel := tracer.Start(ctx, "GetSessionBySessionIDPostgresSpan")
	defer cancel()

	query := `
	SELECT s.user_id, s.state, s.topics, s.questions, s.answers, s.created_at,
	s.duration_limit, s.is_expired
	FROM kvs.sessions s
	WHERE s.session_id = $1
	ORDER BY s.updated_at DESC
	LIMIT 1;`
	sessionParameters := []interface{}{sessionID}

	row := s.db.QueryRow(ctx, query, sessionParameters...)
	var (
		userID         string
		stateName      string
		topics         []string
		questionsIDs   []string
		answersRaw     []byte
		createdAt      *time.Time
		duration_limit uint64
		isExpired      *bool
	)

	err := row.Scan(&userID, &stateName, &topics, &questionsIDs, &answersRaw,
		&createdAt, &duration_limit, &isExpired)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = errors.Wrapf(entities.ErrNotFound, "not found session with requested id: %v", err)
			slog.Error("not found session with requested id", "error", err.Error())
			span.SetError(err)
			return nil, err
		}
		err = errors.Wrapf(entities.ErrInternal, "scan session data failure: %v", err)
		slog.Error("scan session data failure", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("GetSessionBySessionID completed")
	return s.recoverSession(ctx, sessionID, stateName, userID, topics, questionsIDs,
		duration_limit, answersRaw, createdAt, isExpired)
}

//nolint:funlen //ok
func (s *Storage) recoverSession(ctx context.Context, sessionID string, stateName string,
	userID string, topics []string, questionsIDs []string, duration_limit uint64, answersRaw []byte,
	createdAt *time.Time, isExpired *bool) (*entities.Session, error) {
	slog.Info("recoverSession started")
	ctx, span, cancel := tracer.Start(ctx, "recoverSessionPostgresSpan")
	defer cancel()

	switch stateName {
	case entities.InitState:
		initSession, err := entities.NewSession(userID, topics,
			cryptoprocessing.NewUint64Generator(), s, entities.WithSessionID(sessionID),
			entities.WithNilState())
		if err != nil {
			err = errors.Wrap(err, "creating new session with sessionID option failure")
			slog.Error("creating new session with sessionID option failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}

		state := entities.NewInitSessionState(initSession, s)
		initSession.ChangeState(state)

		slog.Info("recoverSession completed")
		return initSession, nil

	case entities.ActiveState:
		activeSession, err := entities.NewSession(userID, topics,
			cryptoprocessing.NewUint64Generator(), s, entities.WithSessionID(sessionID),
			entities.WithNilState())
		if err != nil {
			err = errors.Wrap(err, "creating new session with sessionID option failure")
			slog.Error("creating new session with sessionID option failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}

		questions, err := s.getQuestionsByID(ctx, questionsIDs)
		if err != nil {
			err = errors.Wrap(err, "getQuestionsByID failure")
			slog.Error("getQuestionsByID failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}

		questionsMap := make(map[string]entities.Question, len(questions))
		for _, question := range questions {
			questionsMap[question.ID()] = question
		}
		state := entities.NewActiveSessionState(questionsMap, activeSession,
			time.Duration(duration_limit), entities.WithStartedAt(*createdAt)) //nolint:gosec,lll // ok
		activeSession.ChangeState(state)

		slog.Info("recoverSession completed")
		return activeSession, nil

	case entities.CompletedState:
		completedSession, err := entities.NewSession(userID, topics,
			cryptoprocessing.NewUint64Generator(), s, entities.WithSessionID(sessionID),
			entities.WithNilState())
		if err != nil {
			err = errors.Wrap(err, "creating new session with sessionID option failure")
			slog.Error("creating new session with sessionID option failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}

		questions, err := s.getQuestionsByID(ctx, questionsIDs)
		if err != nil {
			err = errors.Wrap(err, "getQuestionsByID failure")
			slog.Error("getQuestionsByID failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}

		questionsMap := make(map[string]entities.Question, len(questions))
		for _, question := range questions {
			questionsMap[question.ID()] = question
		}

		var answersListDTO dto.UserAnswersListDTO
		if err := json.Unmarshal(answersRaw, &answersListDTO); err != nil {
			err = errors.Wrapf(entities.ErrInternal, "unmarshaling failure: %v", err)
			slog.Error("unmarshaling failure", "error", err.Error())
			return nil, err
		}

		answers := make([]*entities.UserAnswer, 0, len(answersListDTO.AnswersList))
		for _, answerDTO := range answersListDTO.AnswersList {
			answer, err := entities.NewUserAnswer(answerDTO.QuestionID, answerDTO.Answers)
			if err != nil {
				err = errors.Wrap(err, "creating user answer failure")
				slog.Error("creating user answer failure", "error", err.Error())
				span.SetError(err)
				return nil, err
			}
			answers = append(answers, answer)
		}

		state := entities.NewCompletedSessionState(questionsMap, completedSession,
			answers, *createdAt, *isExpired)
		completedSession.ChangeState(state)

		slog.Info("recoverSession completed")
		return completedSession, nil
	}

	err := errors.Wrapf(entities.ErrInternal, "unknown session state: %s", stateName)
	slog.Error("unknown session state", "error", err.Error())
	span.SetError(err)
	return nil, err
}

func (s *Storage) getQuestionsByID(ctx context.Context, questionsIDs []string) (
	[]entities.Question, error) {
	slog.Info("getQuestionsByID strarted")
	ctx, span, cancel := tracer.Start(ctx, "getQuestionsByIDPostgresSpan")
	defer cancel()

	query := `
	SELECT q.question_id, qt.name AS question_type_name, t.name AS topic_name, q.subject,
	q.variants, q.correct_answers
	FROM
    kvs.questions q
	JOIN kvs.question_types qt ON q.question_type_id = qt.id
	JOIN kvs.topics t ON q.topic_id = t.topic_id
	WHERE
    q.question_id =  ANY($1::BIGINT[])
	ORDER BY
    q.question_id;
	`
	params := []interface{}{questionsIDs}

	rows, errDB := s.db.Query(ctx, query, params...)
	if errDB != nil {
		err := errors.Wrapf(entities.ErrInternal,
			"get questions from db failure: %s", errDB.Error())
		slog.Error("get questions from db failure", "error", err.Error())
		span.SetError(err)
		return nil, err
	}
	defer rows.Close()

	questions, err := s.processingQuestionsRows(ctx, rows)
	if err != nil {
		err := errors.Wrap(err, "processingQuestionsRows failure")
		slog.Error("processingQuestionsRows failure", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("getQuestionsByID completed")
	return questions, nil
}

func (s *Storage) processingQuestionsRows(_ context.Context, rows pgx.Rows) ([]entities.Question,
	error) {
	slog.Info("processingQuestionsRows started")

	questions := make([]entities.Question, 0)

	for rows.Next() {
		var (
			questionID    string
			questionType  string
			topic         string
			subject       string
			variants      []string
			correctAnswer []string
		)

		err := rows.Scan(&questionID, &questionType, &topic, &subject, &variants, &correctAnswer)
		if err != nil {
			err := errors.Wrapf(entities.ErrInternal, "scan questions data failure: %v", err)
			slog.Error("scan questions data failure", "error", err.Error())
			return nil, err
		}

		var qt entities.QuestionType
		switch questionType {
		case "single selection":
			qt = entities.SingleSelection
		case "multi selection":
			qt = entities.MultiSelection
		case "true or false":
			qt = entities.TrueOrFalse
		}
		question, err := s.questionFactory.NewQuestion(questionID, qt, topic, subject, variants,
			correctAnswer)
		if err != nil {
			err := errors.Wrapf(entities.ErrInternal, "creating questions failure")
			slog.Error("creating questions failure", "error", err.Error())
			return nil, err
		}

		questions = append(questions, question)
	}

	slog.Info("processingQuestionsRows completed")
	return questions, nil
}

func (s *Storage) makeInitStateSessionQuery() string {
	return `
		) values ($1, $2, $3, $4);
	`
}

func (s *Storage) makeActiveStateSessionQuery() string {
	return `
		, questions, created_at, duration_limit) values ($1, $2, $3, $4, $5, $6, $7);
	`
}

func (s *Storage) makeCompletedStateSessionQuery() string {
	return `
		, questions, created_at, answers, is_expired, is_passed, comment) values ($1, $2, $3, $4, $5,
		$6, $7, $8, $9, $10);
	`
}

func (s *Storage) getQuestionsIDs(session *entities.Session) ([]string, error) {
	questions, err := session.GetQuestions()
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal,
			"get questions from session state: %v", err)
		slog.Error("get questions from session state failure", "error", err.Error())
		return nil, err
	}

	questionsIDs := make([]string, 0, len(questions))
	for _, q := range questions {
		questionsIDs = append(questionsIDs, q.ID())
	}

	return questionsIDs, nil
}

func (s *Storage) IsDailySessionLimitReached(ctx context.Context, userID string,
	topics []string, dailyLimit int) (bool, error) {
	slog.Info("IsDailySessionLimitReached started")
	ctx, span, cancel := tracer.Start(ctx, "IsDailySessionLimitReachedPostgresSpan")
	defer cancel()

	query := `
SELECT
    s.user_id,
    s.topics,
    COUNT(*) AS completed_sessions_today
FROM
    kvs.sessions s
WHERE
    s.user_id = $1
    AND
    s.state = 'completed state'
    AND
    s.updated_at::date >= CURRENT_DATE
    AND
    $2::text[] && s.topics
GROUP BY
    s.user_id,
    s.topics
ORDER BY
    completed_sessions_today DESC,
    user_id ASC;
	`
	parameters := []interface{}{userID, topics}

	row := s.db.QueryRow(ctx, query, parameters...)
	var (
		uid string
		t   []string
		cnt int
	)
	if err := row.Scan(&uid, &t, &cnt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Info("IsDailySessionLimitReached completed")
			return false, nil
		}
		err := errors.Wrapf(entities.ErrInternal, "scan failure: %v", err)
		slog.Error("scan failure", "error", err.Error())
		span.SetError(err)
		return false, err
	}

	slog.Info("IsDailySessionLimitReached completed")
	if cnt >= dailyLimit {
		return true, nil
	}

	return false, nil
}

//nolint:funlen //ok
func (s *Storage) GetAllCompletedUserSessions(ctx context.Context, userID string) (
	[]*entities.Session, error) {
	slog.Info("GetAllCompletedUserSessions started")
	ctx, span, cancel := tracer.Start(ctx,
		"GetAllCompletedUserSessionsPostgresSpan")
	defer cancel()

	args := []interface{}{userID}

	query := `
	SELECT
    	session_id,
    	user_id,
    	state,
    	topics,
    	questions,
    	answers,
    	is_expired,
    	is_passed,
    	comment,
		created_at,
    	updated_at
	FROM kvs.sessions
	WHERE user_id = $1
  	AND state = 'completed state'
	ORDER BY updated_at DESC;
	`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err := errors.Wrapf(entities.ErrNotFound, "not found completed sessions with userID=%s", userID)
			slog.Warn(err.Error())
			return nil, err
		}
		err := errors.Wrapf(entities.ErrInternal, "search completed user sessions failure: %v", err)
		slog.Warn(err.Error())
		span.SetError(err)
		return nil, err
	}

	userSessions := make([]*entities.Session, 0)

	for rows.Next() {
		var (
			sessionID    string
			userID       string
			stateName    string
			topics       []string
			questionsIDs []string
			answersRaw   []byte
			isExpired    *bool
			isPassed     bool
			comment      string
			createdAt    *time.Time
			updatedAt    *time.Time
		)

		if err := rows.Scan(&sessionID, &userID, &stateName, &topics, &questionsIDs, &answersRaw,
			&isExpired, &isPassed, &comment, &createdAt, &updatedAt); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "scan session data failure: %v", err)
			slog.Error("scan session data failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}

		completedSession, err := entities.NewSession(userID, topics,
			cryptoprocessing.NewUint64Generator(), s, entities.WithSessionID(sessionID),
			entities.WithNilState())
		if err != nil {
			err = errors.Wrap(err, "creating new session with sessionID option failure")
			slog.Error("creating new session with sessionID option failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}

		questions, err := s.getQuestionsByID(ctx, questionsIDs)
		if err != nil {
			err = errors.Wrap(err, "getQuestionsByID failure")
			slog.Error("getQuestionsByID failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}

		questionsMap := make(map[string]entities.Question, len(questions))
		for _, question := range questions {
			questionsMap[question.ID()] = question
		}

		var answersListDTO dto.UserAnswersListDTO
		if err := json.Unmarshal(answersRaw, &answersListDTO); err != nil {
			err = errors.Wrapf(entities.ErrInternal, "unmarshaling failure: %v", err)
			slog.Error("unmarshaling failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}

		answers := make([]*entities.UserAnswer, 0, len(answersListDTO.AnswersList))
		for _, answerDTO := range answersListDTO.AnswersList {
			answer, err := entities.NewUserAnswer(answerDTO.QuestionID, answerDTO.Answers)
			if err != nil {
				err = errors.Wrap(err, "creating user answer failure")
				slog.Error("creating user answer failure", "error", err.Error())
				return nil, err
			}
			answers = append(answers, answer)
		}

		state := entities.NewCompletedSessionState(questionsMap, completedSession,
			answers, *createdAt, *isExpired)
		completedSession.ChangeState(state)
		userSessions = append(userSessions, completedSession)
	}

	if err := rows.Err(); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "rows err: %v", err)
		slog.Error("rows err", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("GetAllCompletedUserSessions completed")
	return userSessions, nil
}

func (s *Storage) checkTopics(ctx context.Context, requestdTopics []string) error {
	slog.Info("checkTopics started")
	ctx, span, cancel := tracer.Start(ctx, "checkTopicsPostgresSpan")
	defer cancel()

	existsTopic, err := s.GetTopics(ctx)
	if err != nil {
		return err
	}

	for _, requestedTopic := range requestdTopics {
		if !slices.Contains(existsTopic, requestedTopic) {
			err := errors.Wrapf(entities.ErrNotFound,
				"requested topic: %s not included in existings topics", requestedTopic)
			slog.Error("requested topic not included in existing topics",
				"error", err.Error(), "topic", requestedTopic)
			span.SetError(err)
			return err
		}
	}
	return nil
}

//nolint:funlen //ok
func (s *Storage) GetPassedUserTopics(ctx context.Context, studentds []string) (
	map[string][]*entities.Topic, error) {
	slog.Info("GetPassedUserTopics started")
	ctx, span, cancel := tracer.Start(ctx, "GetPassedUserTopicsPostgresSpan")
	defer cancel()

	passedTopics := make(map[string][]*entities.Topic, 0)

	topicQuery := `SELECT topic_id, name FROM kvs.topics`
	rows, err := s.db.Query(ctx, topicQuery)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err := errors.Wrapf(entities.ErrNotFound, "not found existed topics")
			slog.Warn(err.Error())
			return nil, err
		}
		err := errors.Wrapf(entities.ErrInternal, "not found existed topics failure: %v", err)
		slog.Warn(err.Error())
		span.SetError(err)
		return nil, err
	}

	allTopicsMap := make(map[string]int, 0)
	for rows.Next() {
		var (
			id    int
			title string
		)

		rows.Scan(&id, &title) //nolint:errcheck,gosec //ok
		allTopicsMap[title] = id
	}

	if rows.Err() != nil {
		err := errors.Wrapf(entities.ErrInternal, "rows with titles had failure: %v", err)
		slog.Warn(err.Error())
		span.SetError(err)
		return nil, err
	}

	args := []interface{}{studentds}

	query := `SELECT user_id, topics from kvs.sessions WHERE state = 'completed state' AND
 	is_passed = TRUE AND user_id = ANY($1)`

	rows, err = s.db.Query(ctx, query, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err := errors.Wrapf(entities.ErrNotFound, "not found passed topics for users")
			slog.Warn(err.Error())
			return nil, err
		}
		err := errors.Wrapf(entities.ErrInternal, "search passed users topics failure: %v", err)
		slog.Warn(err.Error())
		span.SetError(err)
		return nil, err
	}

	for rows.Next() {
		topicsMap := make(map[string]struct{}, 0)

		var (
			userID string
			topics []string
		)

		rows.Scan(&userID, &topics) //nolint:errcheck,gosec //ok
		for _, topic := range topics {
			topicsMap[topic] = struct{}{}
		}

		for topic := range topicsMap {
			passedTopics[userID] = append(passedTopics[userID], &entities.Topic{
				ID:    allTopicsMap[topic],
				Title: topic,
			})
		}
	}
	if rows.Err() != nil {
		err := errors.Wrapf(entities.ErrInternal, "rows with passed titles had failure: %v", err)
		slog.Warn(err.Error())
		span.SetError(err)
		return nil, err
	}

	return passedTopics, nil
}

func (s *Storage) storeSessionResultEvent(ctx context.Context, tx pgx.Tx, session *entities.Session,
) (pgconn.CommandTag, error) {
	slog.Info("storeSessionResultEvent started", "session_state", session.GetStatus())

	if session.GetStatus() != entities.CompletedState {
		return pgconn.CommandTag{}, nil
	}

	sessionResult, err := session.GetSessionResult()
	if err != nil {
		slog.Error("session.GetSessionResult", "error", err.Error())
		return pgconn.CommandTag{}, err
	}

	payloadData, err := json.Marshal(natsDTO.SessionResultDTO{
		UserID:      session.GetUserID(),
		Topics:      session.GetTopics(),
		Questions:   sessionResult.Questions,
		UserAnswers: sessionResult.UserAnswers,
		IsExpire:    sessionResult.IsExpire,
		IsSuccess:   sessionResult.IsSuccess,
		Grade:       sessionResult.Grade,
	})
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "marshalling failed with err: %v", err)
		slog.Error("marshalling failed", "error", err.Error())
		return pgconn.CommandTag{}, err
	}

	data, err := json.Marshal(natsDTO.EventDTO{
		EventType: event.SessionCompleteEventType.String(),
		Payload:   payloadData,
	})
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "marshalling failed with err: %v", err)
		slog.Error("marshalling failed", "error", err.Error())
		return pgconn.CommandTag{}, err
	}

	params := []interface{}{event.SessionCompleteEventType, data}

	query := `INSERT INTO kvs.outbox (type, payload) VALUES ($1, $2)`

	return tx.Exec(ctx, query, params...)
}

func (s *Storage) GetUnpublishedEvents(ctx context.Context) ([]event.Event, error) {
	slog.Info("GetUnpublishedEvents started")
	ctx, span, cancel := tracer.Start(ctx, "GetUnpublishedEventsPostgresSpan")
	defer cancel()

	query := `
		SELECT id, type, payload
		FROM kvs.outbox
		WHERE published = FALSE
		LIMIT 100
		FOR UPDATE SKIP LOCKED;
	`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "rows with unsended events failure: %v", err)
		slog.Warn("rows with unsended events", "error", err.Error())
		span.SetError(err)
		return nil, err
	}
	defer rows.Close()

	unpublishedEvents := make([]event.Event, 0, 100)
	for rows.Next() {
		var (
			id        int
			eventType string
			payload   []byte
		)

		if err := rows.Scan(&id, &eventType, &payload); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "scan unsended events failure: %v", err)
			slog.Warn("scan unsended events", "error", err.Error())
			span.SetError(err)
			return nil, err
		}
		var concreteEvent event.Event
		switch eventType {
		case event.SessionCompleteEventType.String():
			sessionCompleteEvent, err := event.NewSessionCompleteEvent(payload)
			if err != nil {
				err := errors.Wrapf(entities.ErrInternal, "NewSessionCompleteEvent failure: %v", err)
				slog.Warn("NewSessionCompleteEvent", "error", err.Error())
				span.SetError(err)
				return nil, err
			}
			sessionCompleteEvent.SetNum(id)

			concreteEvent = sessionCompleteEvent
		default:
			err := errors.Wrap(entities.ErrInternal, "unknown event type")
			slog.Warn("unknown event type", "error", err.Error())
			span.SetError(err)
			return nil, err
		}
		unpublishedEvents = append(unpublishedEvents, concreteEvent)
	}

	return unpublishedEvents, nil
}

//nolint:funlen,gosec //ok
func (s *Storage) MarkEventAsPublished(
	ctx context.Context,
	id int,
	fn func(ctx context.Context) error,
) error {
	slog.Info("MarkEventAsPublished started")
	ctx, span, cancel := tracer.Start(ctx, "MarkEventAsPublishedPostgresSpan")
	defer cancel()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal,
			"create new transaction finished with failure: %v", err)
		slog.Error("create new transaction finished with failure", "error", err.Error())
		span.SetError(err)
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx) //nolint:errcheck //ok
		}
	}()

	params := []interface{}{id}

	query := `
		UPDATE kvs.outbox
		SET
		published = TRUE,
		published_at = NOW()
		WHERE id = $1;
	`

	tag, err := tx.Exec(ctx, query, params...)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "mark event as sended failure: %v", err)
		slog.Error("mark event as sended failure", "error", err.Error())
		span.SetError(err)
		return err
	}

	if tag.RowsAffected() == 0 {
		err = errors.Wrapf(entities.ErrInternal, "expected one affected row, now: %d", tag.RowsAffected())
		slog.Error("expected one affected row", "error", err.Error())
		span.SetError(err)
		return err
	}

	if err = fn(ctx); err != nil {
		slog.Error("send to publisher", "error", err.Error())
		span.SetError(err)
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "commit mark event as published: %v", err)
		slog.Error("commit mark event as published", "error", err.Error())
		span.SetError(err)
		return err
	}

	return nil
}

func (s *Storage) FlushPublishedEvents(ctx context.Context) error {
	slog.Info("FlushPublishedEvents started")
	ctx, span, cancel := tracer.Start(ctx, "FlushPublishedEventsPostgresSpan")
	defer cancel()

	query := `
	DELETE FROM kvs.outbox
	WHERE id IN (SELECT id FROM kvs.outbox WHERE published = TRUE LIMIT 1000);
	`
	_, err := s.db.Exec(ctx, query)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "deleting published events finished with err: %v", err)
		slog.Error("deleting published events finished with err", "error", err.Error())
		span.SetError(err)
		return err
	}

	return err
}
