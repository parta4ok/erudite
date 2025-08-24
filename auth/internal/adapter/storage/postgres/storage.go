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
	query := `SELECT uid, name, password_hash, rights, contacts, linked_id, fullname FROM
	auth.users where uid = $1 LIMIT 1`

	return s.processRow(s.db.QueryRow(ctx, query, params...))

}

func (s *Storage) GetUserByUsername(ctx context.Context, userName string) (*entities.User, error) {
	slog.Info("Get user by name started")

	params := []interface{}{userName}
	query := `SELECT uid, name, password_hash, rights, contacts, linked_id, fullname FROM
	auth.users where name = $1 LIMIT 1`

	return s.processRow(s.db.QueryRow(ctx, query, params...))
}

func (s *Storage) processRow(row pgx.Row) (*entities.User, error) {
	slog.Info("processRow started")

	var (
		id           string
		username     string
		passwordHash string
		rights       []string
		contactsRaw  []byte
		linkedID     string
		fullname     string
	)

	if err := row.Scan(
		&id,
		&username,
		&passwordHash,
		&rights,
		&contactsRaw,
		&linkedID,
		&fullname,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = errors.Wrap(entities.ErrNotFound, "user not found")
			slog.Error(err.Error())
			return nil, err
		}
		err = errors.Wrapf(entities.ErrInternal, "get user failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	var contacts map[string]string
	if err := json.Unmarshal(contactsRaw, &contacts); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "unmarshal contacts failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("processRow completed")
	return &entities.User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		FullName:     fullname,
		Rights:       rights,
		Contacts:     contacts,
		LinkedID:     linkedID,
	}, nil
}

//nolint:funlen //use spaces for visual division of block code
func (s *Storage) StoreUser(ctx context.Context, user *entities.User) error {
	slog.Info("StoreUser started")

	contactsRaw, err := json.Marshal(user.Contacts)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal failure: %v", err)
		slog.Error(err.Error())
		return err
	}

	tx, err := s.db.Begin(ctx)
	defer func() {
		if err != nil {
			if err := tx.Rollback(ctx); err != nil {
				slog.Warn(err.Error())
			}
		}
	}()

	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "transaction failure with err: %v", err)
		slog.Error(err.Error())
		return err
	}

	var paramsForCheck = []interface{}{user.ID, user.Username}
	queryForCheck := `SELECT uid FROM auth.users WHERE uid = $1 OR name = $2 LIMIT 1`
	row := tx.QueryRow(ctx, queryForCheck, paramsForCheck...)
	var uid string
	err = row.Scan(&uid)

	if err == nil {
		err = errors.Wrapf(entities.ErrAlreadyExists, "uid = '%s' or name = '%s' already exists",
			user.ID, user.Username)
		slog.Error(err.Error())
		return err
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		err = errors.Wrapf(entities.ErrInternal, "transaction failure with err: %v", err)
		slog.Error(err.Error())
		return err
	}

	var params = []interface{}{user.ID, user.Username, user.PasswordHash, user.Rights,
		contactsRaw, user.LinkedID, user.FullName}
	query := `INSERT INTO auth.users (uid, name, password_hash, rights, contacts, linked_id, fullname)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if _, err = tx.Exec(ctx, query, params...); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "save user failure: %v", err)
		slog.Error(err.Error())
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "commit failure with err: %v", err)
		slog.Error(err.Error())
		return err
	}

	slog.Info("StoreUser completed")
	return nil
}

func (s *Storage) RemoveUser(ctx context.Context, userID string) error {
	slog.Info("Removing user started")

	query := `DELETE FROM auth.users WHERE uid = $1`
	args := []interface{}{userID}

	tag, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "exec delete query failure: %v", err)
		slog.Error(err.Error())
		return err
	}

	if tag.RowsAffected() == 0 {
		err = errors.Wrapf(entities.ErrNotFound, "not found user with id='%s'", userID)
		slog.Warn(err.Error())
		return err
	}

	slog.Info("Removing user finished")
	return nil
}

func (s *Storage) UpdateUser(ctx context.Context, user *entities.User) error {
	slog.Info("User update started")

	query := `
	UPDATE auth.users
	SET
		name = COALESCE($1, name),
		password_hash = COALESCE($2, password_hash),
		rights = COALESCE($3, rights),
		contacts = COALESCE($4, contacts),
		linked_id = COALESCE($5, linked_id),
		fullname = COALESCE($6, fullname)
	WHERE uid = $7;
	`
	args := make([]interface{}, 7)

	if user.Username != "" {
		args[0] = user.Username
	}

	if user.PasswordHash != "" {
		args[1] = user.PasswordHash
	}

	if len(user.Rights) != 0 {
		args[2] = user.Rights
	}

	if len(user.Contacts) != 0 {
		args[3] = user.Contacts
	}

	if user.LinkedID != "" {
		args[4] = user.LinkedID
	}

	if user.FullName != "" {
		args[5] = user.FullName
	}

	args[6] = user.ID

	tag, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "update user failure with err: %v", err)
		slog.Error(err.Error())
		return err
	}

	if tag.RowsAffected() == 0 {
		err = errors.Wrap(entities.ErrNotFound, "user with requested id not found")
		slog.Error(err.Error())
		return err
	}

	return nil
}

func (s *Storage) GetUserByLinkedID(ctx context.Context, userID string) (*entities.User, error) {
	slog.Info("GetUserByLinkedID started")

	args := []interface{}{userID}

	query := `SELECT uid, name, password_hash, rights, contacts, linked_id, fullname FROM
	auth.users where uid = (SELECT linked_id from auth.users WHERE uid = $1)`

	var uid, name, passwordHash, linkedID, fullname string
	var rights []string
	var contacts map[string]string

	row := s.db.QueryRow(ctx, query, args...)
	if err := row.Scan(
		&uid,
		&name,
		&passwordHash,
		&rights,
		&contacts,
		&linkedID,
		&fullname,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = errors.Wrapf(entities.ErrNotFound, "user not found: %v", err)
			slog.Error(err.Error())
			return nil, err
		}
		err = errors.Wrapf(entities.ErrInternal, "get user by linked_id failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetUserByLinkedID completed")
	return &entities.User{
		ID:           uid,
		Username:     name,
		PasswordHash: passwordHash,
		FullName:     fullname,
		Rights:       rights,
		Contacts:     contacts,
		LinkedID:     linkedID,
	}, nil
}
