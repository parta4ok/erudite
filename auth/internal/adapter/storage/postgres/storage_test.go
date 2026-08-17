//go:build KVS_TEST_L1

package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/parta4ok/kvs/auth/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/auth/internal/entities"
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

func TestStorage_GetUserByID(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	ctx := context.TODO()
	var UserID = "1"

	user, err := db.GetUserByID(ctx, UserID)
	require.NoError(t, err)

	require.Equal(t, user.Username, "admin@kvs.ru")

	UserID = fmt.Sprintf("%d", uint64(time.Now().UTC().UnixNano()))
	user, err = db.GetUserByID(ctx, UserID)
	require.ErrorIs(t, err, entities.ErrNotFound)

	require.Nil(t, user)
}

func TestStorage_GetUserByUsername(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	ctx := context.TODO()
	var userName = "admin@kvs.ru"

	user, err := db.GetUserByUsername(ctx, userName)
	require.NoError(t, err)

	require.Equal(t, user.ID, "1")

	userName = "John Doe"
	user, err = db.GetUserByUsername(ctx, userName)
	require.ErrorIs(t, err, entities.ErrNotFound)

	require.Nil(t, user)
}

func TestStorage_StoreUser(t *testing.T) {
	db := makeDB(t)
	defer db.Close()

	ctx := context.TODO()
	id := fmt.Sprintf("%d", uint64(time.Now().UTC().UnixNano()))
	testUser := &entities.User{
		ID:           id,
		Username:     uuid.New().String(),
		PasswordHash: uuid.New().String(),
		FullName:     uuid.NewString(),
		Rights:       []string{"read", "write"},
		Contacts:     map[string]string{"phone": "891111-11", "tg": "@JDoe"},
	}

	err := db.StoreUser(ctx, testUser)
	require.NoError(t, err)

	user, err := db.GetUserByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, testUser, user)
}

func TestStorage_RemoveUser_Success(t *testing.T) {
	db := makeDB(t)
	defer db.Close()
	ctx := context.TODO()

	id := uuid.New().String()
	user := &entities.User{
		ID:           id,
		Username:     uuid.New().String(),
		PasswordHash: uuid.New().String(),
		Rights:       []string{"read", "write"},
		Contacts:     map[string]string{"phone": "1234567890"},
	}

	require.NoError(t, db.StoreUser(ctx, user))

	require.NoError(t, db.RemoveUser(ctx, id))

	usr, err := db.GetUserByID(ctx, id)
	require.ErrorIs(t, err, entities.ErrNotFound)
	require.Nil(t, usr)
}

func TestStorage_RemoveUser_NotFound(t *testing.T) {
	db := makeDB(t)
	defer db.Close()
	ctx := context.TODO()

	err := db.RemoveUser(ctx, "non-existent-id")
	require.ErrorIs(t, err, entities.ErrNotFound)
}

func ptr[T any](v T) *T {
	return &v
}

func TestStorage_UpdateUser(t *testing.T) {
	type fields struct {
		checkFunc func(t *testing.T, base, updated *entities.User, changes *entities.UserUpdate)
	}
	type args struct {
		updatedUser *entities.UserUpdate
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "update username",
			args: args{
				updatedUser: &entities.UserUpdate{
					Username: ptr(uuid.NewString()),
				},
			},
			fields: fields{
				checkFunc: userNameUpdatedCheck,
			},
		},
		{
			name: "update password hash",
			args: args{
				updatedUser: &entities.UserUpdate{
					PasswordHash: ptr(uuid.NewString()),
				},
			},
			fields: fields{
				checkFunc: passwordHashUpdatedCheck,
			},
		},
		{
			name: "update rights",
			args: args{
				updatedUser: &entities.UserUpdate{
					Rights: ptr([]string{uuid.NewString()}),
				},
			},
			fields: fields{
				checkFunc: rightsUpdatedCheck,
			},
		},
		{
			name: "update contacts",
			args: args{
				updatedUser: &entities.UserUpdate{
					Contacts: ptr(map[string]string{uuid.NewString(): uuid.NewString()}),
				},
			},
			fields: fields{
				checkFunc: contactsUpdatedCheck,
			},
		},
		{
			name: "update fullname",
			args: args{
				updatedUser: &entities.UserUpdate{
					FullName: ptr(uuid.NewString()),
				},
			},
			fields: fields{
				checkFunc: fullnameUpdatedCheck,
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			db := makeDB(it)
			defer db.Close()

			ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
			defer cancel()

			baseUser := &entities.User{
				ID:           uuid.NewString(),
				Username:     uuid.NewString(),
				PasswordHash: uuid.NewString(),
				FullName:     uuid.NewString(),
				Rights:       []string{uuid.NewString()},
				Contacts:     map[string]string{uuid.NewString(): uuid.NewString()},
			}

			err := db.StoreUser(ctx, baseUser)
			require.NoError(t, err)

			tc.args.updatedUser.ID = baseUser.ID

			err = db.UpdateUser(ctx, tc.args.updatedUser)
			require.NoError(t, err)

			resUser, err := db.GetUserByID(ctx, baseUser.ID)
			require.NoError(t, err)

			tc.fields.checkFunc(t, baseUser, resUser, tc.args.updatedUser)
		})
	}
}

