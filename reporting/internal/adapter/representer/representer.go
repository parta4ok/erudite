package representer

import (
	"github.com/parta4ok/kvs/reporting/internal/entities"
	"github.com/pkg/errors"
)

var (
	_ entities.Representer = (*Representer)(nil)
)

type Representer struct {
	strategies []RepresentStrategy
}

func NewRepresenter() (*Representer, error) {
	return &Representer{
		strategies: []RepresentStrategy{
			&PassedTopicsToHTMLStrategy{},
			&SessionResultToHTMLStrategy{},
		},
	}, nil
}

func (r *Representer) CovertToFormat(format entities.Format, report entities.Report,
	) ([]byte, error) {
	var strategy RepresentStrategy

	for _, s := range r.strategies {
		if s.Apply(format, report.Kind()) {
			strategy = s
			break
		}
	}

	if strategy == nil || strategy == RepresentStrategy(nil) {
		return nil, errors.Wrap(entities.ErrInvalidParam, "strategy not set")
	}

	return strategy.Proccess(report)
}
