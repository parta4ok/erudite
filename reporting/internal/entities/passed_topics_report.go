package entities

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

type PassedTopicsReport struct {
	students []Student
}

type GroupReport struct {
	GroupTitle string            `json:"group_title"`
	Students   []StudentProgress `json:"students"`
}

type StudentProgress struct {
	Name         string   `json:"name"`
	PassedTopics []string `json:"passed_topics"`
}

func NewPassedTopicsReport(students []Student) (*PassedTopicsReport, error) {
	if len(students) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "students not set")
	}
	return &PassedTopicsReport{
		students: students,
	}, nil
}

func (report *PassedTopicsReport) GetReport() ([]byte, error) {
	report.sort()

	groupsMap := make(map[string][]Student)
	var groupOrder []string

	for _, student := range report.students {
		groupTitle := student.Group.Title
		if _, exists := groupsMap[groupTitle]; !exists {
			groupOrder = append(groupOrder, groupTitle)
			groupsMap[groupTitle] = make([]Student, 0)
		}
		groupsMap[groupTitle] = append(groupsMap[groupTitle], student)
	}

	var reportData []GroupReport
	for _, groupTitle := range groupOrder {
		students := groupsMap[groupTitle]

		var studentProgress []StudentProgress
		for _, student := range students {
			var topicTitles []string
			for _, topic := range student.PassedTopics {
				topicTitles = append(topicTitles, topic.Title)
			}

			studentProgress = append(studentProgress, StudentProgress{
				Name:         student.FullName,
				PassedTopics: topicTitles,
			})
		}

		reportData = append(reportData, GroupReport{
			GroupTitle: groupTitle,
			Students:   studentProgress,
		})
	}

	data, err := json.Marshal(reportData)
	if err != nil {
		return nil, errors.Wrapf(ErrInternal, "marshalling failed: %v", err)
	}

	return data, nil
}

func (report *PassedTopicsReport) sort() {
	slices.SortFunc(report.students, func(a, b Student) int {
		if a.Group.Title != b.Group.Title {
			return strings.Compare(a.Group.Title, b.Group.Title)
		}
		return strings.Compare(a.FullName, b.FullName)
	})

	for i := range report.students {
		slices.SortFunc(report.students[i].PassedTopics, func(a, b Topic) int {
			return strings.Compare(a.ID, b.ID)
		})
	}
}