func userNameUpdatedCheck(t *testing.T, base, updated *entities.User, changes *entities.UserUpdate) {
	t.Helper()

	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, *changes.Username, updated.Username)
	require.Equal(t, base.PasswordHash, updated.PasswordHash)
	require.Equal(t, base.Rights, updated.Rights)
	require.Equal(t, base.Contacts, updated.Contacts)
	require.Equal(t, base.GroupID, updated.GroupID)
	require.Equal(t, base.FullName, updated.FullName)
}

func passwordHashUpdatedCheck(t *testing.T, base, updated *entities.User, changes *entities.UserUpdate) {
	t.Helper()

	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, base.Username, updated.Username)
	require.Equal(t, *changes.PasswordHash, updated.PasswordHash)
	require.Equal(t, base.Rights, updated.Rights)
	require.Equal(t, base.Contacts, updated.Contacts)
	require.Equal(t, base.GroupID, updated.GroupID)
	require.Equal(t, base.FullName, updated.FullName)

}

func rightsUpdatedCheck(t *testing.T, base, updated *entities.User, changes *entities.UserUpdate) {
	t.Helper()

	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, base.Username, updated.Username)
	require.Equal(t, base.PasswordHash, updated.PasswordHash)
	require.Equal(t, *changes.Rights, updated.Rights)
	require.Equal(t, base.Contacts, updated.Contacts)
	require.Equal(t, base.GroupID, updated.GroupID)
	require.Equal(t, base.FullName, updated.FullName)

}

func contactsUpdatedCheck(t *testing.T, base, updated *entities.User, changes *entities.UserUpdate) {
	t.Helper()

	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, base.Username, updated.Username)
	require.Equal(t, base.PasswordHash, updated.PasswordHash)
	require.Equal(t, base.Rights, updated.Rights)
	require.Equal(t, *changes.Contacts, updated.Contacts)
	require.Equal(t, base.GroupID, updated.GroupID)
	require.Equal(t, base.FullName, updated.FullName)

}

func fullnameUpdatedCheck(t *testing.T, base, updated *entities.User, changes *entities.UserUpdate) {
	t.Helper()

	require.Equal(t, base.ID, updated.ID)
	require.Equal(t, base.Username, updated.Username)
	require.Equal(t, base.PasswordHash, updated.PasswordHash)
	require.Equal(t, base.Rights, updated.Rights)
	require.Equal(t, base.Contacts, updated.Contacts)
	require.Equal(t, base.GroupID, updated.GroupID)
	require.Equal(t, *changes.FullName, updated.FullName)
}

func TestStorage_UpdateUser_GroupID(t *testing.T) {
	t.Parallel()

	db := makeDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	baseUser := &entities.User{
		ID:           uuid.NewString(),
		Username:     uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     uuid.NewString(),
		Rights:       []string{uuid.NewString()},
	}
	err := db.StoreUser(ctx, baseUser)
	require.NoError(t, err)

	groupID := uuid.NewString()
	err = db.AddGroup(ctx, groupID, "Group_"+uuid.NewString(), uuid.NewString())
	require.NoError(t, err)

	err = db.UpdateUser(ctx, &entities.UserUpdate{
		ID:      baseUser.ID,
		GroupID: ptr(groupID),
	})
	require.NoError(t, err)

	resUser, err := db.GetUserByID(ctx, baseUser.ID)
	require.NoError(t, err)
	require.Equal(t, groupID, resUser.GroupID)
}

func TestStorage_UpdateUser_GroupID_EmptyStringIsNoop(t *testing.T) {
	t.Parallel()

	db := makeDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	groupID := uuid.NewString()
	err := db.AddGroup(ctx, groupID, "Group_"+uuid.NewString(), uuid.NewString())
	require.NoError(t, err)

	baseUser := &entities.User{
		ID:           uuid.NewString(),
		Username:     uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     uuid.NewString(),
		Rights:       []string{uuid.NewString()},
		GroupID:      groupID,
	}
	err = db.StoreUser(ctx, baseUser)
	require.NoError(t, err)

	err = db.UpdateUser(ctx, &entities.UserUpdate{
		ID:      baseUser.ID,
		GroupID: ptr(""),
	})
	require.NoError(t, err)

	resUser, err := db.GetUserByID(ctx, baseUser.ID)
	require.NoError(t, err)
	require.Equal(t, groupID, resUser.GroupID, "empty string must not clear group_id (current no-op behaviour)")
}

