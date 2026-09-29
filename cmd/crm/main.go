package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/Aleksandr5325/crm-mini/internal/core/logger"
	"github.com/Aleksandr5325/crm-mini/internal/core/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logCfg, err := logger.NewConfig()
	if err != nil {
		return fmt.Errorf("logger config: %w", err)
	}

	log, err := logger.New(logCfg)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer log.Sync()

	pgCfg, err := postgres.NewConfig()
	if err != nil {
		return fmt.Errorf("postgres config: %w", err)
	}

	pool, err := postgres.NewPool(ctx, pgCfg)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	log.Info("connected to postgres",
		zap.String("host", pgCfg.Host),
		zap.String("database", pgCfg.Database),
	)

	<-ctx.Done()
	log.Info("shutting down")

	return nil
}
