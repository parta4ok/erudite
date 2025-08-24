package entities_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

func TestNewUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		password string
		fullName string
		rights   []string
		contacts map[string]string
		linkedID string
		wantErr  bool
		errCheck func(t *testing.T, err error)
	}{
		{
			name:     "valid user with all fields",
			username: "admin@example.com",
			password: "password123321",
			fullName: "Admin User",
			rights:   []string{"admin", "read", "write"},
			contacts: map[string]string{
				"email":    "admin@example.com",
				"telegram": uuid.NewString(),
			},
			linkedID: "mentor-123",
			wantErr:  false,
		},
		{
			name:     "valid user with minimal fields",
			username: "user@test.com",
			password: "pass",
			fullName: "Test User",
			rights:   []string{"read"},
			contacts: nil,
			linkedID: "",
			wantErr:  false,
		},
		{
			name:     "valid user with empty contacts",
			username: "user2@test.com",
			password: "password",
			fullName: "User Two",
			rights:   []string{"read", "write"},
			contacts: map[string]string{},
			linkedID: "student-456",
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			password: "password123321",
			fullName: "Test User",
			rights:   []string{"read"},
			contacts: nil,
			linkedID: "",
			wantErr:  true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "some of required fields is empty")
			},
		},
		{
			name:     "empty password",
			username: "user@test.com",
			password: "",
			fullName: "Test User",
			rights:   []string{"read"},
			contacts: nil,
			linkedID: "",
			wantErr:  true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "some of required fields is empty")
			},
		},
		{
			name:     "empty fullname",
			username: "user@test.com",
			password: "password123321",
			fullName: "",
			rights:   []string{"read"},
			contacts: nil,
			linkedID: "",
			wantErr:  true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "some of required fields is empty")
			},
		},
		{
			name:     "whitespace only username - should pass (not validated)",
			username: "   ",
			password: "password123321",
			fullName: "Test User",
			rights:   []string{"read"},
			contacts: nil,
			linkedID: "",
			wantErr:  false,
		},
		{
			name:     "whitespace only password - should pass (not validated)",
			username: "user@test.com",
			password: "   ",
			fullName: "Test User",
			rights:   []string{"read"},
			contacts: nil,
			linkedID: "",
			wantErr:  false,
		},
		{
			name:     "whitespace only fullname - should pass (not validated)",
			username: "user@test.com",
			password: "password123321",
			fullName: "   ",
			rights:   []string{"read"},
			contacts: nil,
			linkedID: "",
			wantErr:  false,
		},
		{
			name:     "nil rights",
			username: "user@test.com",
			password: "password123321",
			fullName: "Test User",
			rights:   nil,
			contacts: nil,
			linkedID: "",
			wantErr:  true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "user has not rights")
			},
		},
		{
			name:     "empty rights slice - should pass (only nil is validated)",
			username: "user@test.com",
			password: "password123321",
			fullName: "Test User",
			rights:   []string{},
			contacts: nil,
			linkedID: "",
			wantErr:  false,
		},
		{
			name:     "valid user with single right",
			username: "student@example.com",
			password: "studentpass",
			fullName: "Student Name",
			rights:   []string{"student"},
			contacts: map[string]string{
				"email": "student@example.com",
			},
			linkedID: "mentor-789",
			wantErr:  false,
		},
		{
			name:     "valid user with multiple contacts",
			username: "mentor@example.com",
			password: "mentorpass",
			fullName: "Mentor Name",
			rights:   []string{"mentor", "read"},
			contacts: map[string]string{
				"email":    "mentor@example.com",
				"telegram": uuid.NewString(),
				"phone":    "+1234567890",
			},
			linkedID: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(tc *testing.T) {
			tc.Parallel()

			user, err := entities.NewUser(
				tt.username,
				tt.password,
				tt.fullName,
				tt.rights,
				tt.contacts,
				tt.linkedID,
			)

			if tt.wantErr {
				require.Error(tc, err)
				require.Nil(tc, user)
				if tt.errCheck != nil {
					tt.errCheck(tc, err)
				}
				return
			}

			require.NoError(tc, err)
			require.NotNil(tc, user)
			require.Equal(tc, tt.username, user.Username)
			require.Equal(tc, tt.password, user.PasswordHash)
			require.Equal(tc, tt.fullName, user.FullName)
			require.Equal(tc, tt.rights, user.Rights)
			require.Equal(tc, tt.contacts, user.Contacts)
			require.Equal(tc, tt.linkedID, user.LinkedID)
			require.Empty(tc, user.ID)
		})
	}
}
