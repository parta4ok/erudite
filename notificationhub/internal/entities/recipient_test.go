package entities_test

import (
	"testing"

	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	"github.com/stretchr/testify/require"
)

func TestNewRecipient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		id       string
		contacts map[string]string
		wantErr  bool
		errCheck func(t *testing.T, err error)
	}{
		{
			name: "valid recipient",
			id:   "user123",
			contacts: map[string]string{
				"email": "test@example.com",
				"phone": "+1234567890",
			},
			wantErr: false,
		},
		{
			name: "valid recipient with single contact",
			id:   "user456",
			contacts: map[string]string{
				"email": "user@domain.com",
			},
			wantErr: false,
		},
		{
			name:     "empty id",
			id:       "",
			contacts: map[string]string{"email": "test@example.com"},
			wantErr:  true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "recipient id is empty")
			},
		},
		{
			name:     "nil contacts",
			id:       "user789",
			contacts: nil,
			wantErr:  true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "contacts is empty")
			},
		},
		{
			name:     "empty contacts map",
			id:       "user000",
			contacts: map[string]string{},
			wantErr:  true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "contacts is empty")
			},
		},
		{
			name: "whitespace only id",
			id:   "   ",
			contacts: map[string]string{
				"email": "test@example.com",
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "recipient id is empty")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(tc *testing.T) {
			tc.Parallel()

			recipient, err := entities.NewRecipient(tt.id, tt.contacts)

			if tt.wantErr {
				require.Error(tc, err)
				require.Nil(tc, recipient)
				if tt.errCheck != nil {
					tt.errCheck(tc, err)
				}
				return
			}

			require.NoError(tc, err)
			require.NotNil(tc, recipient)
			require.Equal(tc, tt.id, recipient.ID)
			require.Equal(tc, tt.contacts, recipient.Contacts)
		})
	}
}
