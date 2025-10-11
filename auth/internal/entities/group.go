package entities

type Student struct {
	id       string
	name     string
	fullname string
}

type Group struct {
	id       string
	name     string
	students []*Student
}

func NewStudent(id, name string, fullname string) *Student {
	return &Student{
		id:       id,
		name:     name,
		fullname: fullname,
	}
}

func (s *Student) GetID() string {
	return s.id
}

func NewGroup(id, name string) *Group {
	return &Group{
		id:   id,
		name: name,
	}
}

func (g *Group) AddStudents(students []*Student) {
	g.students = students
}

func (g *Group) GetID() string {
	return g.id
}

func (g *Group) GetStudents() []*Student {
	return g.students
}
