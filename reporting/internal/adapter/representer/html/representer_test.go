package html_test

import (
	"context"
	"testing"

	"github.com/parta4ok/kvs/reporting/internal/adapter/representer/html"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	"github.com/stretchr/testify/require"
)

func TestHTMLRepresenter_Success(t *testing.T) {
	t.Parallel()

	representer := html.NewHTMLRepresenter()
	require.NotNil(t, representer)

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
	require.NotNil(t, report)
	reportData, err := report.GetReport()
	require.NoError(t, err)
	require.NotNil(t, reportData)

	data, err := representer.CovertToFormat(context.Background(), reportData)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	htmlContent := string(data)
	require.Contains(t, htmlContent, "<!DOCTYPE html>")
	require.Contains(t, htmlContent, "<html lang=\"ru\">")
	require.Contains(t, htmlContent, "</html>")
}

func TestHTMLRepresenter_CyrillicSupport(t *testing.T) {
	t.Parallel()

	students := []entities.Student{
		{
			ID:       "stu1",
			Name:     "Иван",
			FullName: "Иван Петров",
			Group:    entities.Group{ID: "g1", Title: "Go Май 2024"},
			PassedTopics: []entities.Topic{
				{ID: "1", Title: "Go Базовые типы"},
				{ID: "2", Title: "Go Составные типы"},
			},
		},
		{
			ID:       "stu2",
			Name:     "Светлана",
			FullName: "Светлана Петрова",
			Group:    entities.Group{ID: "g2", Title: "Go Август 2024"},
			PassedTopics: []entities.Topic{
				{ID: "1", Title: "Go Базовые типы"},
				{ID: "3", Title: "Go Функции"},
				{ID: "4", Title: "Go Конкурентность"},
			},
		},
		{
			ID:       "stu3",
			Name:     "Кирилл",
			FullName: "Кирилл Сидоров",
			Group:    entities.Group{ID: "g1", Title: "Go Май 2024"},
			PassedTopics: []entities.Topic{
				{ID: "2", Title: "Go Составные типы"},
				{ID: "1", Title: "Go Базовые типы"},
			},
		},
	}

	representer := html.NewHTMLRepresenter()
	require.NotNil(t, representer)

	report, err := entities.NewPassedTopicsReport(students)
	require.NoError(t, err)
	require.NotNil(t, report)

	reportData, err := report.GetReport()
	require.NoError(t, err)
	require.NotNil(t, reportData)

	jsonStr := string(reportData)
	require.Contains(t, jsonStr, "Иван")
	require.Contains(t, jsonStr, "Светлана")
	require.Contains(t, jsonStr, "Кирилл")
	require.Contains(t, jsonStr, "Базовые типы")
	require.Contains(t, jsonStr, "Конкурентность")

	htmlData, err := representer.CovertToFormat(context.Background(), reportData)
	require.NoError(t, err)
	require.NotEmpty(t, htmlData)

	htmlContent := string(htmlData)
	require.Contains(t, htmlContent, "Иван Петров")
	require.Contains(t, htmlContent, "Светлана Петрова")
	require.Contains(t, htmlContent, "Кирилл Сидоров")
	require.Contains(t, htmlContent, "Go Базовые типы")
	require.Contains(t, htmlContent, "Go Конкурентность")

	require.Contains(t, htmlContent, "<!DOCTYPE html>")
	require.Contains(t, htmlContent, "<meta charset=\"UTF-8\">")
	require.Contains(t, htmlContent, "Отчет по пройденным темам")
}

func TestHTMLRepresenter_WithCustomOptions(t *testing.T) {
	t.Parallel()

	representer := html.NewHTMLRepresenter(
		html.WithTitle("Кастомный отчет по студентам"),
		html.WithDateFormat("02.01.2006 15:04"),
	)
	require.NotNil(t, representer)

	students := []entities.Student{
		{
			ID:       "stu1",
			Name:     "Test",
			FullName: "Test User",
			Group:    entities.Group{ID: "g1", Title: "Test Group"},
			PassedTopics: []entities.Topic{
				{ID: "1", Title: "Test Topic"},
			},
		},
	}

	report, err := entities.NewPassedTopicsReport(students)
	require.NoError(t, err)

	reportData, err := report.GetReport()
	require.NoError(t, err)

	htmlData, err := representer.CovertToFormat(context.Background(), reportData)
	require.NoError(t, err)
	require.NotEmpty(t, htmlData)

	htmlContent := string(htmlData)

	require.Contains(t, htmlContent, "Кастомный отчет по студентам")

	require.Contains(t, htmlContent, "Дата формирования отчета:")

	require.Contains(t, htmlContent, "Всего групп: <strong>1</strong>")
	require.Contains(t, htmlContent, "Всего студентов: <strong>1</strong>")
}

func TestHTMLRepresenter_GetReportFormat(t *testing.T) {
	t.Parallel()

	representer := html.NewHTMLRepresenter()
	require.Equal(t, "html", representer.GetReportFormat())
}

func TestHTMLRepresenter_EmptyData(t *testing.T) {
	t.Parallel()

	representer := html.NewHTMLRepresenter()
	require.NotNil(t, representer)

	emptyData := []byte("[]")

	htmlData, err := representer.CovertToFormat(context.Background(), emptyData)
	require.NoError(t, err)
	require.NotEmpty(t, htmlData)

	htmlContent := string(htmlData)

	require.Contains(t, htmlContent, "<!DOCTYPE html>")
	require.Contains(t, htmlContent, "Всего групп: <strong>0</strong>")
	require.Contains(t, htmlContent, "Всего студентов: <strong>0</strong>")
}

func TestHTMLRepresenter_InvalidJSON(t *testing.T) {
	t.Parallel()

	representer := html.NewHTMLRepresenter()
	require.NotNil(t, representer)

	invalidData := []byte("invalid json data")

	_, err := representer.CovertToFormat(context.Background(), invalidData)
	require.ErrorIs(t, err, entities.ErrInternal)
}

func TestHTMLRepresenter_HTMLStructure(t *testing.T) {
	t.Parallel()

	representer := html.NewHTMLRepresenter()

	testData := []byte(`[{"group_title":"Test Group","students":[{"name":"Test Student","passed_topics":["Topic 1","Topic 2"]}]}]`)

	htmlData, err := representer.CovertToFormat(context.Background(), testData)
	require.NoError(t, err)

	htmlContent := string(htmlData)

	expectedElements := []string{
		"<!DOCTYPE html>",
		"<html lang=\"ru\">",
		"<meta charset=\"UTF-8\">",
		"<title>",
		"<style>",
		"<body>",
		"<div class=\"container\">",
		"<header class=\"header\">",
		"<main class=\"content\">",
		"<footer class=\"footer\">",
		"</html>",
	}

	for _, element := range expectedElements {
		require.Contains(t, htmlContent, element)
	}

	expectedClasses := []string{
		"class=\"container\"",
		"class=\"header\"",
		"class=\"content\"",
		"class=\"group-section\"",
		"class=\"student-card\"",
		"class=\"topics-list\"",
	}

	for _, class := range expectedClasses {
		require.Contains(t, htmlContent, class)
	}
}
