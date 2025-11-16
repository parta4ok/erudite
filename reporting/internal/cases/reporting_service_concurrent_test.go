//go:build KVS_TEST_CONCURRENT

package cases_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/reporting/internal/cases"
	"github.com/parta4ok/kvs/reporting/internal/cases/testdata"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	entitiesTestdata "github.com/parta4ok/kvs/reporting/internal/entities/testdata"
)

func TestReportingService_ConcurrentExecution(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		workersLimit int
		totalTasks   int
		taskDuration time.Duration
		timeout      time.Duration
	}{
		{
			name:         "respects_worker_limit_3",
			workersLimit: 3,
			totalTasks:   6,
			taskDuration: 50 * time.Millisecond,
			timeout:      10 * time.Second,
		},
		{
			name:         "respects_worker_limit_2",
			workersLimit: 2,
			totalTasks:   4,
			taskDuration: 30 * time.Millisecond,
			timeout:      8 * time.Second,
		},
		{
			name:         "single_worker",
			workersLimit: 1,
			totalTasks:   3,
			taskDuration: 20 * time.Millisecond,
			timeout:      5 * time.Second,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			broker, authClient, questionClient, representer := setupConcurrencyMocks(
				t, ctrl, tc.totalTasks, tc.taskDuration,
			)

			service, err := cases.NewReportingService(
				broker, representer, "json",
				authClient, questionClient, tc.workersLimit,
			)
			require.NoError(t, err)
			t.Cleanup(func() { service.Stop() })

			var wg sync.WaitGroup
			ctx := context.Background()

			for i := 0; i < tc.totalTasks; i++ {
				wg.Add(1)
				mentorID := generateMentorID(i)

				go func(id string) {
					defer wg.Done()
					err := service.GetPassedTopicsByGroups(ctx, id)
					require.NoError(t, err)
				}(mentorID)
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				wg.Wait()
			}()

			select {
			case <-done:
				time.Sleep(500 * time.Millisecond)
			case <-time.After(tc.timeout):
				require.Fail(t, "Tasks did not complete within timeout")
			}
		})
	}
}

func TestReportingService_RaceConditions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		numGoroutines int
		workersLimit  int
		timeout       time.Duration
	}{
		{
			name:          "concurrent_calls_20",
			numGoroutines: 20,
			workersLimit:  5,
			timeout:       15 * time.Second,
		},
		{
			name:          "concurrent_calls_50",
			numGoroutines: 50,
			workersLimit:  3,
			timeout:       20 * time.Second,
		},
		{
			name:          "high_concurrent_calls_100",
			numGoroutines: 100,
			workersLimit:  10,
			timeout:       30 * time.Second,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			broker, authClient, questionClient, representer := setupRaceConditionMocks(
				t, ctrl, tc.numGoroutines,
			)

			service, err := cases.NewReportingService(
				broker, representer, "json",
				authClient, questionClient, tc.workersLimit,
			)
			require.NoError(t, err)
			t.Cleanup(func() { service.Stop() })

			var successfulCalls atomic.Int64
			var wg sync.WaitGroup
			ctx := context.Background()

			for i := 0; i < tc.numGoroutines; i++ {
				wg.Add(1)
				mentorID := generateMentorID(i)

				go func(id string) {
					defer wg.Done()
					err := service.GetPassedTopicsByGroups(ctx, id)
					require.NoError(t, err)
					successfulCalls.Add(1)
				}(mentorID)
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				wg.Wait()
			}()

			select {
			case <-done:
				time.Sleep(500 * time.Millisecond) // Allow internal tasks to complete
				require.Equal(t, int64(tc.numGoroutines), successfulCalls.Load())
			case <-time.After(tc.timeout):
				require.Fail(t, "Race condition test did not complete within timeout")
			}
		})
	}
}

