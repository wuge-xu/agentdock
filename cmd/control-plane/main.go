package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/wuge-xu/agentdock/internal/application"
	"github.com/wuge-xu/agentdock/internal/config"
	"github.com/wuge-xu/agentdock/internal/platform/logging"
	"github.com/wuge-xu/agentdock/internal/platform/postgres"
	httptransport "github.com/wuge-xu/agentdock/internal/transport/http"
)

const version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"configuration error: %v\n",
			err,
		)
		os.Exit(1)
	}

	logger := logging.New(
		os.Stdout,
		logging.Config{
			Level:   cfg.Log.Level,
			Service: logging.DefaultService,
			Version: version,
		},
	)

	signalContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := run(
		signalContext,
		cfg,
		logger,
	); err != nil {
		logger.Error(
			"control plane stopped with error",
			slog.String(
				"error",
				err.Error(),
			),
		)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
) error {
	databasePool, err := postgres.NewPool(
		ctx,
		cfg.Database,
	)
	if err != nil {
		return fmt.Errorf(
			"initialize PostgreSQL pool: %w",
			err,
		)
	}
	defer databasePool.Close()

	taskRepository := postgres.NewTaskRepository(
		databasePool,
	)

	taskService := application.NewTaskService(
		taskRepository,
		nil,
		nil,
	)

	taskHandler := httptransport.NewTaskHandler(
		taskService,
	)

	httpLogger := logging.Component(
		logger,
		"http",
	)

	router := httptransport.NewRouter(
		httpLogger,
		databasePool,
		cfg.Database.ConnectTimeout,
		taskHandler,
	)

	server := httptransport.NewServer(
		cfg.HTTP,
		router,
	)

	serverErrors := make(
		chan error,
		1,
	)

	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	logger.Info(
		"control plane started",
		slog.String(
			"address",
			server.Addr,
		),
	)

	select {
	case <-ctx.Done():
		logger.Info(
			"shutdown signal received",
		)

	case err := <-serverErrors:
		if errors.Is(
			err,
			http.ErrServerClosed,
		) {
			return nil
		}

		return fmt.Errorf(
			"serve HTTP: %w",
			err,
		)
	}

	shutdownContext, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(
		shutdownContext,
	); err != nil {
		return fmt.Errorf(
			"shutdown HTTP server: %w",
			err,
		)
	}

	select {
	case err := <-serverErrors:
		if err != nil &&
			!errors.Is(
				err,
				http.ErrServerClosed,
			) {
			return fmt.Errorf(
				"HTTP server stopped unexpectedly: %w",
				err,
			)
		}

	case <-shutdownContext.Done():
		return fmt.Errorf(
			"wait for HTTP server shutdown: %w",
			shutdownContext.Err(),
		)
	}

	logger.Info(
		"control plane stopped",
	)

	return nil
}
