package cases_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/reporting/internal/cases"
	"github.com/parta4ok/kvs/reporting/internal/cases/testdata"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	entitiesTestdata "github.com/parta4ok/kvs/reporting/internal/entities/testdata"
)

func TestReportingService_GetPassedTopicsByGroups_PopulatesPassedTopics(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	broker := testdata.NewMockMessageBroker(ctrl)
	authClient := testdata.NewMockAuthClient(ctrl)
	questionClient := testdata.NewMockQuestionClient(ctrl)
	representer := entitiesTestdata.NewMockRepresenter(ctrl)

	students := []entities.Student{
		{
			ID:       "student1",
			Name:     "Test Student 1",
			FullName: "Test Student One",
			Group:    entities.Group{ID: "group1", Title: "Group 1"},
		},
		{
			ID:       "student2",
			Name:     "Test Student 2",
			FullName: "Test Student Two",
			Group:    entities.Group{ID: "group1", Title: "Group 1"},
		},
	}
	authClient.EXPECT().GetMentorGroups(gomock.Any(), "mentor1").Return(students, nil)

	passedTopics := map[string][]entities.Topic{
		"student1": {{ID: "topic1", Title: "Topic 1"}, {ID: "topic2", Title: "Topic 2"}},
		"student2": {},
	}
	questionClient.EXPECT().GetPassedStudentsTopics(gomock.Any(), gomock.Any()).
		Return(passedTopics, nil)

	mentor := &entities.User{ID: "mentor1", Name: "Test Mentor"}
	authClient.EXPECT().GetUserByID(gomock.Any(), "mentor1").Return(mentor, nil)

	done := make(chan []entities.Student, 1)
	representer.EXPECT().CovertToFormat(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ entities.Format, report entities.Report) ([]byte, error) {
			reportedStudents, _ := report.GetReport().([]entities.Student)
			done <- reportedStudents
			return []byte("report"), nil
		})

	broker.EXPECT().ReportEvent(gomock.Any(), gomock.Any()).Return(nil)

	service, err := cases.NewReportingService(
		broker, representer, "json", authClient, questionClient, 1, 10, time.Second*3,
	)
	require.NoError(t, err)
	t.Cleanup(func() { service.Stop() })

	err = service.GetPassedTopicsByGroups(context.Background(), "mentor1")
	require.NoError(t, err)

	select {
	case reportedStudents := <-done:
		require.Len(t, reportedStudents, 2)

		byID := make(map[string]entities.Student, len(reportedStudents))
		for _, s := range reportedStudents {
			byID[s.ID] = s
		}

		require.Len(t, byID["student1"].PassedTopics, 2)
		require.Empty(t, byID["student2"].PassedTopics)
	case <-time.After(5 * time.Second):
		require.Fail(t, "representer was not called within timeout")
	}
}
