package html

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"time"

	"github.com/parta4ok/kvs/reporting/internal/cases"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	"github.com/pkg/errors"
)

var (
	_ cases.Representer = (*HTMLRepresenter)(nil)
)

type HTMLRepresenter struct {
	title      string
	cssStyles  string
	dateFormat string
}

type HTMLReprOption func(*HTMLRepresenter)

func WithTitle(title string) HTMLReprOption {
	return func(h *HTMLRepresenter) {
		h.title = title
	}
}

func WithCSSStyles(styles string) HTMLReprOption {
	return func(h *HTMLRepresenter) {
		h.cssStyles = styles
	}
}

func WithDateFormat(format string) HTMLReprOption {
	return func(h *HTMLRepresenter) {
		h.dateFormat = format
	}
}

func (h *HTMLRepresenter) setOption(opts ...HTMLReprOption) {
	for _, opt := range opts {
		opt(h)
	}
}

func NewHTMLRepresenter(opts ...HTMLReprOption) *HTMLRepresenter {
	htmlRepresenter := &HTMLRepresenter{
		title:      "Отчет по пройденным темам",
		cssStyles:  getDefaultCSS(),
		dateFormat: "2006-01-02 15:04:05",
	}

	htmlRepresenter.setOption(opts...)

	return htmlRepresenter
}

type ReportData struct {
	Title         string      `json:"title"`
	GeneratedAt   string      `json:"generated_at"`
	Groups        []GroupData `json:"groups"`
	TotalStudents int         `json:"total_students"`
	TotalGroups   int         `json:"total_groups"`
}

type GroupData struct {
	GroupTitle string        `json:"group_title"`
	Students   []StudentData `json:"students"`
}

type StudentData struct {
	Name         string   `json:"name"`
	PassedTopics []string `json:"passed_topics"`
}

func (h *HTMLRepresenter) CovertToFormat(ctx context.Context, payload []byte) ([]byte, error) {
	var groups []GroupData
	if err := json.Unmarshal(payload, &groups); err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "unmarshal json err: %v", err)
	}

	totalStudents := 0
	for _, group := range groups {
		totalStudents += len(group.Students)
	}

	reportData := ReportData{
		Title:         h.title,
		GeneratedAt:   time.Now().Format(h.dateFormat),
		Groups:        groups,
		TotalStudents: totalStudents,
		TotalGroups:   len(groups),
	}

	htmlContent, err := h.generateHTML(reportData)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "html generation failure: %v", err)
	}

	return []byte(htmlContent), nil
}

func (h *HTMLRepresenter) generateHTML(data ReportData) (string, error) {
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"CSS": func() template.CSS {
			return template.CSS(h.cssStyles) //nolint:gosec //ok
		},
	}).Parse(getHTMLTemplate())
	if err != nil {
		return "", errors.Wrapf(entities.ErrInternal, "parsing failure: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", errors.Wrapf(entities.ErrInternal, "template using failure: %v", err)
	}

	return buf.String(), nil
}

func (representer *HTMLRepresenter) GetReportFormat() string {
	return "html"
}

//nolint:lll //ok
func getHTMLTemplate() string {
	return `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        {{CSS}}
    </style>
</head>
<body>
    <div class="container">
        <header class="header">
            <h1>{{.Title}}</h1>
            <div class="meta-info">
                <p><strong>Дата формирования отчета:</strong> {{.GeneratedAt}}</p>
                <div class="statistics">
                    <span class="stat-item">Всего групп: <strong>{{.TotalGroups}}</strong></span>
                    <span class="stat-item">Всего студентов: <strong>{{.TotalStudents}}</strong></span>
                </div>
            </div>
        </header>

        <main class="content">
            {{range .Groups}}
            <section class="group-section">
                <h2 class="group-title">{{.GroupTitle}}</h2>
                <div class="students-grid">
                    {{range .Students}}
                    <div class="student-card">
                        <h3 class="student-name">{{.Name}}</h3>
                        <div class="topics-section">
                            <h4>Пройденные темы:</h4>
                            <ul class="topics-list">
                                {{range .PassedTopics}}
                                <li class="topic-item">{{.}}</li>
                                {{end}}
                            </ul>
                        </div>
                    </div>
                    {{end}}
                </div>
            </section>
            {{end}}
        </main>

        <footer class="footer">
            <p>Отчет сгенерирован автоматически системой отчетности</p>
        </footer>
    </div>
</body>
</html>`
}

//nolint:funlen,lll //ok
func getDefaultCSS() string {
	return `
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            line-height: 1.6;
            color: #333;
            background-color: #f5f5f5;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #fff;
            box-shadow: 0 0 10px rgba(0,0,0,0.1);
            min-height: 100vh;
        }

        .header {
            text-align: center;
            margin-bottom: 40px;
            padding-bottom: 20px;
            border-bottom: 2px solid #e0e0e0;
        }

        .header h1 {
            color: #2c3e50;
            margin-bottom: 20px;
            font-size: 2.5em;
            font-weight: 300;
        }

        .meta-info {
            color: #666;
            font-size: 1.1em;
        }

        .statistics {
            margin-top: 15px;
            display: flex;
            justify-content: center;
            gap: 30px;
            flex-wrap: wrap;
        }

        .stat-item {
            background-color: #ecf0f1;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 0.95em;
        }

        .content {
            margin-bottom: 40px;
        }

        .group-section {
            margin-bottom: 50px;
        }

        .group-title {
            background: linear-gradient(135deg, #3498db, #2980b9);
            color: white;
            padding: 15px 25px;
            border-radius: 8px;
            margin-bottom: 25px;
            font-size: 1.4em;
            font-weight: 500;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }

        .students-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
            gap: 20px;
        }

        .student-card {
            background-color: #fff;
            border: 1px solid #ddd;
            border-radius: 10px;
            padding: 20px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.08);
            transition: transform 0.2s ease, box-shadow 0.2s ease;
        }

        .student-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 15px rgba(0,0,0,0.12);
        }

        .student-name {
            color: #2c3e50;
            margin-bottom: 15px;
            font-size: 1.2em;
            font-weight: 600;
            border-bottom: 2px solid #e8f4f8;
            padding-bottom: 8px;
        }

        .topics-section h4 {
            color: #555;
            margin-bottom: 10px;
            font-size: 1em;
            font-weight: 500;
        }

        .topics-list {
            list-style: none;
            padding: 0;
        }

        .topic-item {
            background-color: #f8f9fa;
            margin-bottom: 6px;
            padding: 8px 12px;
            border-radius: 5px;
            border-left: 4px solid #3498db;
            font-size: 0.95em;
            transition: background-color 0.2s ease;
        }

        .topic-item:hover {
            background-color: #e3f2fd;
        }

        .footer {
            text-align: center;
            padding: 20px;
            border-top: 1px solid #e0e0e0;
            color: #888;
            font-size: 0.9em;
        }

        @media (max-width: 768px) {
            .container {
                padding: 15px;
            }

            .header h1 {
                font-size: 2em;
            }

            .statistics {
                flex-direction: column;
                align-items: center;
                gap: 10px;
            }

            .students-grid {
                grid-template-columns: 1fr;
            }

            .group-title {
                font-size: 1.2em;
                padding: 12px 20px;
            }
        }

        @media print {
            body {
                background-color: #fff;
            }

            .container {
                box-shadow: none;
                padding: 0;
            }

            .student-card {
                break-inside: avoid;
                box-shadow: none;
                border: 1px solid #ccc;
            }

            .group-section {
                break-inside: avoid;
            }
        }
    `
}
