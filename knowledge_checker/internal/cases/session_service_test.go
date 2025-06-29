package cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/knowledge_checker/internal/cases"
	"github.com/parta4ok/kvs/knowledge_checker/internal/cases/testdata"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
	entitiesTestdata "github.com/parta4ok/kvs/knowledge_checker/internal/entities/testdata"
)

func TestNewSessionService_Success(t *testing.T) {

	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := testdata.NewMockStorage(ctrl)
	sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
	generator := entitiesTestdata.NewMockIDGenerator(ctrl)

	service, err := cases.NewSessionService(storage, sessionStorage, generator)

	require.NoError(t, err)
	require.NotNil(t, service)
}

func TestNewSessionService_WithCustomDuration(t *testing.T) {

	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := testdata.NewMockStorage(ctrl)
	sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
	generator := entitiesTestdata.NewMockIDGenerator(ctrl)
	customDuration := time.Minute * 15

	service, err := cases.NewSessionService(storage, sessionStorage, generator,
		cases.WithCustomSessionDuration(customDuration))

	require.NoError(t, err)
	require.NotNil(t, service)
}

func TestNewSessionService_ValidationErrors(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := testdata.NewMockStorage(ctrl)
	sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
	generator := entitiesTestdata.NewMockIDGenerator(ctrl)

	testCases := []struct {
		name          string
		setupMocks    func() (cases.Storage, entities.SessionStorage, entities.IDGenerator)
		expectedError string
	}{
		{
			name: "nil_storage",
			setupMocks: func() (cases.Storage, entities.SessionStorage, entities.IDGenerator) {
				return nil, sessionStorage, generator
			},
			expectedError: "storage not set",
		},
		{
			name: "nil_session_storage",
			setupMocks: func() (cases.Storage, entities.SessionStorage, entities.IDGenerator) {
				return storage, nil, generator
			},
			expectedError: "session storage not set",
		},
		{
			name: "nil_generator",
			setupMocks: func() (cases.Storage, entities.SessionStorage, entities.IDGenerator) {
				return storage, sessionStorage, nil
			},
			expectedError: "generator not set",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			storage, sessionStorage, generator := tc.setupMocks()

			service, err := cases.NewSessionService(storage, sessionStorage, generator)

			require.Error(t, err)
			require.Nil(t, service)
			require.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

func TestSessionService_ShowTopics(t *testing.T) {

	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name       string
		setupMocks func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
			*entitiesTestdata.MockIDGenerator)
		expectedTopics []string
		expectedError  string
	}{
		{
			name: "success",
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				storage.EXPECT().GetTopics(gomock.Any()).Return([]string{"Go", "Databases"}, nil)

				return storage, sessionStorage, generator
			},
			expectedTopics: []string{"Go", "Databases"},
			expectedError:  "",
		},
		{
			name: "storage_error",
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				storage.EXPECT().GetTopics(gomock.Any()).Return(nil, errors.New("database error"))

				return storage, sessionStorage, generator
			},
			expectedTopics: nil,
			expectedError:  "GetTopics",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			storage, sessionStorage, generator := tc.setupMocks()

			service, err := cases.NewSessionService(storage, sessionStorage, generator)
			require.NoError(t, err)

			ctx := context.Background()
			topics, err := service.ShowTopics(ctx)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
				require.Nil(t, topics)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedTopics, topics)
			}
		})
	}
}

