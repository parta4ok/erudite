package representer

import "github.com/parta4ok/kvs/reporting/internal/entities"

type RepresentStrategy interface {
	Apply(format entities.Format, reportType entities.MessageType) bool
	Proccess(entities.Report) ([]byte, error)
}
