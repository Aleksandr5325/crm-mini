// Package postgres отвечает за подключение к PostgreSQL
package postgres

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config — настройки подключения к PostgreSQL из переменных окружения.
type Config struct {
	Host     string `env:"POSTGRES_HOST,required,notEmpty"`
	Port     string `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER,required,notEmpty"`
	Password string `env:"POSTGRES_PASSWORD,required,notEmpty"`
	Database string `env:"POSTGRES_DB,required,notEmpty"`
}

// NewConfig читает настройки подключения из переменных окружения.
func NewConfig() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("process env: %w", err)
	}
	return cfg, nil
}