func TestSessionService_CreateSession(t *testing.T) {

	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name       string
		userID     uint64
		topics     []string
		setupMocks func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
			*entitiesTestdata.MockIDGenerator)
		expectedSessionID uint64
		expectedError     string
	}{
		{
			name:   "success",
			userID: 1,
			topics: []string{"Go"},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				generator.EXPECT().GenerateID().Return(uint64(123))
				sessionStorage.EXPECT().IsDailySessionLimitReached(gomock.Any(), uint64(1),
					[]string{"Go"}).Return(false, nil)

				mockQuestion := entitiesTestdata.NewMockQuestion(ctrl)
				mockQuestion.EXPECT().ID().Return(uint64(1)).AnyTimes()
				storage.EXPECT().GetQuesions(gomock.Any(), []string{"Go"}).Return(
					[]entities.Question{mockQuestion}, nil)

				storage.EXPECT().StoreSession(gomock.Any(), gomock.Any()).Return(nil)

				return storage, sessionStorage, generator
			},
			expectedSessionID: 123,
			expectedError:     "",
		},
		{
			name:   "daily_limit_reached",
			userID: 1,
			topics: []string{"Go"},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				generator.EXPECT().GenerateID().Return(uint64(123))
				sessionStorage.EXPECT().IsDailySessionLimitReached(gomock.Any(), uint64(1),
					[]string{"Go"}).Return(true, nil)

				return storage, sessionStorage, generator
			},
			expectedSessionID: 0,
			expectedError:     "creating new session for this user",
		},
		{
			name:   "session_storage_error",
			userID: 1,
			topics: []string{"Go"},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				generator.EXPECT().GenerateID().Return(uint64(123))
				sessionStorage.EXPECT().IsDailySessionLimitReached(gomock.Any(), uint64(1),
					[]string{"Go"}).Return(false, errors.New("storage error"))

				return storage, sessionStorage, generator
			},
			expectedSessionID: 0,
			expectedError:     "IsDailySessionLimitReached",
		},
		{
			name:   "get_questions_error",
			userID: 1,
			topics: []string{"Go"},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				generator.EXPECT().GenerateID().Return(uint64(123))
				sessionStorage.EXPECT().IsDailySessionLimitReached(gomock.Any(),
					uint64(1), []string{"Go"}).Return(false, nil)
				storage.EXPECT().GetQuesions(gomock.Any(), []string{"Go"}).Return(nil,
					errors.New("questions error"))

				return storage, sessionStorage, generator
			},
			expectedSessionID: 0,
			expectedError:     "GetQuesions",
		},
		{
			name:   "store_session_error",
			userID: 1,
			topics: []string{"Go"},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				generator.EXPECT().GenerateID().Return(uint64(123))
				sessionStorage.EXPECT().IsDailySessionLimitReached(gomock.Any(), uint64(1),
					[]string{"Go"}).Return(false, nil)

				mockQuestion := entitiesTestdata.NewMockQuestion(ctrl)
				mockQuestion.EXPECT().ID().Return(uint64(1)).AnyTimes()
				storage.EXPECT().GetQuesions(gomock.Any(), []string{"Go"}).Return(
					[]entities.Question{mockQuestion}, nil)
				storage.EXPECT().StoreSession(gomock.Any(), gomock.Any()).Return(
					errors.New("store error"))

				return storage, sessionStorage, generator
			},
			expectedSessionID: 0,
			expectedError:     "StoreSession",
		},
		{
			name:   "new_session_error_invalid_user_id",
			userID: 0,
			topics: []string{"Go"},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				return storage, sessionStorage, generator
			},
			expectedSessionID: 0,
			expectedError:     "NewSession",
		},
		{
			name:   "new_session_error_empty_topics",
			userID: 1,
			topics: []string{},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				return storage, sessionStorage, generator
			},
			expectedSessionID: 0,
			expectedError:     "NewSession",
		},
		{
			name:   "set_questions_error",
			userID: 1,
			topics: []string{"Go"},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				generator.EXPECT().GenerateID().Return(uint64(123))
				sessionStorage.EXPECT().IsDailySessionLimitReached(gomock.Any(), uint64(1),
					[]string{"Go"}).Return(false, nil)

				storage.EXPECT().GetQuesions(gomock.Any(), []string{"Go"}).Return(
					[]entities.Question{}, nil)

				return storage, sessionStorage, generator
			},
			expectedSessionID: 0,
			expectedError:     "SetQuestions",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			storage, sessionStorage, generator := tc.setupMocks()

			service, err := cases.NewSessionService(storage, sessionStorage, generator)
			require.NoError(t, err)

			ctx := context.Background()
			sessionID, questions, err := service.CreateSession(ctx, tc.userID, tc.topics)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
				require.Equal(t, uint64(0), sessionID)
				require.Nil(t, questions)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedSessionID, sessionID)
				require.NotNil(t, questions)
				require.Len(t, questions, 1)
			}
		})
	}
}

