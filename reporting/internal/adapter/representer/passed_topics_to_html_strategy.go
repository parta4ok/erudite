package representer

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/parta4ok/kvs/reporting/internal/entities"
	"github.com/pkg/errors"
)

var (
	_ RepresentStrategy = (*PassedTopicsToHTMLStrategy)(nil)
)

type PassedTopicsToHTMLStrategy struct{}

func (strategy *PassedTopicsToHTMLStrategy) Apply(format entities.Format,
	reportType entities.MessageType) bool {
	return format == entities.Format("html") &&
		strings.Contains(reportType.String(), entities.ReportType) &&
		strings.Contains(reportType.String(), entities.PassedTopicsType)
}

func (strategy *PassedTopicsToHTMLStrategy) Proccess(report entities.Report) ([]byte, error) {
	passedTopicsReport, ok := report.(*entities.PassedTopicsReport)
	if !ok {
		return nil, errors.Wrap(entities.ErrInvalidParam, "expected *entities.PassedTopicsReport")
	}

	groupedData, err := strategy.prepareGroupedData(passedTopicsReport)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare grouped data")
	}

	htmlContent, err := strategy.generateHTML(groupedData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate HTML")
	}

	return htmlContent, nil
}

type GroupData struct {
	GroupTitle string
	Students   []StudentData
}

type StudentData struct {
	Name         string
	PassedTopics []string
	TopicsCount  int
}

func (strategy *PassedTopicsToHTMLStrategy) prepareGroupedData(report *entities.PassedTopicsReport,
) ([]GroupData, error) {
	reportData := report.GetReport()
	students, ok := reportData.([]entities.Student)
	if !ok {
		return nil, errors.Wrap(entities.ErrInternal, "invalid report data format")
	}

	groupsMap := make(map[string][]StudentData)
	var groupOrder []string

	for _, student := range students {
		groupTitle := student.Group.Title
		if _, exists := groupsMap[groupTitle]; !exists {
			groupOrder = append(groupOrder, groupTitle)
			groupsMap[groupTitle] = make([]StudentData, 0)
		}

		var topicTitles []string
		for _, topic := range student.PassedTopics {
			topicTitles = append(topicTitles, topic.Title)
		}

		studentData := StudentData{
			Name:         student.FullName,
			PassedTopics: topicTitles,
			TopicsCount:  len(topicTitles),
		}

		groupsMap[groupTitle] = append(groupsMap[groupTitle], studentData)
	}

	var result []GroupData
	for _, groupTitle := range groupOrder {
		result = append(result, GroupData{
			GroupTitle: groupTitle,
			Students:   groupsMap[groupTitle],
		})
	}

	return result, nil
}

func (strategy *PassedTopicsToHTMLStrategy) generateHTML(groupedData []GroupData) ([]byte, error) {
	tmpl := template.Must(template.New("passedTopicsReport").Parse(htmlTemplate))

	data := struct {
		Title  string
		Groups []GroupData
	}{
		Title:  "Отчет по пройденным темам",
		Groups: groupedData,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, errors.Wrap(err, "template execution failed")
	}

	return buf.Bytes(), nil
}

//nolint:lll // HTML template contains long lines for readability
const htmlTemplate = `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
            color: #333;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 2.2em;
            font-weight: 300;
        }
        .content {
            padding: 30px;
        }
        .group {
            margin-bottom: 40px;
            border: 1px solid #e1e8ed;
            border-radius: 8px;
            overflow: hidden;
        }
        .group-header {
            background-color: #f8f9fa;
            padding: 20px;
            border-bottom: 2px solid #e1e8ed;
        }
        .group-title {
            font-size: 1.5em;
            font-weight: 600;
            color: #495057;
            margin: 0;
            display: flex;
            align-items: center;
        }
        .group-title::before {
            content: "👥";
            margin-right: 10px;
            font-size: 1.2em;
        }
        .students-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
            gap: 20px;
            padding: 20px;
        }
        .student-card {
            background-color: #fff;
            border: 1px solid #e1e8ed;
            border-radius: 8px;
            padding: 20px;
            transition: transform 0.2s ease, box-shadow 0.2s ease;
        }
        .student-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 20px rgba(0,0,0,0.1);
        }
        .student-name {
            font-size: 1.2em;
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 15px;
            display: flex;
            align-items: center;
        }
        .student-name::before {
            content: "👤";
            margin-right: 8px;
        }
        .topics-count {
            background-color: #e3f2fd;
            color: #1976d2;
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 0.9em;
            font-weight: 500;
            margin-left: auto;
        }
        .topics-section {
            margin-top: 15px;
        }
        .topics-label {
            font-weight: 600;
            color: #495057;
            margin-bottom: 10px;
            display: flex;
            align-items: center;
            font-size: 0.95em;
        }
        .topics-label::before {
            content: "📚";
            margin-right: 8px;
        }
        .topics-list {
            list-style: none;
            padding: 0;
            margin: 0;
        }
        .topic-item {
            background-color: #f8f9fa;
            margin: 6px 0;
            padding: 10px 15px;
            border-radius: 6px;
            border-left: 4px solid #28a745;
            font-size: 0.95em;
            transition: background-color 0.2s ease;
        }
        .topic-item:hover {
            background-color: #e9ecef;
        }
        .no-topics {
            color: #6c757d;
            font-style: italic;
            text-align: center;
            padding: 20px;
            background-color: #f8f9fa;
            border-radius: 6px;
        }
        .footer {
            background-color: #f8f9fa;
            padding: 20px;
            text-align: center;
            color: #6c757d;
            font-size: 0.9em;
        }
        @media (max-width: 768px) {
            .students-grid {
                grid-template-columns: 1fr;
            }
            .header h1 {
                font-size: 1.8em;
            }
            .content {
                padding: 20px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
        </div>
        
        <div class="content">
            {{range .Groups}}
            <div class="group">
                <div class="group-header">
                    <h2 class="group-title">{{.GroupTitle}}</h2>
                </div>
                
                <div class="students-grid">
                    {{range .Students}}
                    <div class="student-card">
                        <div class="student-name">
                            {{.Name}}
                            <span class="topics-count">{{.TopicsCount}} тем</span>
                        </div>
                        
                        <div class="topics-section">
                            <div class="topics-label">Пройденные темы:</div>
                            {{if .PassedTopics}}
                            <ul class="topics-list">
                                {{range .PassedTopics}}
                                <li class="topic-item">{{.}}</li>
                                {{end}}
                            </ul>
                            {{else}}
                            <div class="no-topics">Темы не пройдены</div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>
        
        <div class="footer">
            Отчет сгенерирован автоматически системой обучения
        </div>
    </div>
</body>
</html>
`
