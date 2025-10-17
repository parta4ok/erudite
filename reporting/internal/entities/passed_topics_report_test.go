package entities_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/reporting/internal/entities"
)

func TestPassedTopicsReport_Success(t *testing.T) {
	t.Parallel()

	students := []entities.Student{
		{
			ID:       "stu1",
			Name:     "Ivan",
			FullName: "Ivan Petrov",
			Group:    entities.Group{ID: "g1", Title: "Go Май 2024"},
			PassedTopics: []entities.Topic{
				{ID: "1", Title: "Go Базовые типы"},
				{ID: "2", Title: "Go Составные типы"},
			},
		},
		{
			ID:       "stu6",
			Name:     "Svetlana",
			FullName: "Svetlana Petrova",
			Group:    entities.Group{ID: "g2", Title: "Go Август 2024"},
			PassedTopics: []entities.Topic{
				{ID: "4", Title: "Go Cuncurrency"},
				{ID: "1", Title: "Go Базовые типы"},
				{ID: "3", Title: "Go Функции"},
				{ID: "2", Title: "Go Составные типы"},
			},
		},
		{
			ID:       "stu3",
			Name:     "Kirill",
			FullName: "Kirill Sidorov",
			Group:    entities.Group{ID: "g1", Title: "Go Май 2024"},
			PassedTopics: []entities.Topic{
				{ID: "2", Title: "Go Составные типы"},
				{ID: "1", Title: "Go Базовые типы"},
			},
		},
		{
			ID:       "stu4",
			Name:     "Anna",
			FullName: "Anna Volkova",
			Group:    entities.Group{ID: "g2", Title: "Go Август 2024"},
			PassedTopics: []entities.Topic{
				{ID: "1", Title: "Go Базовые типы"},
				{ID: "3", Title: "Go Функции"},
				{ID: "2", Title: "Go Составные типы"},
			},
		},
		{
			ID:       "stu5",
			Name:     "Dmitry",
			FullName: "Dmitry Kiselev",
			Group:    entities.Group{ID: "g2", Title: "Go Август 2024"},
			PassedTopics: []entities.Topic{
				{ID: "1", Title: "Go Базовые типы"},
			},
		},
		{
			ID:       "stu2",
			Name:     "Elena",
			FullName: "Elena Smirnova",
			Group:    entities.Group{ID: "g1", Title: "Go Май 2024"},
			PassedTopics: []entities.Topic{
				{ID: "2", Title: "Go Составные типы"},
				{ID: "1", Title: "Go Базовые типы"},
			},
		},
	}

	report, err := entities.NewPassedTopicsReport(students)
	require.NoError(t, err)

	reportRes, err := report.GetReport()
	require.NoError(t, err)
	require.NotNil(t, reportRes)

	expectedReport := []entities.GroupReport{
		{
			GroupTitle: "Go Август 2024",
			Students: []entities.StudentProgress{
				{
					Name:         "Anna Volkova",
					PassedTopics: []string{"Go Базовые типы", "Go Составные типы", "Go Функции"},
				},
				{
					Name:         "Dmitry Kiselev",
					PassedTopics: []string{"Go Базовые типы"},
				},
				{
					Name:         "Svetlana Petrova",
					PassedTopics: []string{"Go Базовые типы", "Go Составные типы", "Go Функции", "Go Cuncurrency"},
				},
			},
		},
		{
			GroupTitle: "Go Май 2024",
			Students: []entities.StudentProgress{
				{
					Name:         "Elena Smirnova",
					PassedTopics: []string{"Go Базовые типы", "Go Составные типы"},
				},
				{
					Name:         "Ivan Petrov",
					PassedTopics: []string{"Go Базовые типы", "Go Составные типы"},
				},
				{
					Name:         "Kirill Sidorov",
					PassedTopics: []string{"Go Базовые типы", "Go Составные типы"},
				},
			},
		},
	}

	expectedJSON, err := json.Marshal(expectedReport)
	require.NoError(t, err)

	require.JSONEq(t, string(expectedJSON), string(reportRes))
}

func TestNewPassedTopicsReport_Failure(t *testing.T) {
	t.Parallel()

	var badArg []entities.Student

	res, err := entities.NewPassedTopicsReport(badArg)
	require.ErrorIs(t, err, entities.ErrInvalidParam)
	require.Nil(t, res)
}
