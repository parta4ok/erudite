package postgres

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	cryptoprocessing "github.com/parta4ok/kvs/knowledge_checker/internal/adapter/generator/crypto_processing"
	"github.com/parta4ok/kvs/knowledge_checker/internal/cases"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
	_ "github.com/parta4ok/kvs/knowledge_checker/internal/port/http/public"
	"github.com/parta4ok/kvs/knowledge_checker/pkg/dto"
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
	query := `SELECT t.name FROM kvs.topics t`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "getting topic names failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}
	defer rows.Close()

	topics := make([]string, 0)

	for rows.Next() {
		var topicName string
		if err := rows.Scan(&topicName); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "scan topic name failure: %v", err)
			slog.Error(err.Error())
			return nil, err
		}
		topics = append(topics, topicName)
	}

	if err := rows.Err(); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "rows err: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetTopics completed")
	return topics, nil
}

func (s *Storage) GetQuesions(ctx context.Context, topics []string) (
	[]entities.Question, error) {
	slog.Info("GetQuesions started")

	params := make([]interface{}, 0)
	params = append(params, topics, s.questionsLimits)

	query := `
	WITH ranked_questions AS (
    SELECT
        q.question_id,
        qt.name AS question_type,
        t.name AS topic,
        q.subject,
        q.variants,
        q.correct_answers,
        ROW_NUMBER() OVER (
            PARTITION BY t.topic_id, qt.id
            ORDER BY q.usage_count ASC, RANDOM()
        ) AS rn
    FROM kvs.questions q
    JOIN kvs.topics t ON q.topic_id = t.topic_id
    JOIN kvs.question_types qt ON q.question_type_id = qt.id
    WHERE t.name = ANY($1::text[])
	),
	to_update AS (
    	SELECT question_id
    	FROM ranked_questions
    	WHERE rn <= $2
	),
	updated AS (
    	UPDATE kvs.questions
    	SET usage_count = usage_count + 1
   		WHERE question_id IN (SELECT question_id FROM to_update)
    	RETURNING question_id
	)
	SELECT
    	rq.question_id,
    	rq.question_type,
    	rq.topic,
   		rq.subject,
    	rq.variants,
    	rq.correct_answers
	FROM ranked_questions rq
	JOIN updated u ON rq.question_id = u.question_id
	WHERE rq.rn <= $2
	ORDER BY rq.topic, rq.question_type;
	`

	rows, errDB := s.db.Query(ctx, query, params...)
	if errDB != nil {
		err := errors.Wrapf(entities.ErrInternal, "get questions from db failure: %v", errDB)
		slog.Error(err.Error())
		return nil, err
	}
	defer rows.Close()

	questions, err := s.processingQuestionsRows(ctx, rows)
	if err != nil {
		err := errors.Wrap(err, "processingQuestionsRows")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetQuesions completed")
	return questions, nil
}

func (s *Storage) StoreSession(ctx context.Context, session *entities.Session) error {
	slog.Info("StoreSession started")

	userID := session.GetUserID()
	sessionID := session.GetSesionID()
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
			slog.Error(err.Error())
			return err
		}

		startedAt, err := session.GetStartedAt()
		if err != nil {
			err := errors.Wrap(err, "session GetStartedAt failure")
			slog.Error(err.Error())
			return err
		}

		duration, err := session.GetSessionDurationLimit()
		if err != nil {
			err := errors.Wrap(err, "session GetSessionDurationLimit failure")
			slog.Error(err.Error())
			return err
		}

		parameters = append(parameters, questionsIDs, startedAt, duration)

	case entities.CompletedState:
		query += s.makeCompletedStateSessionQuery()

		questionsIDs, err := s.getQuestionsIDs(session)
		if err != nil {
			err := errors.Wrap(err, "getQuestionsIDs failure")
			slog.Error(err.Error())
			return err
		}

		userAnswers, err := session.GetUserAnswers()
		if err != nil {
			err := errors.Wrap(err, "session GetUserAnswers failure")
			slog.Error(err.Error())
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
			slog.Error(err.Error())
			return err
		}

		isExpired, err := session.IsExpired()
		if err != nil {
			err := errors.Wrap(err, "session IsExpired failure")
			slog.Error(err.Error())
			return err
		}

		sesseionResult, err := session.GetSessionResult()
		if err != nil {
			err := errors.Wrap(err, "session GetSessionResult failure")
			slog.Error(err.Error())
			return err
		}

		parameters = append(parameters, questionsIDs, answersListJSON, isExpired,
			sesseionResult.IsSuccess, sesseionResult.Grade)
	}

	_, err := s.db.Exec(ctx, query, parameters...)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "store session finished with failure: %v", err)
		slog.Error(err.Error())
		return err
	}

	slog.Info("StoreSession completed")
	return nil
}

func (s *Storage) GetSessionBySessionID(ctx context.Context, sessionID uint64) (*entities.Session,
	error) {
	slog.Info("GetSessionBySessionID started")

	query := `
	SELECT 
    s.user_id,
    s.state,
    s.topics,
    s.questions,
    s.answers,
    s.created_at,
    s.duration_limit,
    s.is_expired
	FROM kvs.sessions s 
	WHERE s.session_id = $1
	ORDER BY s.updated_at DESC
	LIMIT 1;
	`
	sessionParameters := []interface{}{sessionID}

	row := s.db.QueryRow(ctx, query, sessionParameters...)
	var (
		userID         uint64
		stateName      string
		topics         []string
		questionsIDs   []uint64
		answersRaw     []byte
		createdAt      *time.Time
		duration_limit uint64
		isExpired      *bool
	)

	if err := row.Scan(&userID, &stateName, &topics, &questionsIDs, &answersRaw,
		&createdAt, &duration_limit, &isExpired); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "scan session data failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetSessionBySessionID completed")
	return s.recoverSession(ctx, sessionID, stateName, userID, topics, questionsIDs,
		duration_limit, answersRaw, createdAt, isExpired)
}

