package postgres

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/knowledge_checker/internal/cases"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
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
	params = append(params, topics)

	query := `
	SELECT q.question_id, qt.name as question_type, t.name as topic, q.subject, q.variants, q.correct_answers 
	FROM kvs.questions q
	JOIN kvs.topics t ON q.topic_id = t.topic_id
	JOIN kvs.question_types qt on q.question_type_id  = qt.id
	WHERE t.name = ANY($1::text[]) LIMIT 60;
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
				"scan questions data failure: %s", errDB.Error())
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
				"creating questions failure: %s", errDB.Error())
			slog.Error(err.Error())
			return nil, err
		}

		questions = append(questions, question)
	}

	return questions, nil
}

func (s *Storage) StoreSession(ctx context.Context, session *entities.Session) error { return nil }
func (s *Storage) GetSessionBySessionID(ctx context.Context, sessionID uint64) (*entities.Session, error) {
	return nil, nil
}
