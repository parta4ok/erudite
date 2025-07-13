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

func (cfg *Config) GetAuthConn() string {
	return cfg.viper.GetString("auth_service.address")
}