func (s *Storage) recoverSession(ctx context.Context, sessionID uint64, stateName string,
	userID uint64, topics []string, questionsIDs []uint64, duration_limit uint64, answersRaw []byte,
	createdAt *time.Time, isExpired *bool) (*entities.Session, error) {
	slog.Info("recoverSession started")

	switch stateName {
	case entities.InitState:
		initSession, err := entities.NewSession(userID, topics,
			cryptoprocessing.NewUint64Generator(), s, entities.WithSessionID(sessionID),
			entities.WithNilState())
		if err != nil {
			err = errors.Wrap(err, "creating new session with sessionID option failure")
			slog.Error(err.Error())
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
			slog.Error(err.Error())
			return nil, err
		}

		questions, err := s.getQuestionsByID(ctx, questionsIDs)
		if err != nil {
			err = errors.Wrap(err, "getQuestionsByID failure")
			slog.Error(err.Error())
			return nil, err
		}

		questionsMap := make(map[uint64]entities.Question, len(questions))
		for _, question := range questions {
			questionsMap[question.ID()] = question
		}
		state := entities.NewActiveSessionState(questionsMap, activeSession,
			time.Microsecond*time.Duration(duration_limit), entities.WithStartedAt(*createdAt))
		activeSession.ChangeState(state)

		slog.Info("recoverSession completed")
		return activeSession, nil

	case entities.CompletedState:
		completedSession, err := entities.NewSession(userID, topics,
			cryptoprocessing.NewUint64Generator(), s, entities.WithSessionID(sessionID),
			entities.WithNilState())
		if err != nil {
			err = errors.Wrap(err, "creating new session with sessionID option failure")
			slog.Error(err.Error())
			return nil, err
		}

		questions, err := s.getQuestionsByID(ctx, questionsIDs)
		if err != nil {
			err = errors.Wrap(err, "getQuestionsByID failure")
			slog.Error(err.Error())
			return nil, err
		}

		questionsMap := make(map[uint64]entities.Question, len(questions))
		for _, question := range questions {
			questionsMap[question.ID()] = question
		}

		var answersListDTO dto.UserAnswersListDTO
		if err := json.Unmarshal(answersRaw, &answersListDTO); err != nil {
			err = errors.Wrapf(entities.ErrInternal, "unmarshaling failure: %v", err)
			slog.Error(err.Error())
			return nil, err
		}

		answers := make([]*entities.UserAnswer, 0, len(answersListDTO.AnswersList))
		for _, answerDTO := range answersListDTO.AnswersList {
			answer, err := entities.NewUserAnswer(answerDTO.QuestionID, answerDTO.Answers)
			if err != nil {
				err = errors.Wrap(err, "creating user answer failure")
				slog.Error(err.Error())
				return nil, err
			}
			answers = append(answers, answer)
		}

		state := entities.NewCompletedSessionState(questionsMap, completedSession,
			answers, *isExpired)
		completedSession.ChangeState(state)

		slog.Info("recoverSession completed")
		return completedSession, nil
	}

	err := errors.Wrapf(entities.ErrInternal, "unknown session state: %s", stateName)
	slog.Error(err.Error())
	return nil, err
}

func (s *Storage) getQuestionsByID(ctx context.Context, questionsIDs []uint64) (
	[]entities.Question, error) {
	slog.Info("getQuestionsByID strarted")

	query := `
	SELECT 
    q.question_id,
    qt.name AS question_type_name,
    t.name AS topic_name,
    q.subject,
    q.variants,
    q.correct_answers
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
		slog.Error(err.Error())
		return nil, err
	}
	defer rows.Close()

	questions, err := s.processingQuestionsRows(ctx, rows)
	if err != nil {
		err := errors.Wrap(err, "processingQuestionsRows failure")
		slog.Error(err.Error())
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
			questionID    uint64
			questionType  string
			topic         string
			subject       string
			variants      []string
			correctAnswer []string
		)

		err := rows.Scan(&questionID, &questionType, &topic, &subject, &variants, &correctAnswer)
		if err != nil {
			err := errors.Wrapf(entities.ErrInternal, "scan questions data failure: %v", err)
			slog.Error(err.Error())
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
			slog.Error(err.Error())
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
		, questions, answers, is_expired, is_passed, comment) values ($1, $2, $3, $4, $5, $6, $7, $8, $9);
	`
}

func (s *Storage) getQuestionsIDs(session *entities.Session) ([]uint64, error) {
	questions, err := session.GetQuestions()
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal,
			"get questions from session state: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	questionsIDs := make([]uint64, 0, len(questions))
	for _, q := range questions {
		questionsIDs = append(questionsIDs, q.ID())
	}

	return questionsIDs, nil
}

func (s *Storage) IsDailySessionLimitReached(ctx context.Context, userID uint64,
	topics []string) (bool, error) {
	slog.Info("IsDailySessionLimitReached started")

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
		uid uint64
		t   []string
		cnt int
	)
	if err := row.Scan(&uid, &t, &cnt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Info("IsDailySessionLimitReached completed")
			return false, nil
		}
		err := errors.Wrapf(entities.ErrInternal, "scan failure: %v", err)
		slog.Error(err.Error())
		return false, err
	}

	slog.Info("IsDailySessionLimitReached completed")
	if cnt >= 1 {
		return true, nil
	}

	return false, nil
}
