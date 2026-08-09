package config

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var (
	ErrConfig = errors.New("config error")
)

type Config struct {
	viper *viper.Viper
}

func NewConfig(path string) (*Config, error) {
	if path == "" {
		return nil, errors.Wrap(ErrConfig, "invalid path")
	}

	config := &Config{
		viper: viper.New(),
	}

	config.viper.SetConfigFile(path)
	config.viper.SetConfigType("yaml")

	if err := config.viper.ReadInConfig(); err != nil {
		return nil, errors.Wrapf(ErrConfig, "read config failure: %v", err)
	}

	return config, nil
}

func (cfg *Config) GetPublicPort() string {
	return cfg.viper.GetString("kvs.http.public.port")
}

func (cfg *Config) GetPrivatePort() string {
	return cfg.viper.GetString("kvs.http.private.port")
}

func (cfg *Config) GetLogLevel() string {
	return cfg.viper.GetString("kvs.logging.level")
}

func (cfg *Config) GetLogFormat() string {
	return cfg.viper.GetString("kvs.logging.format")
}

func (cfg *Config) GetLogAddSource() bool {
	return cfg.viper.GetBool("kvs.logging.add_source")
}

func (cfg *Config) GetServiceName() string {
	return cfg.viper.GetString("kvs.logging.service_name")
}

func (cfg *Config) GetServiceVersion() string {
	return cfg.viper.GetString("kvs.logging.service_version")
}

func (cfg *Config) GetServiceStorageType() string {
	return cfg.viper.GetString("kvs.storage.type")
}

func (cfg *Config) GetStorageConnStr(storageType string) string {
	return cfg.viper.GetString(fmt.Sprintf("%s.connection", storageType))
}

func (cfg *Config) GetPublicTimeout() time.Duration {
	timeoutStr := cfg.viper.GetString("kvs.http.public.timeout")
	if timeoutStr == "" {
		return 30 * time.Second
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 30 * time.Second
	}

	return timeout
}

func (cfg *Config) GetPrivateTimeout() time.Duration {
	timeoutStr := cfg.viper.GetString("kvs.http.private.timeout")
	if timeoutStr == "" {
		return 30 * time.Second
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 30 * time.Second
	}

	return timeout
}

func (cfg *Config) GetAuthConn() string {
	return cfg.viper.GetString("auth_service.address")
}

func (cfg *Config) GetEventTimeout() time.Duration {
	return cfg.viper.GetDuration("nats.event_timeout")
}

func (cfg *Config) GetNatsURL() string {
	return cfg.viper.GetString("nats.url")
}

func (cfg *Config) GetNatsSubject() string {
	return cfg.viper.GetString("nats.subject")
}

func (cfg *Config) GetTimeToRespond() time.Duration {
	return cfg.viper.GetDuration("kvs.session.time_to_respond")
}

func (cfg *Config) GetSessionLimit() int {
	return cfg.viper.GetInt("kvs.session.day_session_limit")
}

func (cfg *Config) IsTracingEnabled() bool {
	return cfg.viper.GetBool("tracing.switch_on")
}

func (cfg *Config) GetOtelEndpoint() string {
	return cfg.viper.GetString("tracing.jaeger")
}

func (cfg *Config) GetPublisherInterval() time.Duration {
	return cfg.viper.GetDuration("kvs.scheduler.publisher.interval")
}

func (cfg *Config) GetFlusherInterval() time.Duration {
	return cfg.viper.GetDuration("kvs.scheduler.flusher.interval")
}

func (cfg *Config) ServiceName() string {
	return cfg.GetServiceName()
}

func (cfg *Config) ServiceVersion() string {
	return cfg.GetServiceVersion()
}

func (cfg *Config) LogLevel() string {
	return cfg.GetLogLevel()
}

func (cfg *Config) LogAddSource() bool {
	return cfg.GetLogAddSource()
}

func (cfg *Config) LogFormat() string {
	return cfg.GetLogFormat()
}

func (cfg *Config) TracingEnabled() bool {
	return cfg.IsTracingEnabled()
}

func (cfg *Config) TracingEndpoint() string {
	return cfg.GetOtelEndpoint()
}

func (cfg *Config) ShutdownTimeout() time.Duration {
	timeout := cfg.viper.GetDuration("kvs.shutdown_timeout")
	if timeout == 0 {
		return 5 * time.Second
	}

	return timeout
}
