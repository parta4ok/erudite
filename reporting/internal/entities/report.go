package entities

const (
	ReportType = "report"
)

//go:generate mockgen -source=./report.go -destination=./testdata/report.go -package=testdata
type Report interface {
	Kind() MessageType
	GetReport() interface{}
	SetMessageType()
}