func TestSessionService_CompleteSession(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name       string
		sessionID  uint64
		answers    []*entities.UserAnswer
		setupMocks func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
			*entitiesTestdata.MockIDGenerator)
		expectedResult *entities.SessionResult
		expectedError  string
	}{
		{
			name:      "success",
			sessionID: 123,
			answers:   []*entities.UserAnswer{},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				mockState := entitiesTestdata.NewMockSessionState(ctrl)
				session := entities.NewSessionWithCustomState(123, 1, []string{"Go"}, mockState)

				expectedResult := &entities.SessionResult{
					IsSuccess: true,
					Grade:     "100%",
				}

				storage.EXPECT().GetSessionBySessionID(gomock.Any(), uint64(123)).Return(session,
					nil)
				mockState.EXPECT().SetUserAnswer([]*entities.UserAnswer{}).Return(nil)
				mockState.EXPECT().GetSessionResult().Return(expectedResult, nil)
				storage.EXPECT().StoreSession(gomock.Any(), session).Return(nil)

				return storage, sessionStorage, generator
			},
			expectedResult: &entities.SessionResult{
				IsSuccess: true,
				Grade:     "100%",
			},
			expectedError: "",
		},
		{
			name:      "session_not_found",
			sessionID: 999,
			answers:   []*entities.UserAnswer{},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				storage.EXPECT().GetSessionBySessionID(gomock.Any(), uint64(999)).Return(nil,
					errors.New("session not found"))

				return storage, sessionStorage, generator
			},
			expectedResult: nil,
			expectedError:  "GetSessionBySessionID",
		},
		{
			name:      "get_session_result_error",
			sessionID: 123,
			answers:   []*entities.UserAnswer{},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				mockState := entitiesTestdata.NewMockSessionState(ctrl)
				session := entities.NewSessionWithCustomState(123, 1, []string{"Go"}, mockState)

				storage.EXPECT().GetSessionBySessionID(gomock.Any(), uint64(123)).Return(session,
					nil)
				mockState.EXPECT().SetUserAnswer([]*entities.UserAnswer{}).Return(nil)
				mockState.EXPECT().GetSessionResult().Return(nil,
					errors.New("session not completed"))

				return storage, sessionStorage, generator
			},
			expectedResult: nil,
			expectedError:  "GetSessionResult",
		},
		{
			name:      "store_session_error",
			sessionID: 123,
			answers:   []*entities.UserAnswer{},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				mockState := entitiesTestdata.NewMockSessionState(ctrl)
				session := entities.NewSessionWithCustomState(123, 1, []string{"Go"}, mockState)
				expectedResult := &entities.SessionResult{
					IsSuccess: true,
					Grade:     "100%",
				}

				storage.EXPECT().GetSessionBySessionID(gomock.Any(), uint64(123)).Return(session,
					nil)
				mockState.EXPECT().SetUserAnswer([]*entities.UserAnswer{}).Return(nil)
				mockState.EXPECT().GetSessionResult().Return(expectedResult, nil)
				storage.EXPECT().StoreSession(gomock.Any(), session).Return(
					errors.New("store error"))

				return storage, sessionStorage, generator
			},
			expectedResult: nil,
			expectedError:  "StoreSession",
		},
		{
			name:      "set_user_answer_error",
			sessionID: 123,
			answers:   []*entities.UserAnswer{},
			setupMocks: func() (*testdata.MockStorage, *entitiesTestdata.MockSessionStorage,
				*entitiesTestdata.MockIDGenerator) {
				storage := testdata.NewMockStorage(ctrl)
				sessionStorage := entitiesTestdata.NewMockSessionStorage(ctrl)
				generator := entitiesTestdata.NewMockIDGenerator(ctrl)

				mockState := entitiesTestdata.NewMockSessionState(ctrl)
				session := entities.NewSessionWithCustomState(123, 1, []string{"Go"}, mockState)

				storage.EXPECT().GetSessionBySessionID(gomock.Any(), uint64(123)).Return(session,
					nil)
				mockState.EXPECT().SetUserAnswer([]*entities.UserAnswer{}).Return(
					errors.New("invalid answers"))

				return storage, sessionStorage, generator
			},
			expectedResult: nil,
			expectedError:  "SetUserAnswer",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			storage, sessionStorage, generator := tc.setupMocks()

			service, err := cases.NewSessionService(storage, sessionStorage, generator)
			require.NoError(t, err)

			ctx := context.Background()
			result, err := service.CompleteSession(ctx, tc.sessionID, tc.answers)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}
