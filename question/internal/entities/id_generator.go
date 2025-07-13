package entities

//go:generate mockgen -source=id_generator.go -destination=./testdata/id_generator.go -package=testdata
type IDGenerator interface {
	GenerateID() string
}