func TestStorage_FlowWithGroupAndLinkedUsers(t *testing.T) {
	t.Parallel()

	db := makeDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mentor := &entities.User{
		ID:           uuid.NewString(),
		Username:     uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     uuid.NewString(),
		Rights:       []string{uuid.NewString()},
		Contacts:     map[string]string{uuid.NewString(): uuid.NewString()},
	}

	err := db.StoreUser(ctx, mentor)
	require.NoError(t, err)

	gid := uuid.NewString()
	gTitle := uuid.NewString()

	err = db.AddGroup(ctx, gid, gTitle, mentor.ID)
	require.NoError(t, err)

	student := &entities.User{
		ID:           uuid.NewString(),
		Username:     uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     uuid.NewString(),
		Rights:       []string{uuid.NewString()},
		Contacts:     map[string]string{uuid.NewString(): uuid.NewString()},
		GroupID:      gid,
	}

	err = db.StoreUser(ctx, student)
	require.NoError(t, err)

	pair, err := db.GetLinkedUsers(ctx, student.ID)
	require.NoError(t, err)

	require.Equal(t, mentor, pair.Recipient)
	require.Equal(t, student, pair.Student)
}

func TestStorage_GetMentorGroups(t *testing.T) {
	t.Parallel()

	db := makeDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mentorID := uuid.NewString()
	mentor := &entities.User{
		ID:           mentorID,
		Username:     "mentor_" + uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     "Test Mentor",
		Rights:       []string{"mentor"},
		Contacts:     map[string]string{"email": "mentor@test.com"},
	}

	err := db.StoreUser(ctx, mentor)
	require.NoError(t, err)

	group1ID := uuid.NewString()
	group1Title := "Group_Math_2024_" + uuid.NewString()
	err = db.AddGroup(ctx, group1ID, group1Title, mentorID)
	require.NoError(t, err)

	group2ID := uuid.NewString()
	group2Title := "Group_Physics_2024_" + uuid.NewString()
	err = db.AddGroup(ctx, group2ID, group2Title, mentorID)
	require.NoError(t, err)

	student1Math := &entities.User{
		ID:           uuid.NewString(),
		Username:     "math_student1_" + uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     "Math Student One",
		Rights:       []string{"student"},
		Contacts:     map[string]string{"email": "math.student1@test.com"},
		GroupID:      group1ID,
	}

	student2Math := &entities.User{
		ID:           uuid.NewString(),
		Username:     "math_student2_" + uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     "Math Student Two",
		Rights:       []string{"student"},
		Contacts:     map[string]string{"email": "math.student2@test.com"},
		GroupID:      group1ID,
	}

	student1Physics := &entities.User{
		ID:           uuid.NewString(),
		Username:     "physics_student1_" + uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     "Physics Student One",
		Rights:       []string{"student"},
		Contacts:     map[string]string{"email": "physics.student1@test.com"},
		GroupID:      group2ID,
	}

	student2Physics := &entities.User{
		ID:           uuid.NewString(),
		Username:     "physics_student2_" + uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     "Physics Student Two",
		Rights:       []string{"student"},
		Contacts:     map[string]string{"email": "physics.student2@test.com"},
		GroupID:      group2ID,
	}

	student3Physics := &entities.User{
		ID:           uuid.NewString(),
		Username:     "physics_student3_" + uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     "Physics Student Three",
		Rights:       []string{"student"},
		Contacts:     map[string]string{"email": "physics.student3@test.com"},
		GroupID:      group2ID,
	}

	err = db.StoreUser(ctx, student1Math)
	require.NoError(t, err)

	err = db.StoreUser(ctx, student2Math)
	require.NoError(t, err)

	err = db.StoreUser(ctx, student1Physics)
	require.NoError(t, err)

	err = db.StoreUser(ctx, student2Physics)
	require.NoError(t, err)

	err = db.StoreUser(ctx, student3Physics)
	require.NoError(t, err)

	groups, err := db.GetMentorGroups(ctx, mentorID)
	require.NoError(t, err)
	require.NotNil(t, groups)
	require.Len(t, groups, 2)

	var mathGroup, physicsGroup *entities.Group
	for _, group := range groups {
		if group.GetID() == group1ID {
			mathGroup = group
		} else if group.GetID() == group2ID {
			physicsGroup = group
		}
	}

	require.NotNil(t, mathGroup)
	require.Equal(t, group1ID, mathGroup.GetID())
	require.Len(t, mathGroup.GetStudents(), 2)

	mathStudentIDs := make(map[string]bool)
	for _, student := range mathGroup.GetStudents() {
		mathStudentIDs[student.GetID()] = true
	}
	require.True(t, mathStudentIDs[student1Math.ID])
	require.True(t, mathStudentIDs[student2Math.ID])

	require.NotNil(t, physicsGroup)
	require.Equal(t, group2ID, physicsGroup.GetID())
	require.Len(t, physicsGroup.GetStudents(), 3)

	physicsStudentIDs := make(map[string]bool)
	for _, student := range physicsGroup.GetStudents() {
		physicsStudentIDs[student.GetID()] = true
	}
	require.True(t, physicsStudentIDs[student1Physics.ID])
	require.True(t, physicsStudentIDs[student2Physics.ID])
	require.True(t, physicsStudentIDs[student3Physics.ID])
}

