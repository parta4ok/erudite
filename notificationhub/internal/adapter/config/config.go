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

func (cfg *Config) GetMailSender() string {
	return cfg.viper.GetString("notificationhub.notifiers.email.sender")
}

func (cfg *Config) GetMailSenderSMTP() string {
	return cfg.viper.GetString("notificationhub.notifiers.email.smtp")
}

func (cfg *Config) GetMailSenderPort() string {
	return cfg.viper.GetString("notificationhub.notifiers.email.port")
}

func (cfg *Config) GetLogLevel() string {
	return cfg.viper.GetString("notificationhub.logging.level")
}

func (cfg *Config) GetLogFormat() string {
	return cfg.viper.GetString("notificationhub.logging.format")
}

func (cfg *Config) GetLogAddSource() bool {
	return cfg.viper.GetBool("notificationhub.logging.add_source")
}

func (cfg *Config) GetServiceName() string {
	return cfg.viper.GetString("notificationhub.logging.service_name")
}

func (cfg *Config) GetServiceVersion() string {
	return cfg.viper.GetString("notificationhub.logging.service_version")
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

func (cfg *Config) GetTracingType() string {
	return cfg.viper.GetString("tracing.system")
}

func (cfg *Config) TracingSystemName() string {
	return cfg.viper.GetString("tracing.servicename")
}

func (cfg *Config) IsTracingEnabled() bool {
	return cfg.viper.GetBool("tracing.enabled")
}

func (cfg *Config) GetTracingInfraURL(tracingSystemName string) string {
	return cfg.viper.GetString(fmt.Sprintf("%s.address", tracingSystemName))
}

func (cfg *Config) GetOtelEndpoint() string {
	return cfg.viper.GetString("otel.endpoint")
}
