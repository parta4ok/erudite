package entities_test

import (
	"testing"

	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	"github.com/stretchr/testify/require"
)

func TestNewSessionResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userID    string
		topics    []string
		questions map[string][]string
		answers   map[string][]string
		isExpire  bool
		isSuccess bool
		resume    string
		wantErr   bool
		errCheck  func(t *testing.T, err error)
	}{
		{
			name:   "valid session result",
			userID: "user123",
			topics: []string{"math", "physics"},
			questions: map[string][]string{
				"q1": []string{"1", "2"},
				"q2": []string{"1", "2"},
			},
			answers: map[string][]string{
				"1": {"4"},
				"2": {"9.8 m/s²"},
			},
			isExpire:  false,
			isSuccess: true,
			resume:    "Good performance overall",
			wantErr:   false,
		},
		{
			name:   "valid session result with single topic",
			userID: "user456",
			topics: []string{"chemistry"},
			questions: map[string][]string{
				"q1": []string{"1", "2"},
				"q2": []string{"1", "2"},
			},
			answers: map[string][]string{
				"1": {"water"},
			},
			isExpire:  true,
			isSuccess: false,
			resume:    "Session expired",
			wantErr:   false,
		},
		{
			name:   "empty userID",
			userID: "",
			topics: []string{"math"},
			questions: map[string][]string{
				"q1": []string{"1", "2"},
				"q2": []string{"1", "2"},
			},
			answers: map[string][]string{
				"1": {"4"},
			},
			isExpire:  false,
			isSuccess: true,
			resume:    "Good job",
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "user id is empty")
			},
		},
		{
			name:   "whitespace only userID",
			userID: "   ",
			topics: []string{"math"},
			questions: map[string][]string{
				"q1": []string{"1", "2"},
				"q2": []string{"1", "2"},
			},
			answers: map[string][]string{
				"1": {"4"},
			},
			isExpire:  false,
			isSuccess: true,
			resume:    "Good job",
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "user id is empty")
			},
		},
		{
			name:   "empty topics",
			userID: "user123",
			topics: []string{},
			questions: map[string][]string{
				"q1": []string{"1", "2"},
				"q2": []string{"1", "2"}},
			answers:   map[string][]string{"1": {"answer"}},
			isExpire:  false,
			isSuccess: true,
			resume:    "Good job",
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "topics list is empty")
			},
		},
		{
			name:   "nil topics",
			userID: "user123",
			topics: nil,
			questions: map[string][]string{
				"q1": []string{"1", "2"},
				"q2": []string{"1", "2"},
			},
			answers:   map[string][]string{"1": {"answer"}},
			isExpire:  false,
			isSuccess: true,
			resume:    "Good job",
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "topics list is empty")
			},
		},
		{
			name:      "empty questions",
			userID:    "user123",
			topics:    []string{"math"},
			questions: map[string][]string{},
			answers:   map[string][]string{"1": {"answer"}},
			isExpire:  false,
			isSuccess: true,
			resume:    "Good job",
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "questions list is empty")
			},
		},
		{
			name:      "nil questions",
			userID:    "user123",
			topics:    []string{"math"},
			questions: nil,
			answers:   map[string][]string{"1": {"answer"}},
			isExpire:  false,
			isSuccess: true,
			resume:    "Good job",
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "questions list is empty")
			},
		},
		{
			name:   "empty answers",
			userID: "user123",
			topics: []string{"math"},
			questions: map[string][]string{
				"q1": []string{"1", "2"},
				"q2": []string{"1", "2"},
			},
			answers:   map[string][]string{},
			isExpire:  false,
			isSuccess: true,
			resume:    "Good job",
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "answers list is empty")
			},
		},
		{
			name:   "nil answers",
			userID: "user123",
			topics: []string{"math"},
			questions: map[string][]string{
				"q1": []string{"1", "2"},
				"q2": []string{"1", "2"},
			},
			answers:   nil,
			isExpire:  false,
			isSuccess: true,
			resume:    "Good job",
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "answers list is empty")
			},
		},
		{
			name:   "empty resume",
			userID: "user123",
			topics: []string{"math"},
			questions: map[string][]string{
				"q1": []string{"1", "2"},
				"q2": []string{"1", "2"},
			},
			answers: map[string][]string{
				"1": {"4"},
			},
			isExpire:  false,
			isSuccess: true,
			resume:    "",
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "resume is empty")
			},
		},
		{
			name:   "whitespace only resume",
			userID: "user123",
			topics: []string{"math"},
			questions: map[string][]string{
				"q1": []string{"1", "2"},
				"q2": []string{"1", "2"},
			},
			answers: map[string][]string{
				"1": {"4"},
			},
			isExpire:  false,
			isSuccess: true,
			resume:    "   ",
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				require.Contains(t, err.Error(), "resume is empty")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(tc *testing.T) {
			tc.Parallel()

			sessionResult, err := entities.NewSessionResult(
				tt.userID,
				tt.topics,
				tt.questions,
				tt.answers,
				tt.isExpire,
				tt.isSuccess,
				tt.resume,
			)

			if tt.wantErr {
				require.Error(tc, err)
				require.Nil(tc, sessionResult)
				if tt.errCheck != nil {
					tt.errCheck(tc, err)
				}
				return
			}

			require.NoError(tc, err)
			require.NotNil(tc, sessionResult)
			require.Equal(tc, tt.userID, sessionResult.GetUserID())
			require.Equal(tc, tt.topics, sessionResult.Topics)
			require.Equal(tc, tt.questions, sessionResult.Questions)
			require.Equal(tc, tt.answers, sessionResult.UserAnswer)
			require.Equal(tc, tt.isExpire, sessionResult.IsExpire)
			require.Equal(tc, tt.isSuccess, sessionResult.IsSuccess)
			require.Equal(tc, tt.resume, sessionResult.Resume)
		})
	}
}

func TestSessionResult_GetUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userID         string
		expectedUserID string
	}{
		{
			name:           "regular user id",
			userID:         "user123",
			expectedUserID: "user123",
		},
		{
			name:           "user id with spaces",
			userID:         "user with spaces",
			expectedUserID: "user with spaces",
		},
		{
			name:           "numeric user id",
			userID:         "12345",
			expectedUserID: "12345",
		},
		{
			name:           "uuid user id",
			userID:         "550e8400-e29b-41d4-a716-446655440000",
			expectedUserID: "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(tc *testing.T) {
			tc.Parallel()

			sessionResult, err := entities.NewSessionResult(
				tt.userID,
				[]string{"math"},
				map[string][]string{
					"q1": []string{"1", "2"},
					"q2": []string{"1", "2"},
				},
				map[string][]string{"1": {"4"}},
				false,
				true,
				"Test completed",
			)

			require.NoError(tc, err)
			require.NotNil(tc, sessionResult)

			actualUserID := sessionResult.GetUserID()
			require.Equal(tc, tt.expectedUserID, actualUserID)
		})
	}
}
