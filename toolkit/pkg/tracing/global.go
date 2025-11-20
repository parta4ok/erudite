package tracing

import (
	"context"
	"fmt"
	"sync"
)

const (
	portType = "tracer jaeger"
	noOpType = "noOp tracer"
)

var (
	globalTracer Tracer
	once         sync.Once
	mu           sync.RWMutex
)

func InitGlobalTracer(tracer Tracer) {
	once.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		globalTracer = tracer
		fmt.Printf("globalTracer initialized with tracer type: %T\n", tracer)
	})
}

func GlobalTracer() Tracer {
	mu.RLock()
	defer mu.RUnlock()

	if globalTracer == nil {
		fmt.Printf("globalTracer is nil, returning NoOpTracer\n")
		return &NoOpTracer{}
	}

	fmt.Printf("globalTracer returning tracer type: %T\n", globalTracer)
	return globalTracer
}

func CloseGlobalTracer() error {
	mu.RLock()
	tracer := globalTracer
	mu.RUnlock()

	if tracer != nil {
		return tracer.Close()
	}

	return nil
}

type TracerPort struct {
	tracer Tracer
}

func NewTracePort(tracer Tracer) *TracerPort {
	return &TracerPort{
		tracer: tracer,
	}
}

func (t *TracerPort) Start(_ context.Context) error {
	InitGlobalTracer(t.tracer)
	return nil
}

func (t *TracerPort) Stop(_ context.Context) error {
	return CloseGlobalTracer()
}

func (t *TracerPort) Type() string {
	return portType
}