func TestReportingService_GracefulShutdown(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		workersLimit int
		numTasks     int
		taskDuration time.Duration
		minStopTime  time.Duration
		timeout      time.Duration
	}{
		{
			name:         "shutdown_with_running_tasks",
			workersLimit: 3,
			numTasks:     6,
			taskDuration: 300 * time.Millisecond,
			minStopTime:  200 * time.Millisecond,
			timeout:      15 * time.Second,
		},
		{
			name:         "shutdown_with_slow_tasks",
			workersLimit: 2,
			numTasks:     4,
			taskDuration: 500 * time.Millisecond,
			minStopTime:  400 * time.Millisecond,
			timeout:      20 * time.Second,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			var tasksInProgress atomic.Int32
			var tasksCompleted atomic.Int32

			broker, authClient, questionClient, representer := setupGracefulShutdownMocks(
				t, ctrl, tc.taskDuration, &tasksInProgress, &tasksCompleted,
			)

			service, err := cases.NewReportingService(
				broker, representer, "json",
				authClient, questionClient, tc.workersLimit,
			)
			require.NoError(t, err)

			ctx := context.Background()

			for i := 0; i < tc.numTasks; i++ {
				mentorID := generateMentorID(i)
				go func(id string) {
					service.GetPassedTopicsByGroups(ctx, id)
				}(mentorID)
			}

			startTime := time.Now()
			for tasksInProgress.Load() < int32(tc.workersLimit) {
				require.Less(t, time.Since(startTime), 10*time.Second, "Tasks did not start within timeout")
				time.Sleep(50 * time.Millisecond)
			}

			stopStart := time.Now()
			err = service.Stop()
			stopDuration := time.Since(stopStart)

			require.NoError(t, err)
			require.GreaterOrEqual(t, stopDuration, tc.minStopTime)
			require.Equal(t, tasksInProgress.Load(), tasksCompleted.Load())
		})
	}
}

func TestReportingService_StopIdempotency(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		numStopCalls int
		workersLimit int
		timeout      time.Duration
	}{
		{
			name:         "multiple_stop_calls_10",
			numStopCalls: 10,
			workersLimit: 3,
			timeout:      10 * time.Second,
		},
		{
			name:         "concurrent_stop_calls_20",
			numStopCalls: 20,
			workersLimit: 5,
			timeout:      15 * time.Second,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			service := createTestServiceBasic(t, ctrl, tc.workersLimit)

			var wg sync.WaitGroup
			errors := make(chan error, tc.numStopCalls)

			for i := 0; i < tc.numStopCalls; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := service.Stop()
					errors <- err
				}()
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				wg.Wait()
				close(errors)
			}()

			select {
			case <-done:
				var successCount, errorCount int
				for err := range errors {
					if err == nil {
						successCount++
					} else {
						errorCount++
					}
				}

				require.Equal(t, 1, successCount)
				require.Equal(t, tc.numStopCalls-1, errorCount)
			case <-time.After(tc.timeout):
				require.Fail(t, "Stop idempotency test did not complete within timeout")
			}
		})
	}
}

func TestReportingService_ErrorHandling(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		numTasks     int
		errorStage   string
		workersLimit int
		timeout      time.Duration
	}{
		{
			name:         "auth_client_errors",
			numTasks:     5,
			errorStage:   "auth",
			workersLimit: 2,
			timeout:      10 * time.Second,
		},
		{
			name:         "question_client_errors",
			numTasks:     3,
			errorStage:   "question",
			workersLimit: 1,
			timeout:      8 * time.Second,
		},
		{
			name:         "broker_errors",
			numTasks:     4,
			errorStage:   "broker",
			workersLimit: 3,
			timeout:      10 * time.Second,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			broker, authClient, questionClient, representer := setupErrorHandlingMocks(
				t, ctrl, tc.errorStage, tc.numTasks,
			)

			service, err := cases.NewReportingService(
				broker, representer, "json",
				authClient, questionClient, tc.workersLimit,
			)
			require.NoError(t, err)
			t.Cleanup(func() { service.Stop() })

			var wg sync.WaitGroup
			ctx := context.Background()

			for i := 0; i < tc.numTasks; i++ {
				wg.Add(1)
				mentorID := generateMentorID(i)

				go func(id string) {
					defer wg.Done()
					err := service.GetPassedTopicsByGroups(ctx, id)
					require.NoError(t, err)
				}(mentorID)
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				wg.Wait()
			}()

			select {
			case <-done:
				time.Sleep(500 * time.Millisecond)
			case <-time.After(tc.timeout):
				require.Fail(t, "Error handling test did not complete within timeout")
			}
		})
	}
}

