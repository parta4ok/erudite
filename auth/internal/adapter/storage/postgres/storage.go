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
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
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
	ctx, _, cancel := tracing.GlobalTracer().Start(ctx, "GetUserByIDPostgresSpan")
	defer cancel()

	params := []interface{}{userID}
	query := `SELECT uid, name, password_hash, rights, contacts, group_id, fullname FROM
	auth.users where uid = $1 LIMIT 1`

	return s.processRow(s.db.QueryRow(ctx, query, params...))

}

func (s *Storage) GetUserByUsername(ctx context.Context, userName string) (*entities.User, error) {
	slog.Info("Get user by name started")
	ctx, _, cancel := tracing.GlobalTracer().Start(ctx, "GetUserByUsernamePostgresSpan")
	defer cancel()

	params := []interface{}{userName}
	query := `SELECT uid, name, password_hash, rights, contacts, group_id, fullname FROM
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
		groupIDRaw   interface{}
		fullname     string
	)

	if err := row.Scan(
		&id,
		&username,
		&passwordHash,
		&rights,
		&contactsRaw,
		&groupIDRaw,
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

	var groupID string
	if groupIDRaw != nil {
		groupID, _ = groupIDRaw.(string)
	}

	slog.Info("processRow completed")
	return &entities.User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		FullName:     fullname,
		Rights:       rights,
		Contacts:     contacts,
		GroupID:      groupID,
	}, nil
}

//nolint:funlen //use spaces for visual division of block code
func (s *Storage) StoreUser(ctx context.Context, user *entities.User) error {
	slog.Info("StoreUser started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "StoreUserPostgresSpan")
	defer cancel()

	contactsRaw, err := json.Marshal(user.Contacts)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "marshal failure")
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
		span.SetError(err, "transaction failure")
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
		span.SetError(err, "user already exists")
		return err
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		err = errors.Wrapf(entities.ErrInternal, "transaction failure with err: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "transaction failure")
		return err
	}

	var gid interface{}
	if user.GroupID != "" {
		gid = user.GroupID
	}

	var params = []interface{}{user.ID, user.Username, user.PasswordHash, user.Rights,
		contactsRaw, gid, user.FullName}
	slog.Info("-- store user with params", slog.Any("params", params))
	query := `INSERT INTO auth.users (uid, name, password_hash, rights, contacts, group_id, fullname)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if _, err = tx.Exec(ctx, query, params...); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "save user failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "save user failure")
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "commit failure with err: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "commit failure")
		return err
	}

	slog.Info("StoreUser completed")
	return nil
}

func (s *Storage) RemoveUser(ctx context.Context, userID string) error {
	slog.Info("Removing user started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "RemoveUserPostgresSpan")
	defer cancel()

	query := `DELETE FROM auth.users WHERE uid = $1`
	args := []interface{}{userID}

	tag, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "exec delete query failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "exec delete query failure")
		return err
	}

	if tag.RowsAffected() == 0 {
		err = errors.Wrapf(entities.ErrNotFound, "not found user with id='%s'", userID)
		slog.Warn(err.Error())
		span.SetError(err, "user not found")
		return err
	}

	slog.Info("Removing user finished")
	return nil
}

