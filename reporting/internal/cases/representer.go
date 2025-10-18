package cases

import "context"

//go:generate mockgen -source=./representer.go -destination=./testdata/representer.go -package=testdata
type Representer interface {
	CovertToFormat(ctx context.Context, payload []byte) ([]byte, error)
	GetReportFormat() string
}
