package entities

type Student struct {
	ID           string
	Name         string
	FullName     string
	Group        Group
	PassedTopics []Topic
}
