package config

import "time"

type BaseConfig interface {
	ServiceName() string
	ServiceVersion() string
	LogLevel() string
	LogAddSource() bool
	LogFormat() string
	TracingEnabled() bool
	TracingEndpoint() string
	ShutdownTimeout() time.Duration
}
