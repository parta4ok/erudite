package representer

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/parta4ok/kvs/reporting/internal/entities"
	"github.com/pkg/errors"
)

var (
	_ RepresentStrategy = (*SessionResultToHTMLStrategy)(nil)
)

type SessionResultToHTMLStrategy struct{}

func (strategy *SessionResultToHTMLStrategy) Apply(format entities.Format,
	reportType entities.MessageType) bool {

	return string(format) == "html" &&
		strings.Contains(reportType.String(), entities.ReportType) &&
		strings.Contains(reportType.String(), entities.SessionResultType)
}

func (strategy *SessionResultToHTMLStrategy) Proccess(report entities.Report) ([]byte, error) {
	sessionResultReport, ok := report.(*entities.SessionResult)
	if !ok {
		return nil, errors.Wrap(entities.ErrInvalidParam, "expected *entities.SessionResult")
	}

	sessionData, err := strategy.prepareSessionData(sessionResultReport)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare session data")
	}

	htmlContent, err := strategy.generateHTML(sessionData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate HTML")
	}

	return htmlContent, nil
}

type SessionData struct {
	StudentName string
	GroupTitle  string
	IsSuccess   bool
	IsExpired   bool
	Resume      string
	Topics      []string
	Questions   []QuestionData
	Status      string
	StatusColor string
}

type QuestionData struct {
	ID          string
	Subject     string
	UserAnswers []string
}

//nolint:funlen //ok
func (strategy *SessionResultToHTMLStrategy) prepareSessionData(report *entities.SessionResult,
) (*SessionData, error) {
	sessionResult := report

	var questions []QuestionData

	for questionText := range sessionResult.Questions {
		userAnswers := sessionResult.UserAnswer[questionText]

		questions = append(questions, QuestionData{
			ID:          fmt.Sprintf("q_%d", len(questions)+1),
			Subject:     questionText,
			UserAnswers: userAnswers,
		})
	}

	var status, statusColor string
	switch {
	case sessionResult.IsSuccess:
		status = "Успех"
		statusColor = "#28a745" // green
	case sessionResult.IsExpire:
		status = "Время истекло"
		statusColor = "#ffc107" // yellow
	default:
		status = "Неудача"
		statusColor = "#dc3545" // red
	}

	return &SessionData{
		StudentName: fmt.Sprintf("%s (%s)", sessionResult.UserName, sessionResult.UserID),
		GroupTitle:  sessionResult.GroupTitle,
		IsSuccess:   sessionResult.IsSuccess,
		IsExpired:   sessionResult.IsExpire,
		Resume:      sessionResult.Resume,
		Topics:      sessionResult.Topics,
		Questions:   questions,
		Status:      status,
		StatusColor: statusColor,
	}, nil
}

func (strategy *SessionResultToHTMLStrategy) generateHTML(sessionData *SessionData,
) ([]byte, error) {
	tmpl := template.Must(template.New("sessionResultReport").Parse(sessionHtmlTemplate))

	data := struct {
		Title   string
		Session *SessionData
	}{
		Title:   fmt.Sprintf("Отчет по сессии - %s", sessionData.StudentName),
		Session: sessionData,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, errors.Wrap(err, "template execution failed")
	}

	return buf.Bytes(), nil
}

