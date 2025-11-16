package entities

//go:generate mockgen -source=./representer.go -destination=./testdata/representer.go -package=testdata
type Representer interface {
	CovertToFormat(format Format, report Report) ([]byte, error)
}
