package entities

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

const (
	PassedTopicsType = "passed_topics"
)

var (
	_ Report = (*PassedTopicsReport)(nil)
)

type Topic struct {
	ID    string
	Title string
}

type Group struct {
	ID    string
	Title string
}

type Student struct {
	ID           string
	Name         string
	FullName     string
	Group        Group
	PassedTopics []Topic
}

type PassedTopicsReport struct {
	kind     MessageType
	students []Student
}

type GroupReport struct {
	GroupTitle string
	Students   []StudentProgress
}

type StudentProgress struct {
	Name         string
	PassedTopics []string
}

func NewPassedTopicsReport(students []Student) (*PassedTopicsReport, error) {
	if len(students) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "students not set")
	}
	return &PassedTopicsReport{
		students: students,
	}, nil
}

func (report *PassedTopicsReport) GetReport() interface{} {
	report.sort()

	return report.students
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

func (report *PassedTopicsReport) Kind() MessageType {
	if report.kind == MessageType("") {
		return ReportAboutPassedTopicsByMentorGroups
	}

	return report.kind
}

func (report *PassedTopicsReport) SetMessageType() {
	report.kind = MessageType(fmt.Sprintf("%s.%s", ReportType, PassedTopicsType))
}
