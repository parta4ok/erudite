package cryptoprocessing

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

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
	var buf [8]byte
	rand.Read(buf[:]) //nolint:errcheck,gosec // ok

	return fmt.Sprintf("%d", binary.BigEndian.Uint64(buf[:]))
}
