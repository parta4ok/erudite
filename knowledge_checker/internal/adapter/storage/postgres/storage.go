package postgres

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/knowledge_checker/internal/cases"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
	"github.com/parta4ok/kvs/knowledge_checker/pkg/dto"
)

var (
	_ cases.Storage = (*Storage)(nil)
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
	query := `SELECT kvs.name FROM kvs.topics`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		slog.Error(err.Error())
		return nil, errors.Wrapf(entities.ErrInternalError, "getting topic names failure: %v", err)
	}

	topics := make([]string, 0)

	for rows.Next() {
		var topicName string
		rows.Scan(&topicName)
		topics = append(topics, topicName)
	}

	if err := rows.Err(); err != nil {
		slog.Error(err.Error())
		return nil, errors.Wrapf(entities.ErrInternalError, "rows err: %v", err)
	}
	slog.Info("GetTopics completed")

	return topics, nil
}

func (s *Storage) GetQuesions(ctx context.Context, topics []string) (
	[]entities.Question, error) {
	slog.Info("GetQuesions started")

	params := make([]interface{}, 0)
	params = append(params, topics, 3)

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
    	WHERE rn <= 2
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
		err := errors.Wrapf(entities.ErrInternalError,
			"get questions from db failure: %s", errDB.Error())
		slog.Error(err.Error())
		return nil, err
	}

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
		if err := rows.Scan(
			&questionID,
			&questionType,
			&topic,
			&subject,
			&variants,
			&correctAnswer,
		); err != nil {
			err := errors.Wrapf(entities.ErrInternalError,
				"scan questions data failure: %v", errDB)
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
			err := errors.Wrapf(entities.ErrInternalError,
				"creating questions failure: %v", errDB)
			slog.Error(err.Error())
			return nil, err
		}

		questions = append(questions, question)
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
			err := errors.Wrapf(entities.ErrInternalError,
				"get startedAt from session state: %v", err)
			slog.Error(err.Error())
			return err
		}

		duration, err := session.GetSessionDurationLimit()
		if err != nil {
			err := errors.Wrapf(entities.ErrInternalError,
				"get duration limit from session state: %v", err)
			slog.Error(err.Error())
			return err
		}

		parameters = append(parameters, questionsIDs, startedAt, duration)

	case entities.CompletedState:
		query += s.makeCompletedStateSessionQuery()

		questionsIDs, err := s.getQuestionsIDs(session)
		if err != nil {
			slog.Error(err.Error())
			return err
		}

		userAnswers, err := session.GetUserAnswers()
		if err != nil {
			err := errors.Wrapf(entities.ErrInternalError,
				"get user answers from session state: %v", err)
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
			err := errors.Wrapf(entities.ErrInternalError,
				"marshalling failure: %v", err)
			slog.Error(err.Error())
			return err
		}

		isExpired, err := session.IsExpired()
		if err != nil {
			err := errors.Wrapf(entities.ErrInternalError,
				"get session expired status failure: %v", err)
			slog.Error(err.Error())
			return err
		}

		sesseionResult, err := session.GetSessionResult()
		if err != nil {
			err := errors.Wrapf(entities.ErrInternalError,
				"get session result status failure: %v", err)
			slog.Error(err.Error())
			return err
		}

		parameters = append(parameters, questionsIDs, answersListJSON, isExpired, 
			sesseionResult.IsSuccess, sesseionResult.Grade)
	}

	_, err := s.db.Exec(ctx, query, parameters...)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternalError, "store session finished with failure: %v", err)
		slog.Error(err.Error())
		return err
	}

	return nil
}

func (s *Storage) GetSessionBySessionID(ctx context.Context, sessionID uint64) (*entities.Session, error) {
	return nil, nil
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
		err := errors.Wrapf(entities.ErrInternalError,
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
