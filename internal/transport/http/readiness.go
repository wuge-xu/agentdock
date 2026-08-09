package httptransport

import (
	"context"
	"net/http"
	"time"
)

type DatabasePinger interface {
	Ping(context.Context) error
}

type ReadinessHandler struct {
	database DatabasePinger
	timeout  time.Duration
}

func NewReadinessHandler(
	database DatabasePinger,
	timeout time.Duration,
) *ReadinessHandler {
	return &ReadinessHandler{
		database: database,
		timeout:  timeout,
	}
}

func (handler *ReadinessHandler) Ready(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if handler.database == nil {
		writeError(
			writer,
			http.StatusServiceUnavailable,
			"database_unavailable",
			"database unavailable",
			RequestIDFromContext(request.Context()),
		)
		return
	}

	contextWithTimeout, cancel := context.WithTimeout(
		request.Context(),
		handler.timeout,
	)
	defer cancel()

	if err := handler.database.Ping(
		contextWithTimeout,
	); err != nil {
		writeError(
			writer,
			http.StatusServiceUnavailable,
			"database_unavailable",
			"database unavailable",
			RequestIDFromContext(request.Context()),
		)
		return
	}

	writeJSON(
		writer,
		http.StatusOK,
		healthResponse{
			Status: "ready",
		},
	)
}
