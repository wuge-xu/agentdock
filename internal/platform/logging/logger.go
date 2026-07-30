package logging

import (
	"io"
	"log/slog"
)

const DefaultService = "agentdock-control-plane"

type Config struct {
	Level   slog.Level
	Service string
	Version string
}

func New(output io.Writer, cfg Config) *slog.Logger {
	service := cfg.Service
	if service == "" {
		service = DefaultService
	}

	handler := slog.NewJSONHandler(
		output,
		&slog.HandlerOptions{
			Level: cfg.Level,
		},
	)

	logger := slog.New(handler).With(
		slog.String("service", service),
	)

	if cfg.Version != "" {
		logger = logger.With(
			slog.String("version", cfg.Version),
		)
	}

	return logger
}

func Component(logger *slog.Logger, component string) *slog.Logger {
	if component == "" {
		return logger
	}

	return logger.With(
		slog.String("component", component),
	)
}
