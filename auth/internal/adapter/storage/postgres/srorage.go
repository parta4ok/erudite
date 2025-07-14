package postgres

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
)

var (
	_ common.Storage = (*Storage)(nil)
)

const (
	DefaultTopicLimit = 10
)

type Storage struct {
	db     *pgxpool.Pool
	once   sync.Once
	cancel context.CancelFunc
}

type StorageOption func(s *Storage)

func (s *Storage) setOptions(opts ...StorageOption) {
	for _, opt := range opts {
		opt(s)
	}
}

func NewStorage(connectionString string, opts ...StorageOption) (*Storage, error) {
	if strings.TrimSpace(connectionString) == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "connection string is empty")
	}
	st := &Storage{}

	st.setOptions(opts...)

	ctx, cancel := context.WithCancel(context.Background())
	st.cancel = cancel

	db, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInvalidParam, "connection creating error: %v", err.Error())
	}
	st.db = db

	return st, nil
}

func (s *Storage) Close() {
	s.once.Do(func() {
		s.cancel()
		s.db.Close()
	})
}

func (s *Storage) GetUserByID(ctx context.Context, userID string) (*entities.User, error) {
	slog.Info("Get user by userID started")

	params := []interface{}{userID}
	query := `SELECT user_id, name, password_hash, rights, contacts FROM 
	auth.users where user_id = $1 LIMIT 1`

	return s.processRow(s.db.QueryRow(ctx, query, params...))

}

func (s *Storage) GetUserByUsername(ctx context.Context, userName string) (*entities.User, error) {
	slog.Info("Get user by name started")

	params := []interface{}{userName}
	query := `SELECT user_id, name, password_hash, rights, contacts FROM 
	auth.users where name = $1 LIMIT 1`

	return s.processRow(s.db.QueryRow(ctx, query, params...))
}

func (s *Storage) processRow(row pgx.Row) (*entities.User, error) {
	slog.Info("processRow started")

	var (
		id           string
		Username     string
		PasswordHash string
		Rights       []string
		ContactsRaw  []byte
	)

	if err := row.Scan(&id, &Username, &PasswordHash, &Rights, &ContactsRaw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = errors.Wrap(entities.ErrNotFound, "user not found")
			slog.Error(err.Error())
			return nil, err
		}
		err = errors.Wrapf(entities.ErrInternal, "get user failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	var Contacts map[string]string
	if err := json.Unmarshal(ContactsRaw, &Contacts); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "unmarshal contacts failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("processRow completed")
	return &entities.User{
		ID:           id,
		Username:     Username,
		PasswordHash: PasswordHash,
		Rights:       Rights,
		Contacts:     Contacts,
	}, nil
}

func (s *Storage) StoreUser(ctx context.Context, user *entities.User) error {
	slog.Info("StoreUser started")

	contactsRaw, err := json.Marshal(user.Contacts)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal failure: %v", err)
		slog.Error(err.Error())
		return err
	}

	params := []interface{}{user.ID, user.Username, user.PasswordHash, user.Rights, contactsRaw}
	query := `INSERT INTO auth.users (user_id, name, password_hash, rights, contacts) 
				VALUES ($1, $2, $3, $4, $5)`

	if _, err = s.db.Exec(ctx, query, params...); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "save user failure: %v", err)
		slog.Error(err.Error())
		return err
	}

	slog.Info("StoreUser completed")
	return nil
}

func (s *Storage) UpdateUser(ctx context.Context, user *entities.User) error {
	return nil
}
