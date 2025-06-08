package generator

import (
	"time"

	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
)

var (
	_ entities.IDGenerator = (*Uint64Generator)(nil)
)

type Uint64Generator struct{}

func NewUint64Generator() *Uint64Generator {
	return &Uint64Generator{}
}

func (gen *Uint64Generator) GenerateID() uint64 {
	return uint64(time.Now().UTC().UnixNano())
}