func createTestServiceBasic(t *testing.T, ctrl *gomock.Controller, workersLimit int) *cases.ReportingService {
	t.Helper()

	broker := testdata.NewMockMessageBroker(ctrl)
	authClient := testdata.NewMockAuthClient(ctrl)
	questionClient := testdata.NewMockQuestionClient(ctrl)
	representer := entitiesTestdata.NewMockRepresenter(ctrl)

	service, err := cases.NewReportingService(
		broker, representer, "json",
		authClient, questionClient, workersLimit,
	)
	require.NoError(t, err)

	return service
}

func generateMentorID(index int) string {
	return "mentor-" + string(rune('0'+index%10))
}

func setupConcurrencyMocks(
	t *testing.T,
	ctrl *gomock.Controller,
	totalTasks int,
	taskDuration time.Duration,
) (*testdata.MockMessageBroker, *testdata.MockAuthClient, *testdata.MockQuestionClient, *entitiesTestdata.MockRepresenter) {
	t.Helper()

	broker := testdata.NewMockMessageBroker(ctrl)
	authClient := testdata.NewMockAuthClient(ctrl)
	questionClient := testdata.NewMockQuestionClient(ctrl)
	representer := entitiesTestdata.NewMockRepresenter(ctrl)

	representer.EXPECT().CovertToFormat(gomock.Any(), gomock.Any()).Return([]byte("test report"), nil).AnyTimes()

	students := []entities.Student{
		{
			ID:       "student1",
			Name:     "Test Student 1",
			FullName: "Test Student 1",
			Group:    entities.Group{ID: "group1", Title: "Group 1"},
		},
	}

	authClient.EXPECT().GetMentorGroups(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, mentorID string) ([]entities.Student, error) {
			time.Sleep(taskDuration)
			return students, nil
		}).Times(totalTasks)

	passedTopics := map[string][]entities.Topic{
		"student1": {{ID: "topic1", Title: "Topic 1"}},
	}
	questionClient.EXPECT().GetPassedStudentsTopics(gomock.Any(), gomock.Any()).
		Return(passedTopics, nil).Times(totalTasks)

	user := &entities.User{ID: "mentor1", Name: "Test Mentor", Contacts: map[string]string{"email": "test@test.com"}}
	authClient.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).
		Return(user, nil).Times(totalTasks)

	broker.EXPECT().ReportEvent(gomock.Any(), gomock.Any()).
		Return(nil).Times(totalTasks)

	return broker, authClient, questionClient, representer
}

func setupRaceConditionMocks(
	t *testing.T,
	ctrl *gomock.Controller,
	numGoroutines int,
) (*testdata.MockMessageBroker, *testdata.MockAuthClient, *testdata.MockQuestionClient, *entitiesTestdata.MockRepresenter) {
	t.Helper()

	broker := testdata.NewMockMessageBroker(ctrl)
	authClient := testdata.NewMockAuthClient(ctrl)
	questionClient := testdata.NewMockQuestionClient(ctrl)
	representer := entitiesTestdata.NewMockRepresenter(ctrl)

	representer.EXPECT().CovertToFormat(gomock.Any(), gomock.Any()).Return([]byte("test report"), nil).AnyTimes()

	students := []entities.Student{
		{
			ID:       "student1",
			Name:     "Test Student 1",
			FullName: "Test Student 1",
			Group:    entities.Group{ID: "group1", Title: "Group 1"},
		},
		{
			ID:       "student2",
			Name:     "Test Student 2",
			FullName: "Test Student 2",
			Group:    entities.Group{ID: "group2", Title: "Group 2"},
		},
	}
	authClient.EXPECT().GetMentorGroups(gomock.Any(), gomock.Any()).
		Return(students, nil).Times(numGoroutines)

	passedTopics := map[string][]entities.Topic{
		"student1": {{ID: "topic1", Title: "Topic 1"}},
		"student2": {{ID: "topic2", Title: "Topic 2"}},
	}
	questionClient.EXPECT().GetPassedStudentsTopics(gomock.Any(), gomock.Any()).
		Return(passedTopics, nil).Times(numGoroutines)

	user := &entities.User{ID: "mentor1", Name: "Test Mentor", Contacts: map[string]string{"email": "test@test.com"}}
	authClient.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).
		Return(user, nil).Times(numGoroutines)

	broker.EXPECT().ReportEvent(gomock.Any(), gomock.Any()).
		Return(nil).Times(numGoroutines)

	return broker, authClient, questionClient, representer
}