//nolint:lll // HTML template contains long lines for readability
const sessionHtmlTemplate = `
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
            line-height: 1.6;
        }
        .container {
            max-width: 1000px;
            margin: 0 auto;
            background-color: white;
            border-radius: 12px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
            color: white;
            padding: 40px;
            text-align: center;
        }
        .header h1 {
            margin: 0 0 10px 0;
            font-size: 2.5em;
            font-weight: 300;
        }
        .header .subtitle {
            font-size: 1.2em;
            opacity: 0.9;
        }
        .summary {
            padding: 30px;
            border-bottom: 1px solid #e5e7eb;
        }
        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        .summary-card {
            background: #f8fafc;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #e5e7eb;
            text-align: center;
        }
        .summary-card.success {
            border-left-color: #10b981;
            background: linear-gradient(135deg, #ecfdf5 0%, #f0fdf4 100%);
        }
        .summary-card.warning {
            border-left-color: #f59e0b;
            background: linear-gradient(135deg, #fffbeb 0%, #fefce8 100%);
        }
        .summary-card.danger {
            border-left-color: #ef4444;
            background: linear-gradient(135deg, #fef2f2 0%, #fef2f2 100%);
        }
        .summary-value {
            font-size: 2em;
            font-weight: bold;
            margin-bottom: 5px;
        }
        .summary-label {
            color: #6b7280;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .status-badge {
            display: inline-block;
            padding: 8px 16px;
            border-radius: 20px;
            color: white;
            font-weight: 600;
            font-size: 1.1em;
            margin: 10px 0;
        }
        .info-section {
            padding: 30px;
            border-bottom: 1px solid #e5e7eb;
        }
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
        }
        .info-item {
            display: flex;
            align-items: center;
        }
        .info-icon {
            font-size: 1.5em;
            margin-right: 12px;
            width: 30px;
            text-align: center;
        }
        .info-content h3 {
            margin: 0 0 5px 0;
            color: #374151;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .info-content p {
            margin: 0;
            font-size: 1.1em;
            font-weight: 500;
        }
        .topics-list {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
            margin-top: 5px;
        }
        .topic-tag {
            background: #e0e7ff;
            color: #4338ca;
            padding: 4px 12px;
            border-radius: 16px;
            font-size: 0.85em;
            font-weight: 500;
        }
        .questions-section {
            padding: 30px;
        }
        .section-title {
            font-size: 1.8em;
            font-weight: 600;
            color: #1f2937;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
        }
        .section-title::before {
            content: "❓";
            margin-right: 12px;
            font-size: 1.2em;
        }
        .questions-grid {
            display: grid;
            gap: 20px;
        }
        .question-card {
            background: #f9fafb;
            border: 1px solid #e5e7eb;
            border-radius: 8px;
            padding: 20px;
            transition: transform 0.2s ease, box-shadow 0.2s ease;
        }
        .question-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 25px rgba(0,0,0,0.1);
        }
        .question-header {
            display: flex;
            justify-content: between;
            align-items: flex-start;
            margin-bottom: 15px;
        }
        .question-info {
            flex: 1;
        }
        .question-subject {
            font-size: 1.2em;
            font-weight: 600;
            color: #1f2937;
            margin: 0 0 5px 0;
        }
        .question-topic {
            color: #6b7280;
            font-size: 0.9em;
            margin: 0;
        }
        .answer-status {
            padding: 6px 12px;
            border-radius: 16px;
            font-size: 0.85em;
            font-weight: 600;
            color: white;
        }
        .user-answers {
            margin-top: 15px;
        }
        .answers-label {
            font-weight: 600;
            color: #374151;
            margin-bottom: 8px;
            font-size: 0.95em;
        }
        .answers-list {
            background: white;
            border: 1px solid #d1d5db;
            border-radius: 6px;
            padding: 12px;
        }
        .answer-item {
            padding: 6px 0;
            border-bottom: 1px solid #f3f4f6;
        }
        .answer-item:last-child {
            border-bottom: none;
        }
        .resume-section {
            background: #f8fafc;
            padding: 30px;
            border-top: 1px solid #e5e7eb;
        }
        .resume-content {
            background: white;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #6366f1;
        }
        .resume-title {
            font-size: 1.2em;
            font-weight: 600;
            color: #1f2937;
            margin: 0 0 10px 0;
            display: flex;
            align-items: center;
        }
        .resume-title::before {
            content: "📝";
            margin-right: 8px;
        }
        .footer {
            background-color: #f8f9fa;
            padding: 20px;
            text-align: center;
            color: #6c757d;
            font-size: 0.9em;
        }
        @media (max-width: 768px) {
            .summary-grid,
            .info-grid {
                grid-template-columns: 1fr;
            }
            .header h1 {
                font-size: 2em;
            }
            .header .subtitle {
                font-size: 1em;
            }
            .container {
                margin: 10px;
                border-radius: 8px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
            <div class="subtitle">Детальный отчет по результатам сессии</div>
        </div>
        
        <div class="summary">
            <div class="summary-grid">
                <div class="summary-card {{if .Session.IsSuccess}}success{{else if .Session.IsExpired}}warning{{else}}danger{{end}}">
                    <div class="summary-value" style="color: {{.Session.StatusColor}}">{{.Session.Resume}}</div>
                    <div class="summary-label">Результат</div>
                </div>
                <div class="summary-card">
                    <div class="summary-value">{{len .Session.Questions}}</div>
                    <div class="summary-label">Всего вопросов</div>
                </div>
                <div class="summary-card">
                    <div class="summary-value">{{len .Session.Topics}}</div>
                    <div class="summary-label">Тем изучено</div>
                </div>
            </div>
            
            <div style="text-align: center;">
                <span class="status-badge" style="background-color: {{.Session.StatusColor}}">
                    {{.Session.Status}}
                </span>
            </div>
        </div>

        <div class="info-section">
            <div class="info-grid">
                <div class="info-item">
                    <div class="info-icon">👤</div>
                    <div class="info-content">
                        <h3>Студент</h3>
                        <p>{{.Session.StudentName}}</p>
                    </div>
                </div>
                <div class="info-item">
                    <div class="info-icon">👥</div>
                    <div class="info-content">
                        <h3>Группа</h3>
                        <p>{{.Session.GroupTitle}}</p>
                    </div>
                </div>
                <div class="info-item">
                    <div class="info-icon">📚</div>
                    <div class="info-content">
                        <h3>Изученные темы</h3>
                        <div class="topics-list">
                            {{range .Session.Topics}}
                            <span class="topic-tag">{{.}}</span>
                            {{end}}
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="questions-section">
            <h2 class="section-title">Детали по вопросам</h2>
            <div class="questions-grid">
                {{range .Session.Questions}}
                <div class="question-card">
                    <div class="question-header">
                        <div class="question-info">
                            <h3 class="question-subject">{{.Subject}}</h3>
                        </div>
                    </div>
                    
                    <div class="user-answers">
                        <div class="answers-label">Ваши ответы:</div>
                        <div class="answers-list">
                            {{if and .UserAnswers (ne (index .UserAnswers 0) "")}}
                                {{range .UserAnswers}}
                                <div class="answer-item">{{.}}</div>
                                {{end}}
                            {{else}}
                                <div class="answer-item" style="color: #6b7280; font-style: italic;">Ответ не дан</div>
                            {{end}}
                        </div>
                    </div>
                </div>
                {{end}}
            </div>
        </div>

        {{if .Session.Resume}}
        <div class="resume-section">
            <div class="resume-content">
                <h3 class="resume-title">Резюме сессии</h3>
                <p>{{.Session.Resume}}</p>
            </div>
        </div>
        {{end}}
        
        <div class="footer">
            Отчет сгенерирован автоматически системой обучения
        </div>
    </div>
</body>
</html>
` //nolint:lll //ok
