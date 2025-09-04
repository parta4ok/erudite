package tracing

import (
	"sync"
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
	})
}

func GlobalTracer() Tracer {
	mu.RLock()
	defer mu.RUnlock()

	if globalTracer == nil {

		return &NoOpTracer{}
	}

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
