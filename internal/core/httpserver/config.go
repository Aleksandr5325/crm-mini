// Package httpserver отвечает за запуск сервера
package httpserver

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config - настройки запуска сервера из переменных окружения.
type Config struct {
	Addr               string        `env:"HTTP_ADDR" envDefault:":8080"`
	ShutdownTimeout    time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"5s"`
	CORSAllowedOrigins []string      `env:"HTTP_CORS_ALLOWED_ORIGINS" envSeparator:","`
}

// NewConfig читает настройки запуска из переменных окружения.
func NewConfig() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("process env: %w", err)
	}
	return cfg, nil
}
