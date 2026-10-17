package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/Aleksandr5325/crm-mini/internal/core/httpserver"
	"github.com/Aleksandr5325/crm-mini/internal/core/logger"
	"github.com/Aleksandr5325/crm-mini/internal/core/middleware"
	"github.com/Aleksandr5325/crm-mini/internal/core/postgres"
	"github.com/Aleksandr5325/crm-mini/internal/features/employees"
	"github.com/Aleksandr5325/crm-mini/internal/features/health"
	"github.com/Aleksandr5325/crm-mini/internal/features/leads"
	"github.com/Aleksandr5325/crm-mini/internal/features/reports"
	"github.com/Aleksandr5325/crm-mini/internal/features/sources"
	"github.com/Aleksandr5325/crm-mini/internal/web"
)

func main() {
	time.Local = time.UTC
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

	httpCfg, err := httpserver.NewConfig()
	if err != nil {
		return fmt.Errorf("http server config: %w", err)
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

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/health", health.NewHandler(pool))

	sourcesRepo := sources.NewRepository(pool)
	sourcesService := sources.NewService(sourcesRepo)
	sources.NewHandler(sourcesService).Register(mux)

	employeesRepo := employees.NewRepository(pool)
	employeesService := employees.NewService(employeesRepo)
	employees.NewHandler(employeesService).Register(mux)

	leadsRepo := leads.NewRepository(pool)
	leadsService := leads.NewService(leadsRepo)
	leads.NewHandler(leadsService).Register(mux)

	reportsRepo := reports.NewRepository(pool)
	reportsService := reports.NewService(reportsRepo)
	reports.NewHandler(reportsService).Register(mux)

	web.Register(mux)

	var handler http.Handler = mux
	handler = middleware.Recover(handler)
	handler = middleware.CORS(httpCfg.CORSAllowedOrigins, handler)
	handler = middleware.AccessLog(handler)
	handler = middleware.RequestID(log, handler)

	if err := httpserver.Run(ctx, httpCfg, handler, log); err != nil {
		return fmt.Errorf("http server: %w", err)
	}

	return nil
}