func setupGracefulShutdownMocks(
	t *testing.T,
	ctrl *gomock.Controller,
	taskDuration time.Duration,
	tasksInProgress *atomic.Int32,
	tasksCompleted *atomic.Int32,
) (*testdata.MockMessageBroker, *testdata.MockAuthClient, *testdata.MockQuestionClient, *entitiesTestdata.MockRepresenter) {
	t.Helper()

	broker := testdata.NewMockMessageBroker(ctrl)
	authClient := testdata.NewMockAuthClient(ctrl)
	questionClient := testdata.NewMockQuestionClient(ctrl)
	representer := entitiesTestdata.NewMockRepresenter(ctrl)

	representer.EXPECT().CovertToFormat(gomock.Any(), gomock.Any()).Return([]byte("test report"), nil).AnyTimes()

	students := []entities.Student{
		{
			ID:       "student1",
			Name:     "Test Student 1",
			FullName: "Test Student 1",
			Group:    entities.Group{ID: "group1", Title: "Group 1"},
		},
	}
	authClient.EXPECT().GetMentorGroups(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mentorID string) ([]entities.Student, error) {
			tasksInProgress.Add(1)
			time.Sleep(taskDuration)
			tasksCompleted.Add(1)
			return students, nil
		}).AnyTimes()

	passedTopics := map[string][]entities.Topic{
		"student1": {{ID: "topic1", Title: "Topic 1"}},
	}
	questionClient.EXPECT().GetPassedStudentsTopics(gomock.Any(), gomock.Any()).
		Return(passedTopics, nil).AnyTimes()

	user := &entities.User{ID: "mentor1", Name: "Test Mentor", Contacts: map[string]string{"email": "test@test.com"}}
	authClient.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).
		Return(user, nil).AnyTimes()

	broker.EXPECT().ReportEvent(gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	return broker, authClient, questionClient, representer
}

func setupErrorHandlingMocks(
	t *testing.T,
	ctrl *gomock.Controller,
	errorStage string,
	numTasks int,
) (*testdata.MockMessageBroker, *testdata.MockAuthClient, *testdata.MockQuestionClient, *entitiesTestdata.MockRepresenter) {
	t.Helper()

	broker := testdata.NewMockMessageBroker(ctrl)
	authClient := testdata.NewMockAuthClient(ctrl)
	questionClient := testdata.NewMockQuestionClient(ctrl)
	representer := entitiesTestdata.NewMockRepresenter(ctrl)

	representer.EXPECT().CovertToFormat(gomock.Any(), gomock.Any()).Return([]byte("test report"), nil).AnyTimes()

	switch errorStage {
	case "auth":
		authClient.EXPECT().GetMentorGroups(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("auth service error")).Times(numTasks)
	case "question":
		students := []entities.Student{
			{
				ID:       "student1",
				Name:     "Test Student 1",
				FullName: "Test Student 1",
				Group:    entities.Group{ID: "group1", Title: "Group 1"},
			},
		}
		authClient.EXPECT().GetMentorGroups(gomock.Any(), gomock.Any()).
			Return(students, nil).Times(numTasks)
		questionClient.EXPECT().GetPassedStudentsTopics(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("question service error")).Times(numTasks)
	case "broker":
		students := []entities.Student{
			{
				ID:       "student1",
				Name:     "Test Student 1",
				FullName: "Test Student 1",
				Group:    entities.Group{ID: "group1", Title: "Group 1"},
			},
		}
		authClient.EXPECT().GetMentorGroups(gomock.Any(), gomock.Any()).
			Return(students, nil).Times(numTasks)
		user := &entities.User{ID: "mentor1", Name: "Test Mentor", Contacts: map[string]string{"email": "test@test.com"}}
		authClient.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).
			Return(user, nil).Times(numTasks)
		passedTopics := map[string][]entities.Topic{
			"student1": {{ID: "topic1", Title: "Topic 1"}},
		}
		questionClient.EXPECT().GetPassedStudentsTopics(gomock.Any(), gomock.Any()).
			Return(passedTopics, nil).Times(numTasks)
		broker.EXPECT().ReportEvent(gomock.Any(), gomock.Any()).
			Return(errors.New("broker error")).Times(numTasks)
	}

	return broker, authClient, questionClient, representer
}