func TestStorage_GetGroupTitleByID_Success(t *testing.T) {
	t.Parallel()

	db := makeDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testUser := &entities.User{
		ID:           uuid.NewString(),
		Username:     uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     uuid.NewString(),
		Rights:       []string{"read", "write"},
		Contacts:     map[string]string{"phone": "891111-11", "tg": "@JDoe"},
	}

	err := db.StoreUser(ctx, testUser)
	require.NoError(t, err)

	groupID := uuid.NewString()
	groupTitle := uuid.NewString()

	err = db.AddGroup(ctx, groupID, groupTitle, testUser.ID)
	require.NoError(t, err)

	resTitle, err := db.GetGroupTitleByID(ctx, groupID)
	require.NoError(t, err)
	require.Equal(t, groupTitle, resTitle)
}

func TestStorage_GetGroupTitleByID_NotFound(t *testing.T) {
	t.Parallel()

	db := makeDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	groupID := uuid.NewString()

	resTitle, err := db.GetGroupTitleByID(ctx, groupID)
	require.ErrorIs(t, err, entities.ErrNotFound)
	require.Equal(t, "", resTitle)
}

func TestStorage_GetAllUsers(t *testing.T) {
	t.Parallel()

	db := makeDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user1 := &entities.User{
		ID:           uuid.NewString(),
		Username:     "all_users_" + uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     "All Users Test One",
		Rights:       []string{"student"},
		Contacts:     map[string]string{"email": "all-users-1@test.com"},
	}

	user2 := &entities.User{
		ID:           uuid.NewString(),
		Username:     "all_users_" + uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     "All Users Test Two",
		Rights:       []string{"mentor"},
		Contacts:     map[string]string{"email": "all-users-2@test.com"},
	}

	err := db.StoreUser(ctx, user1)
	require.NoError(t, err)

	err = db.StoreUser(ctx, user2)
	require.NoError(t, err)

	users, err := db.GetAllUsers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)

	userIDs := make(map[string]bool)
	for _, user := range users {
		userIDs[user.ID] = true
	}
	require.True(t, userIDs[user1.ID])
	require.True(t, userIDs[user2.ID])
}

func TestStorage_GetAllGroups(t *testing.T) {
	t.Parallel()

	db := makeDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mentorID := uuid.NewString()
	mentor := &entities.User{
		ID:           mentorID,
		Username:     "all_groups_mentor_" + uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     "All Groups Test Mentor",
		Rights:       []string{"mentor"},
		Contacts:     map[string]string{"email": "all-groups-mentor@test.com"},
	}

	err := db.StoreUser(ctx, mentor)
	require.NoError(t, err)

	groupID := uuid.NewString()
	groupTitle := "AllGroups_" + uuid.NewString()

	err = db.AddGroup(ctx, groupID, groupTitle, mentorID)
	require.NoError(t, err)

	student := &entities.User{
		ID:           uuid.NewString(),
		Username:     "all_groups_student_" + uuid.NewString(),
		PasswordHash: uuid.NewString(),
		FullName:     "All Groups Test Student",
		Rights:       []string{"student"},
		Contacts:     map[string]string{"email": "all-groups-student@test.com"},
		GroupID:      groupID,
	}

	err = db.StoreUser(ctx, student)
	require.NoError(t, err)

	groups, err := db.GetAllGroups(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, groups)

	var createdGroup *entities.Group
	for _, group := range groups {
		if group.GetID() == groupID {
			createdGroup = group
		}
	}

	require.NotNil(t, createdGroup)
	require.Equal(t, groupTitle, createdGroup.GetName())
	require.Equal(t, mentorID, createdGroup.GetLinkedID())
	require.Len(t, createdGroup.GetStudents(), 1)
	require.Equal(t, student.ID, createdGroup.GetStudents()[0].GetID())
}
