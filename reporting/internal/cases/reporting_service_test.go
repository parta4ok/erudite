package cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/reporting/internal/cases"
	"github.com/parta4ok/kvs/reporting/internal/cases/testdata"
	"github.com/parta4ok/kvs/reporting/internal/entities"
)

func TestNewReportingService_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	broker := testdata.NewMockMessageBroker(ctrl)
	authClient := testdata.NewMockAuthClient(ctrl)
	questionClient := testdata.NewMockQuestionClient(ctrl)
	representer := testdata.NewMockRepresenter(ctrl)
	representers := []cases.Representer{representer}

	service, err := cases.NewReportingService(broker, representers, authClient, questionClient, 3)

	require.NoError(t, err)
	require.NotNil(t, service)

	service.Stop()
}

func TestNewReportingService_ValidationErrors(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	broker := testdata.NewMockMessageBroker(ctrl)
	authClient := testdata.NewMockAuthClient(ctrl)
	questionClient := testdata.NewMockQuestionClient(ctrl)
	representer := testdata.NewMockRepresenter(ctrl)
	representers := []cases.Representer{representer}

	testCases := []struct {
		name          string
		setupMocks    func() (cases.MessageBroker, []cases.Representer, cases.AuthClient, cases.QuestionClient, int)
		expectedError string
	}{
		{
			name: "nil_broker",
			setupMocks: func() (cases.MessageBroker, []cases.Representer, cases.AuthClient, cases.QuestionClient, int) {
				return nil, representers, authClient, questionClient, 3
			},
			expectedError: "broker not set",
		},
		{
			name: "nil_representer",
			setupMocks: func() (cases.MessageBroker, []cases.Representer, cases.AuthClient, cases.QuestionClient, int) {
				invalidRepresenters := []cases.Representer{nil}
				return broker, invalidRepresenters, authClient, questionClient, 3
			},
			expectedError: "one or more representer not set",
		},
		{
			name: "nil_auth_client",
			setupMocks: func() (cases.MessageBroker, []cases.Representer, cases.AuthClient, cases.QuestionClient, int) {
				return broker, representers, nil, questionClient, 3
			},
			expectedError: "auth client not set",
		},
		{
			name: "nil_question_client",
			setupMocks: func() (cases.MessageBroker, []cases.Representer, cases.AuthClient, cases.QuestionClient, int) {
				return broker, representers, authClient, nil, 3
			},
			expectedError: "question client not set",
		},
		{
			name: "zero_workers_limit",
			setupMocks: func() (cases.MessageBroker, []cases.Representer, cases.AuthClient, cases.QuestionClient, int) {
				return broker, representers, authClient, questionClient, 0
			},
			expectedError: "workers limit must be greater than 0",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			broker, representers, authClient, questionClient, workersLimit := tc.setupMocks()

			service, err := cases.NewReportingService(broker, representers, authClient, questionClient, workersLimit)

			require.Error(t, err)
			require.Nil(t, service)
			require.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

func TestReportingService_GetPassedTopicsByGroups(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mentorID := "test-mentor-123"
	reportFormat := "json"

	testCases := []struct {
		name          string
		mentorID      string
		reportFormat  string
		setupMocks    func() (*testdata.MockMessageBroker, *testdata.MockAuthClient, *testdata.MockQuestionClient, *testdata.MockRepresenter)
		expectedError string
	}{
		{
			name:         "success",
			mentorID:     mentorID,
			reportFormat: reportFormat,
			setupMocks: func() (*testdata.MockMessageBroker, *testdata.MockAuthClient, *testdata.MockQuestionClient, *testdata.MockRepresenter) {
				broker := testdata.NewMockMessageBroker(ctrl)
				authClient := testdata.NewMockAuthClient(ctrl)
				questionClient := testdata.NewMockQuestionClient(ctrl)
				representer := testdata.NewMockRepresenter(ctrl)

				representer.EXPECT().GetReportFormat().Return("json")

				students := []entities.Student{
					{
						ID:       "student1",
						Name:     "Test Student 1",
						FullName: "Test Student 1 Full",
						Group:    entities.Group{ID: "group1", Title: "Group 1"},
					},
				}
				authClient.EXPECT().GetMentorGroups(ctx, mentorID).Return(students, nil)

				passedTopics := map[string][]entities.Topic{
					"student1": {{ID: "topic1", Title: "Topic 1"}},
				}
				questionClient.EXPECT().GetPassedStudentsTopics(ctx, []string{"student1"}).Return(passedTopics, nil)

				broker.EXPECT().ReportEvent(ctx, gomock.Any(), representer).Return(nil)

				return broker, authClient, questionClient, representer
			},
			expectedError: "",
		},
		{
			name:         "unknown_report_format",
			mentorID:     mentorID,
			reportFormat: "unknown",
			setupMocks: func() (*testdata.MockMessageBroker, *testdata.MockAuthClient, *testdata.MockQuestionClient, *testdata.MockRepresenter) {
				broker := testdata.NewMockMessageBroker(ctrl)
				authClient := testdata.NewMockAuthClient(ctrl)
				questionClient := testdata.NewMockQuestionClient(ctrl)
				representer := testdata.NewMockRepresenter(ctrl)

				representer.EXPECT().GetReportFormat().Return("json")

				return broker, authClient, questionClient, representer
			},
			expectedError: "unknown report format",
		},
		{
			name:         "auth_client_error",
			mentorID:     mentorID,
			reportFormat: reportFormat,
			setupMocks: func() (*testdata.MockMessageBroker, *testdata.MockAuthClient, *testdata.MockQuestionClient, *testdata.MockRepresenter) {
				broker := testdata.NewMockMessageBroker(ctrl)
				authClient := testdata.NewMockAuthClient(ctrl)
				questionClient := testdata.NewMockQuestionClient(ctrl)
				representer := testdata.NewMockRepresenter(ctrl)

				representer.EXPECT().GetReportFormat().Return("json")
				authClient.EXPECT().GetMentorGroups(ctx, mentorID).Return(nil, errors.New("auth service error"))

				return broker, authClient, questionClient, representer
			},
			expectedError: "",
		},
		{
			name:         "question_client_error",
			mentorID:     mentorID,
			reportFormat: reportFormat,
			setupMocks: func() (*testdata.MockMessageBroker, *testdata.MockAuthClient, *testdata.MockQuestionClient, *testdata.MockRepresenter) {
				broker := testdata.NewMockMessageBroker(ctrl)
				authClient := testdata.NewMockAuthClient(ctrl)
				questionClient := testdata.NewMockQuestionClient(ctrl)
				representer := testdata.NewMockRepresenter(ctrl)

				representer.EXPECT().GetReportFormat().Return("json")

				students := []entities.Student{
					{
						ID:       "student1",
						Name:     "Test Student 1",
						FullName: "Test Student 1 Full",
						Group:    entities.Group{ID: "group1", Title: "Group 1"},
					},
				}
				authClient.EXPECT().GetMentorGroups(ctx, mentorID).Return(students, nil)
				questionClient.EXPECT().GetPassedStudentsTopics(ctx, []string{"student1"}).Return(nil, errors.New("question service error"))

				return broker, authClient, questionClient, representer
			},
			expectedError: "",
		},
		{
			name:         "broker_error",
			mentorID:     mentorID,
			reportFormat: reportFormat,
			setupMocks: func() (*testdata.MockMessageBroker, *testdata.MockAuthClient, *testdata.MockQuestionClient, *testdata.MockRepresenter) {
				broker := testdata.NewMockMessageBroker(ctrl)
				authClient := testdata.NewMockAuthClient(ctrl)
				questionClient := testdata.NewMockQuestionClient(ctrl)
				representer := testdata.NewMockRepresenter(ctrl)

				representer.EXPECT().GetReportFormat().Return("json")

				students := []entities.Student{
					{
						ID:       "student1",
						Name:     "Test Student 1",
						FullName: "Test Student 1 Full",
						Group:    entities.Group{ID: "group1", Title: "Group 1"},
					},
				}
				authClient.EXPECT().GetMentorGroups(ctx, mentorID).Return(students, nil)

				passedTopics := map[string][]entities.Topic{
					"student1": {{ID: "topic1", Title: "Topic 1"}},
				}
				questionClient.EXPECT().GetPassedStudentsTopics(ctx, []string{"student1"}).Return(passedTopics, nil)

				broker.EXPECT().ReportEvent(ctx, gomock.Any(), representer).Return(errors.New("broker error"))

				return broker, authClient, questionClient, representer
			},
			expectedError: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			broker, authClient, questionClient, representer := tc.setupMocks()

			service, err := cases.NewReportingService(
				broker, []cases.Representer{representer},
				authClient, questionClient, 2,
			)
			require.NoError(t, err)
			defer service.Stop()

			err = service.GetPassedTopicsByGroups(ctx, tc.mentorID, tc.reportFormat)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReportingService_GetPassedTopicsByGroups_ServiceStopped(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	broker := testdata.NewMockMessageBroker(ctrl)
	authClient := testdata.NewMockAuthClient(ctrl)
	questionClient := testdata.NewMockQuestionClient(ctrl)
	representer := testdata.NewMockRepresenter(ctrl)

	service, err := cases.NewReportingService(
		broker, []cases.Representer{representer},
		authClient, questionClient, 1,
	)
	require.NoError(t, err)

	err = service.Stop()
	require.NoError(t, err)

	ctx := context.Background()
	err = service.GetPassedTopicsByGroups(ctx, "mentor1", "json")

	require.Error(t, err)
	require.Contains(t, err.Error(), "service has stopped")
}

func TestReportingService_Stop(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name          string
		setupService  func() *cases.ReportingService
		expectedError string
	}{
		{
			name: "success",
			setupService: func() *cases.ReportingService {
				broker := testdata.NewMockMessageBroker(ctrl)
				authClient := testdata.NewMockAuthClient(ctrl)
				questionClient := testdata.NewMockQuestionClient(ctrl)
				representer := testdata.NewMockRepresenter(ctrl)

				service, err := cases.NewReportingService(
					broker, []cases.Representer{representer},
					authClient, questionClient, 2,
				)
				require.NoError(t, err)

				return service
			},
			expectedError: "",
		},
		{
			name: "double_stop",
			setupService: func() *cases.ReportingService {
				broker := testdata.NewMockMessageBroker(ctrl)
				authClient := testdata.NewMockAuthClient(ctrl)
				questionClient := testdata.NewMockQuestionClient(ctrl)
				representer := testdata.NewMockRepresenter(ctrl)

				service, err := cases.NewReportingService(
					broker, []cases.Representer{representer},
					authClient, questionClient, 2,
				)
				require.NoError(t, err)

				err = service.Stop()
				require.NoError(t, err)

				return service
			},
			expectedError: "service already stopped",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := tc.setupService()

			err := service.Stop()

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
