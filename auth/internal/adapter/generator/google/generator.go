package google

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/parta4ok/kvs/auth/internal/cases/common"
)

var (
	_ common.IDGenerator = (*Generator)(nil)
)

type Generator struct {
}

func NewGenerator() (*Generator, error) {
	g := &Generator{}

	return g, nil
}

func (g *Generator) Generate(_ context.Context) (string, error) {
	slog.Info("Generate started")

	return uuid.New().String(), nil
}