func (s *Storage) UpdateUser(ctx context.Context, user *entities.User) error {
	slog.Info("User update started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "UpdateUserPostgresSpan")
	defer cancel()

	query := `
	UPDATE auth.users
	SET
		name = COALESCE($1, name),
		password_hash = COALESCE($2, password_hash),
		rights = COALESCE($3, rights),
		contacts = COALESCE($4, contacts),
		group_id = COALESCE($5, group_id),
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

	if user.GroupID != "" {
		args[4] = user.GroupID
	}

	if user.FullName != "" {
		args[5] = user.FullName
	}

	args[6] = user.ID

	tag, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "update user failure with err: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "update user failure")
		return err
	}

	if tag.RowsAffected() == 0 {
		err = errors.Wrap(entities.ErrNotFound, "user with requested id not found")
		slog.Error(err.Error())
		span.SetError(err, "user not found")
		return err
	}

	return nil
}

//nolint:funlen //ok
func (s *Storage) GetLinkedUsers(ctx context.Context, userID string,
) (*entities.LinkedUsers, error) {
	slog.Info("GetLinkedUsers started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "GetLinkedUsersPostgresSpan")
	defer cancel()

	args := []interface{}{userID}

	query := `
		SELECT
			student.uid,
			student.name,
			student.password_hash,
			student.rights,
			student.contacts,
			student.group_id,
			student.fullname,
			mentor.uid,
			mentor.name,
			mentor.password_hash,
			mentor.rights,
			mentor.contacts,
			mentor.group_id,
			mentor.fullname
		FROM auth.users student
		LEFT JOIN auth.groups g ON student.group_id = g.gid
		LEFT JOIN auth.users mentor ON g.linked_id = mentor.uid
		WHERE student.uid = $1`

	row := s.db.QueryRow(ctx, query, args...)

	var (
		studentUID, studentName, studentPasswordHash, studentFullname string
		studentRights                                                 []string
		studentContactsRaw                                            []byte
		mentorUID, mentorName, mentorPasswordHash, mentorFullname     string
		mentorRights                                                  []string
		mentorContactsRaw                                             []byte
		studentGroupIDRaw, mentorGroupIDRaw                           interface{}
	)

	if err := row.Scan(
		&studentUID,
		&studentName,
		&studentPasswordHash,
		&studentRights,
		&studentContactsRaw,
		&studentGroupIDRaw,
		&studentFullname,
		&mentorUID,
		&mentorName,
		&mentorPasswordHash,
		&mentorRights,
		&mentorContactsRaw,
		&mentorGroupIDRaw,
		&mentorFullname,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = errors.Wrapf(entities.ErrNotFound, "student with id '%s' not found", userID)
			slog.Error(err.Error())
			span.SetError(err, "student not found")
			return nil, err
		}
		err = errors.Wrapf(entities.ErrInternal, "get linked users failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "get linked users failure")
		return nil, err
	}

	var studentContacts map[string]string
	if err := json.Unmarshal(studentContactsRaw, &studentContacts); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "unmarshal student contacts failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "unmarshal student contacts failure")
		return nil, err
	}

	var mentorContacts map[string]string
	if err := json.Unmarshal(mentorContactsRaw, &mentorContacts); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "unmarshal mentor contacts failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "unmarshal mentor contacts failure")
		return nil, err
	}

	var studentGroupID string
	if studentGroupIDRaw != nil {
		studentGroupID, _ = studentGroupIDRaw.(string)
	}

	student := &entities.User{
		ID:           studentUID,
		Username:     studentName,
		PasswordHash: studentPasswordHash,
		FullName:     studentFullname,
		Rights:       studentRights,
		Contacts:     studentContacts,
		GroupID:      studentGroupID,
	}

	var mentorGroupID string
	if mentorGroupIDRaw != nil {
		mentorGroupID, _ = mentorGroupIDRaw.(string)
	}

	mentor := &entities.User{
		ID:           mentorUID,
		Username:     mentorName,
		PasswordHash: mentorPasswordHash,
		FullName:     mentorFullname,
		Rights:       mentorRights,
		Contacts:     mentorContacts,
		GroupID:      mentorGroupID,
	}

	slog.Info("GetLinkedUsers completed")
	return &entities.LinkedUsers{
		Student:   student,
		Recipient: mentor,
	}, nil
}

func (s *Storage) AddGroup(ctx context.Context, gid, title, mentorID string) error {
	slog.Info("AddGroup started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "AddGroupPostgresSpan")
	defer cancel()

	params := []interface{}{gid, title, mentorID}
	query := `INSERT INTO auth.groups (gid, title, linked_id) VALUES ($1, $2, $3)`

	_, err := s.db.Exec(ctx, query, params...)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "insert group failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "insert group failure")
		return err
	}

	return nil
}

//nolint:funlen //ok
func (s *Storage) GetMentorGroups(ctx context.Context, mentorID string) (
	[]*entities.Group, error) {
	slog.Info("GetMentorsGroups started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "GetMentorsGroupsPostgresSpan")
	defer cancel()

	params := []interface{}{mentorID}
	query := `SELECT gid, title FROM auth.groups WHERE linked_id = $1`

	rows, err := s.db.Query(ctx, query, params...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = errors.Wrapf(entities.ErrNotFound, "no groups found for mentor with id='%s'",
				mentorID)
			slog.Warn(err.Error(), slog.String("mentorID", mentorID))
			return nil, err
		}
		err = errors.Wrapf(entities.ErrInternal, "query groups failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "query groups failure")
		return nil, err
	}
	defer rows.Close()

	groups := make([]*entities.Group, 0)
	gids := make([]string, 0)
	for rows.Next() {
		var (
			gid   string
			title string
		)
		if err := rows.Scan(&gid, &title); err != nil {
			err = errors.Wrapf(entities.ErrInternal, "scan group failure: %v", err)
			slog.Error(err.Error())
			span.SetError(err, "scan group failure")
			return nil, err
		}
		group := entities.NewGroup(gid, title)
		groups = append(groups, group)
		gids = append(gids, gid)
	}

	if err := rows.Err(); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "rows iteration failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "rows iteration failure")
		return nil, err
	}

	if len(gids) == 0 {
		slog.Info("GetMentorsGroups completed with no groups")
		err = errors.Wrapf(entities.ErrNotFound, "no groups found for mentor with id='%s'",
			mentorID)
		slog.Warn(err.Error(), slog.String("mentorID", mentorID))
		return nil, err
	}

	queryStudents := `SELECT uid, name, fullname, group_id FROM auth.users WHERE group_id = ANY($1)`
	rowsStudents, err := s.db.Query(ctx, queryStudents, gids)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Info("GetMentorsGroups completed with no students")
			return groups, nil
		}
		err = errors.Wrapf(entities.ErrInternal, "query students failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "query students failure")
		return nil, err
	}
	defer rowsStudents.Close()

	studentsMap := make(map[string][]*entities.Student)
	for rowsStudents.Next() {
		var (
			studentID   string
			studentName string
			studentFull string
			studentGID  string
		)
		if err := rowsStudents.Scan(&studentID, &studentName, &studentFull, &studentGID); err != nil {
			err = errors.Wrapf(entities.ErrInternal, "scan student failure: %v", err)
			slog.Error(err.Error())
			span.SetError(err, "scan student failure")
			return nil, err
		}
		student := entities.NewStudent(studentID, studentName, studentFull)
		studentsMap[studentGID] = append(studentsMap[studentGID], student)
	}

	if err := rowsStudents.Err(); err != nil {
		err = errors.Wrapf(entities.ErrInternal, "rowsStudents iteration failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "rowsStudents iteration failure")
		return nil, err
	}

	for _, group := range groups {
		if students, ok := studentsMap[group.GetID()]; ok {
			group.AddStudents(students)
		}
	}

	slog.Info("GetMentorsGroups completed")
	return groups, nil
}
