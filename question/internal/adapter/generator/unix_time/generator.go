package unixtime

import (
	"fmt"
	"time"

	"github.com/parta4ok/kvs/question/internal/entities"
)

var (
	_ entities.IDGenerator = (*Uint64Generator)(nil)
)

type Uint64Generator struct{}

func NewUint64Generator() *Uint64Generator {
	return &Uint64Generator{}
}

func (gen *Uint64Generator) GenerateID() string {
	return fmt.Sprintf("%d", uint64(time.Now().UTC().UnixNano()))
}
