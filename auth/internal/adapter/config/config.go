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

func (cfg *Config) GetPrivatePort() string {
	return cfg.viper.GetString("auth.grpc.private.port")
}

func (cfg *Config) GetLogLevel() string {
	return cfg.viper.GetString("auth.logging.level")
}

func (cfg *Config) GetLogFormat() string {
	return cfg.viper.GetString("auth.logging.format")
}

func (cfg *Config) GetLogAddSource() bool {
	return cfg.viper.GetBool("auth.logging.add_source")
}

func (cfg *Config) GetServiceName() string {
	return cfg.viper.GetString("auth.logging.service_name")
}

func (cfg *Config) GetServiceVersion() string {
	return cfg.viper.GetString("auth.logging.service_version")
}

func (cfg *Config) GetServiceStorageType() string {
	return cfg.viper.GetString("auth.storage.type")
}

func (cfg *Config) GetStorageConnStr(storageType string) string {
	return cfg.viper.GetString(fmt.Sprintf("%s.connection", storageType))
}

func (cfg *Config) GetPrivateTimeout() time.Duration {
	timeoutStr := cfg.viper.GetString("auth.grpc.private.timeout")
	if timeoutStr == "" {
		return 30 * time.Second
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 30 * time.Second
	}

	return timeout
}

func (cfg *Config) GetJWTSecret() []byte {
    secret := cfg.viper.GetString("jwt.secret")
    return []byte(secret)
}

func (cfg *Config) GetJWTAudience() []string {
    return cfg.viper.GetStringSlice("jwt.aud")
}

func (cfg *Config) GetJWTIssuer() string {
    return cfg.viper.GetString("jwt.iss")
}

func (cfg *Config) GetJWTTTL() time.Duration {
    ttlStr := cfg.viper.GetString("jwt.ttl")
    if ttlStr == "" {
        return time.Hour
    }
    ttl, err := time.ParseDuration(ttlStr)
    if err != nil {
        return time.Hour
    }
    return ttl
}
