package config

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	sslModeDisable = "disable"
	sslModeRequire = "require"
)

type (
	Config struct {
		Host       string `mapstructure:"HOST"`
		Port       string `mapstructure:"PORT"`
		DBHost     string `mapstructure:"DB_HOST"`
		DBPort     string `mapstructure:"DB_PORT"`
		DBUser     string `mapstructure:"DB_USER"`
		DBPassword string `mapstructure:"DB_PASSWORD"`
		DBName     string `mapstructure:"DB_NAME"`
		DBSSLMode  string `mapstructure:"DB_SSL_MODE"`
	}
)

func NewConfig() (*Config, error) {
	viper.SetEnvPrefix("BOOKMARKER")

	viper.SetDefault("HOST", "0.0.0.0")
	viper.SetDefault("PORT", "1323")
	viper.SetDefault("DB_HOST", "0.0.0.0")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "user")
	viper.SetDefault("DB_PASSWORD", "password")
	viper.SetDefault("DB_NAME", "db")
	viper.SetDefault("DB_SSL_MODE", sslModeDisable)

	envs := []string{"HOST", "PORT", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSL_MODE"}
	for _, key := range envs {
		if err := viper.BindEnv(key); err != nil {
			return nil, err
		}
	}

	cfg := Config{}
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := validate(&cfg); err != nil {
		return nil, errors.Wrap(err, "config validation failed")
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	validSSLValues := []string{sslModeDisable, sslModeRequire}
	for _, validValue := range validSSLValues {
		if cfg.DBSSLMode == validValue {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("DB SSL mode is invalid: %s", cfg.DBSSLMode))
}
