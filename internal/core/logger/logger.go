// Package logger создаёт логгер приложения.
package logger

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config - настройки логгера. Читаются из переменных окружения
type Config struct {
	Level string `env:"LOGGER_LEVEL" envDefault:"info"`
	JSON  bool   `env:"LOGGER_JSON" envDefault:"false"`
}

// NewConfig читает настройки из переменных окружения
func NewConfig() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("process env: %w", err)
	}
	return cfg, nil
}

// New создаёт логгер по настройкам cfg
func New(cfg Config) (*zap.Logger, error) {
	lvl, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", cfg.Level, err)
	}
	zapCfg := zap.NewDevelopmentConfig()
	if cfg.JSON {
		zapCfg = zap.NewProductionConfig()
	}
	zapCfg.Level = zap.NewAtomicLevelAt(lvl)
	return zapCfg.Build()
}
